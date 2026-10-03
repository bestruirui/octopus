package op

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

var (
	channelCache      = cache.New[int, model.Channel](16)      // 渠道配置的进程内副本。
	channelKeyCache   = cache.New[int, model.ChannelKey](16)   // 渠道凭据的进程内副本。
	channelModelCache = cache.New[int, model.ChannelModel](16) // 渠道模型的进程内副本。
	channelGrantCache = cache.New[int, model.ChannelGrant](16) // 渠道授权的进程内副本。
)

// 已定义的全部协议位, 用于校验提交的协议掩码。
const definedProtocols = model.ProtocolOpenAIChatCompletion | model.ProtocolOpenAIResponse | model.ProtocolAnthropicMessage

// ChannelDetailGet 返回指定渠道的完整配置, 供编辑表单读取。
func ChannelDetailGet(id int) (model.ChannelDetail, error) {
	channel, ok := channelCache.Get(id)
	if !ok {
		return model.ChannelDetail{}, fmt.Errorf("channel not found")
	}
	return channelDetail(channel), nil
}

// ChannelStatsList 返回全部渠道及其模型的累计统计, 自带名称与启停状态, 同时充当列表页的渠道列表。
// 不带整份配置: 路径, 代理与凭据明文只在编辑时用得上, 由 ChannelDetailGet 按主键单独给出。
func ChannelStatsList() []model.ChannelStats {
	modelsByChannel := make(map[int][]model.ChannelModelStats, channelCache.Len())
	for _, channelModel := range channelModelCache.GetAll() {
		modelsByChannel[channelModel.ChannelID] = append(modelsByChannel[channelModel.ChannelID], model.ChannelModelStats{
			ModelID:      channelModel.ID,
			ModelName:    channelModel.Name,
			StatsMetrics: channelModel.StatsMetrics,
		})
	}
	stats := make([]model.ChannelStats, 0, channelCache.Len())
	for _, channel := range channelCache.GetAll() {
		models := modelsByChannel[channel.ID]
		if models == nil {
			models = []model.ChannelModelStats{}
		}
		stats = append(stats, model.ChannelStats{
			ChannelID:    channel.ID,
			ChannelName:  channel.Name,
			Enabled:      channel.Enabled,
			Models:       models,
			StatsMetrics: channel.StatsMetrics,
		})
	}
	return stats
}

// ChannelCreate 创建渠道及其凭据, 模型与授权, 返回创建后的完整配置。
// 三者在同一事务内落库: 授权按名称引用两侧, 待凭据与模型拿到主键后由 syncChannelGrants 解析,
// 由此建一个带授权的渠道只需一趟请求。
func ChannelCreate(detail *model.ChannelDetail, ctx context.Context) (*model.ChannelDetail, error) {
	if err := normalizeChannelDetail(detail); err != nil {
		return nil, err
	}

	channel := model.Channel{ChannelConfig: detail.ChannelConfig}
	if err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&channel).Error; err != nil {
			return fmt.Errorf("failed to create channel: %w", err)
		}
		return syncChannelChildren(tx, channel.ID, detail)
	}); err != nil {
		return nil, err
	}

	channelCache.Set(channel.ID, channel)
	// 凭据, 模型与授权的主键都在事务内分配, 此刻只在库里; 重载子表缓存以让授权候选与转发都能查到。
	if err := reloadChannelChildren(ctx, channel.ID); err != nil {
		return nil, err
	}
	created := channelDetail(channel)
	return &created, nil
}

// ChannelUpdate 按提交的完整配置整体替换渠道及其凭据, 模型与授权, 返回刷新后的配置。
// 提交即全量而非按字段比对增量: 渠道是人工编辑的十几个字段, 表单本就一次给出完整配置,
// 未列出的凭据与模型会被删除并级联删除其授权。
func ChannelUpdate(detail *model.ChannelDetail, ctx context.Context) (*model.ChannelDetail, error) {
	if _, ok := channelCache.Get(detail.ID); !ok {
		return nil, fmt.Errorf("channel not found")
	}
	if err := normalizeChannelDetail(detail); err != nil {
		return nil, err
	}

	if err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 逐列点名而不整行覆盖: 全量提交下 enabled 置假与被清空的可选字段都必须落库, 按零值跳过会写不进去;
		// 而统计列由转发累加, 不在提交范围内, 整行覆盖会把它抹回提交时的旧值。
		if err := tx.Model(&model.Channel{}).Where("id = ?", detail.ID).
			Select("name", "dialect", "enabled", "base_url",
				"openai_chat_completion_path", "openai_response_path", "anthropic_message_path",
				"proxy", "channel_proxy", "custom_header", "param_override", "match_regex").
			Updates(&model.Channel{ChannelConfig: detail.ChannelConfig}).Error; err != nil {
			return fmt.Errorf("failed to update channel: %w", err)
		}
		return syncChannelChildren(tx, detail.ID, detail)
	}); err != nil {
		return nil, err
	}

	// 缓存条目由提交的配置重建, 统计从原条目搬过来: 它含本轮尚未落库的累加, 比库内的行更新。
	channelStatsNeedUpdateLock.Lock()
	channel := model.Channel{ID: detail.ID, ChannelConfig: detail.ChannelConfig}
	if cached, ok := channelCache.Get(detail.ID); ok {
		channel.StatsMetrics = cached.StatsMetrics
	}
	channelCache.Set(detail.ID, channel)
	channelStatsNeedUpdateLock.Unlock()

	// 凭据, 模型与授权的增删都会改变可选路由集合, 重载该渠道的三类缓存并刷新分组。
	if err := reloadChannelChildren(ctx, detail.ID); err != nil {
		return nil, err
	}
	if err := groupRefreshCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh groups: %w", err)
	}
	updated := channelDetail(channel)
	return &updated, nil
}

// normalizeChannelConfig 补齐提交配置中的默认值并校验协议路径。
// 路径留空会与地址拼成错误的上游地址, 故一律回退到协议默认路径; Header 恒为数组, 免得落库后读出 null。
// 全部字段在此去空白: 落库后的配置被读侧无条件信任, 空白与空串都不会再传到转发链路。
func normalizeChannelConfig(config model.ChannelConfig) (model.ChannelConfig, error) {
	config.Name = strings.TrimSpace(config.Name)
	if config.Name == "" {
		return config, fmt.Errorf("channel name is required")
	}
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	if config.BaseURL == "" {
		return config, fmt.Errorf("channel base url is required")
	}
	if config.Dialect == "" {
		config.Dialect = model.DialectGeneric
	}
	var err error
	if config.OpenAIChatCompletionPath, err = normalizedPath(config.OpenAIChatCompletionPath, "/v1/chat/completions"); err != nil {
		return config, err
	}
	if config.OpenAIResponsePath, err = normalizedPath(config.OpenAIResponsePath, "/v1/responses"); err != nil {
		return config, err
	}
	if config.AnthropicMessagePath, err = normalizedPath(config.AnthropicMessagePath, "/v1/messages"); err != nil {
		return config, err
	}
	if config.CustomHeader == nil {
		config.CustomHeader = []model.CustomHeader{}
	}

	config.ChannelProxy = strings.TrimSpace(config.ChannelProxy)
	config.ParamOverride = strings.TrimSpace(config.ParamOverride)
	config.MatchRegex = strings.TrimSpace(config.MatchRegex)
	if err := validateParamOverride(config.ParamOverride); err != nil {
		return config, err
	}
	for _, header := range config.CustomHeader {
		if err := validateCustomHeader(header); err != nil {
			return config, err
		}
	}
	return config, nil
}

// validateCustomHeader 校验单条自定义 Header; op 为空按 set 处理, 与历史数据兼容。
// delete 只需头名, 值不参与也不校验; set 与 rename/copy 还需要值(rename/copy 的值是目标头名)。
func validateCustomHeader(header model.CustomHeader) error {
	if err := model.ValidateOverrideTemplate(header.Condition); err != nil {
		return fmt.Errorf("custom header %q condition: %w", header.HeaderKey, err)
	}
	switch header.Op {
	case "", model.OverrideOpSet, model.OverrideOpDelete, model.OverrideOpRename, model.OverrideOpCopy:
	default:
		return fmt.Errorf("custom header %q: unknown op %q", header.HeaderKey, header.Op)
	}
	if strings.TrimSpace(header.HeaderKey) == "" {
		return errors.New("custom header: header_key is required")
	}
	if header.Op == model.OverrideOpDelete {
		return nil // 值不参与。
	}
	if err := model.ValidateOverrideTemplate(header.HeaderValue); err != nil {
		return fmt.Errorf("custom header %q value: %w", header.HeaderKey, err)
	}
	if strings.TrimSpace(header.HeaderValue) == "" {
		return fmt.Errorf("custom header %q: header_value is required", header.HeaderKey)
	}
	return nil
}

// validateParamOverride 校验参数覆盖配置: {...} 为旧的平铺对象, [...] 为操作数组。
// 非法配置在保存时即被拒绝, 避免留到每次转发时报错; 旧格式仅校验 JSON 合法性, 键的约束沿用转发时的静默跳过。
func validateParamOverride(config string) error {
	if config == "" {
		return nil
	}
	switch config[0] {
	case '{':
		var overrides map[string]json.RawMessage
		if err := json.Unmarshal([]byte(config), &overrides); err != nil {
			return fmt.Errorf("invalid param override: %w", err)
		}
		return nil
	case '[':
		var ops []model.OverrideOperation
		if err := json.Unmarshal([]byte(config), &ops); err != nil {
			return fmt.Errorf("invalid param override: %w", err)
		}
		for i, op := range ops {
			if err := validateOverrideOperation(i+1, op); err != nil {
				return err
			}
		}
		return nil
	default:
		return errors.New("param override must be a JSON object or array")
	}
}

// validateOverrideOperation 校验单条覆盖操作的完整性; index 从 1 起, 用于报错定位。
func validateOverrideOperation(index int, op model.OverrideOperation) error {
	switch op.Op {
	case model.OverrideOpSet, model.OverrideOpSetIfAbsent:
		if op.Path == "" {
			return fmt.Errorf("param override %d: path is required", index)
		}
		if strings.TrimSpace(op.Value) == "" {
			return fmt.Errorf("param override %d: value is required", index)
		}
	case model.OverrideOpDelete:
		if op.Path == "" {
			return fmt.Errorf("param override %d: path is required", index)
		}
	default:
		return fmt.Errorf("param override %d: unknown op %q", index, op.Op)
	}
	if op.Path == "model" || op.Path == "stream" {
		return fmt.Errorf("param override %d: path %q is managed by relay", index, op.Path)
	}
	if err := model.ValidateOverrideTemplate(op.Condition); err != nil {
		return fmt.Errorf("param override %d condition: %w", index, err)
	}
	if err := model.ValidateOverrideTemplate(op.Value); err != nil {
		return fmt.Errorf("param override %d value: %w", index, err)
	}
	return nil
}

// normalizeChannelDetail 规范化整份提交配置; 渠道自身的字段交由 normalizeChannelConfig 处理。
// 凭据, 模型与授权的名称在此去空白并校验非空: 名称是三者的匹配与引用依据, 集中在入口清理后,
// 下游三个同步函数拿到的即是干净数据, 无需各自再 trim 一遍。
func normalizeChannelDetail(detail *model.ChannelDetail) error {
	config, err := normalizeChannelConfig(detail.ChannelConfig)
	if err != nil {
		return err
	}
	detail.ChannelConfig = config

	for i := range detail.Keys {
		detail.Keys[i].Name = strings.TrimSpace(detail.Keys[i].Name)
		if detail.Keys[i].Name == "" {
			return fmt.Errorf("channel key name is required")
		}
		// 凭据两端的空白会被原样拼进认证 Header, 一并去掉。
		detail.Keys[i].Key = strings.TrimSpace(detail.Keys[i].Key)
	}
	for i := range detail.Models {
		detail.Models[i] = strings.TrimSpace(detail.Models[i])
		if detail.Models[i] == "" {
			return fmt.Errorf("channel model name is required")
		}
	}
	for i := range detail.Grants {
		detail.Grants[i].ModelName = strings.TrimSpace(detail.Grants[i].ModelName)
		detail.Grants[i].KeyName = strings.TrimSpace(detail.Grants[i].KeyName)
	}
	return nil
}

// syncChannelChildren 按提交的完整配置整体替换渠道下的凭据, 模型与授权。
// 凭据与模型必须先落库: 授权引用两者的主键, 新增的两者在同一事务内才拿得到。
func syncChannelChildren(tx *gorm.DB, channelID int, detail *model.ChannelDetail) error {
	if err := syncChannelKeys(tx, channelID, detail.Keys); err != nil {
		return err
	}
	if err := syncChannelModels(tx, channelID, detail.Models); err != nil {
		return err
	}
	return syncChannelGrants(tx, channelID, detail.Grants)
}

// ChannelEnabled 更新渠道启用状态。
func ChannelEnabled(id int, enabled bool, ctx context.Context) error {
	channel, ok := channelCache.Get(id)
	if !ok {
		return fmt.Errorf("channel not found")
	}
	if err := db.GetDB().WithContext(ctx).Model(&model.Channel{}).Where("id = ?", id).Update("enabled", enabled).Error; err != nil {
		return err
	}
	channel.Enabled = enabled
	channelCache.Set(id, channel)
	return nil
}

// ChannelDel 删除渠道及其凭据, 模型与渠道授权, 关联分组项由数据库外键级联删除。
func ChannelDel(id int, ctx context.Context) error {
	if _, ok := channelCache.Get(id); !ok {
		return fmt.Errorf("channel not found")
	}
	grantIDs := channelGrantIDs(id)
	if err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(grantIDs) > 0 {
			if err := clearActiveItems(tx, grantIDs); err != nil {
				return err
			}
		}
		if err := tx.Delete(&model.Channel{}, id).Error; err != nil {
			return fmt.Errorf("failed to delete channel: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	channelStatsNeedUpdateLock.Lock()
	channelCache.Del(id)
	delete(channelStatsNeedUpdate, id)
	channelStatsNeedUpdateLock.Unlock()

	channelKeyStatsNeedUpdateLock.Lock()
	for _, channelKey := range channelKeyCache.GetAll() {
		if channelKey.ChannelID == id {
			channelKeyCache.Del(channelKey.ID)
			delete(channelKeyStatsNeedUpdate, channelKey.ID)
		}
	}
	channelKeyStatsNeedUpdateLock.Unlock()

	channelModelStatsNeedUpdateLock.Lock()
	for _, channelModel := range channelModelCache.GetAll() {
		if channelModel.ChannelID == id {
			channelModelCache.Del(channelModel.ID)
			delete(channelModelStatsNeedUpdate, channelModel.ID)
		}
	}
	channelModelStatsNeedUpdateLock.Unlock()

	channelGrantCache.Del(grantIDs...)
	if err := groupRefreshCache(ctx); err != nil {
		return fmt.Errorf("failed to refresh groups: %w", err)
	}
	return nil
}

// ChannelGet 返回指定渠道的缓存副本, 供转发按地址, 路径与代理构造上游请求。
// 不补齐凭据, 模型与授权: 转发所需的授权由 ChannelGrantGet 按主键单独取, 那里已连带给出两侧。
func ChannelGet(id int) (model.Channel, error) {
	channel, ok := channelCache.Get(id)
	if !ok {
		return model.Channel{}, fmt.Errorf("channel not found")
	}
	return channel, nil
}

// ChannelGrantGet 返回可用于转发的渠道授权, 并补齐其模型与凭据。
// 凭据被停用, 以及模型, 凭据缺失时一律返回错误, 使调用方拿到的授权必然可直接转发, 无需再逐项检查。
// 授权本身没有停用状态: 不再授权就删掉该组合, 无需保留一行停用记录。
func ChannelGrantGet(id int) (model.ChannelGrant, error) {
	grant, ok := channelGrantCache.Get(id)
	if !ok {
		return model.ChannelGrant{}, fmt.Errorf("channel grant not found")
	}
	channelModel, ok := channelModelCache.Get(grant.ChannelModelID)
	if !ok {
		return model.ChannelGrant{}, fmt.Errorf("channel model %d not found", grant.ChannelModelID)
	}
	channelKey, ok := channelKeyCache.Get(grant.ChannelKeyID)
	if !ok {
		return model.ChannelGrant{}, fmt.Errorf("channel key %d not found", grant.ChannelKeyID)
	}
	if !channelKey.Enabled {
		return model.ChannelGrant{}, fmt.Errorf("channel key %d is disabled", channelKey.ID)
	}
	grant.ChannelModel = &channelModel
	grant.ChannelKey = &channelKey
	return grant, nil
}

// ChannelGrantCandidates 返回全部渠道授权及其展示字段, 供分组页选取成员。
// 可用性与 GroupList 补齐成员时同一口径: 渠道与凭据均启用即可用, 由此候选与已选成员不会各判一套。
// 两侧任一缺失的授权直接跳过: 它无法转发, 也无从展示名称。
func ChannelGrantCandidates() []model.ChannelGrantCandidate {
	candidates := make([]model.ChannelGrantCandidate, 0, channelGrantCache.Len())
	for _, grant := range channelGrantCache.GetAll() {
		channelModel, modelOK := channelModelCache.Get(grant.ChannelModelID)
		channelKey, keyOK := channelKeyCache.Get(grant.ChannelKeyID)
		if !modelOK || !keyOK {
			continue
		}
		channel, channelOK := channelCache.Get(channelModel.ChannelID)
		if !channelOK {
			continue
		}
		candidates = append(candidates, model.ChannelGrantCandidate{
			ID:          grant.ID,
			ChannelID:   channel.ID,
			ChannelName: channel.Name,
			ModelName:   channelModel.Name,
			KeyName:     channelKey.Name,
			Protocols:   grant.Protocols,
			Available:   channel.Enabled && channelKey.Enabled,
		})
	}
	// 按渠道, 模型, 凭据三级定序: 分组页按这三级组织候选且不提供排序开关, 顺序须由此处定稿。
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ChannelID != candidates[j].ChannelID {
			return candidates[i].ChannelID < candidates[j].ChannelID
		}
		if candidates[i].ModelName != candidates[j].ModelName {
			return candidates[i].ModelName < candidates[j].ModelName
		}
		return candidates[i].KeyName < candidates[j].KeyName
	})
	return candidates
}

// channelRefreshCache 从数据库刷新渠道, 凭据, 模型与渠道授权缓存。
func channelRefreshCache(ctx context.Context) error {
	conn := db.GetDB().WithContext(ctx)
	channels := []model.Channel{}
	if err := conn.Find(&channels).Error; err != nil {
		log.Warnf("failed to get channels: %v", err)
		return err
	}
	channelKeys := []model.ChannelKey{}
	if err := conn.Find(&channelKeys).Error; err != nil {
		return err
	}
	channelModels := []model.ChannelModel{}
	if err := conn.Find(&channelModels).Error; err != nil {
		return err
	}
	channelGrants := []model.ChannelGrant{}
	if err := conn.Find(&channelGrants).Error; err != nil {
		return err
	}

	channelCache.Clear()
	channelKeyCache.Clear()
	channelModelCache.Clear()
	channelGrantCache.Clear()
	for _, channel := range channels {
		channelCache.Set(channel.ID, channel)
	}
	for _, channelKey := range channelKeys {
		channelKeyCache.Set(channelKey.ID, channelKey)
	}
	for _, channelModel := range channelModels {
		channelModelCache.Set(channelModel.ID, channelModel)
	}
	for _, grant := range channelGrants {
		channelGrantCache.Set(grant.ID, grant)
	}
	return nil
}

// reloadChannelChildren 重新加载单个渠道的凭据, 模型与渠道授权缓存。
// 存活目标保留缓存中尚未落库的统计, 避免刷新丢失本轮累加。
func reloadChannelChildren(ctx context.Context, channelID int) error {
	conn := db.GetDB().WithContext(ctx)
	channelKeys := []model.ChannelKey{}
	if err := conn.Where("channel_id = ?", channelID).Find(&channelKeys).Error; err != nil {
		return fmt.Errorf("failed to load channel keys: %w", err)
	}
	channelModels := []model.ChannelModel{}
	if err := conn.Where("channel_id = ?", channelID).Find(&channelModels).Error; err != nil {
		return fmt.Errorf("failed to load channel models: %w", err)
	}
	modelIDs := make([]int, 0, len(channelModels))
	for _, channelModel := range channelModels {
		modelIDs = append(modelIDs, channelModel.ID)
	}
	grants := []model.ChannelGrant{}
	if len(modelIDs) > 0 {
		if err := conn.Where("channel_model_id IN ?", modelIDs).Find(&grants).Error; err != nil {
			return fmt.Errorf("failed to load channel grants: %w", err)
		}
	}

	// 先按库内行覆盖再清理消失的行: 配置以库内为准, 统计以缓存为准。
	// 存活行的缓存值含尚未落库的累加, 比库内的行更新, 不能被库内的统计覆盖。
	channelKeyStatsNeedUpdateLock.Lock()
	liveKeys := make(map[int]struct{}, len(channelKeys))
	for _, channelKey := range channelKeys {
		liveKeys[channelKey.ID] = struct{}{}
		if cached, ok := channelKeyCache.Get(channelKey.ID); ok {
			channelKey.StatsMetrics = cached.StatsMetrics
		}
		channelKeyCache.Set(channelKey.ID, channelKey)
	}
	for _, cached := range channelKeyCache.GetAll() {
		if _, live := liveKeys[cached.ID]; cached.ChannelID == channelID && !live {
			channelKeyCache.Del(cached.ID)
		}
	}
	channelKeyStatsNeedUpdateLock.Unlock()

	channelModelStatsNeedUpdateLock.Lock()
	liveModels := make(map[int]struct{}, len(channelModels))
	for _, channelModel := range channelModels {
		liveModels[channelModel.ID] = struct{}{}
		if cached, ok := channelModelCache.Get(channelModel.ID); ok {
			channelModel.StatsMetrics = cached.StatsMetrics
		}
		channelModelCache.Set(channelModel.ID, channelModel)
	}
	for _, cached := range channelModelCache.GetAll() {
		if _, live := liveModels[cached.ID]; cached.ChannelID == channelID && !live {
			channelModelCache.Del(cached.ID)
		}
	}
	channelModelStatsNeedUpdateLock.Unlock()

	// 授权不带统计, 直接按库内行整体替换。
	for _, grantID := range channelGrantIDs(channelID) {
		channelGrantCache.Del(grantID)
	}
	for _, grant := range grants {
		channelGrantCache.Set(grant.ID, grant)
	}
	return nil
}

// channelDetail 把渠道缓存与其凭据, 模型和授权合并为编辑表单所需的完整配置。
// 授权按名称给出而不给主键: 名称在渠道内唯一, 提交时也按名称引用, 读写同一种寻址界面才无需翻译。
// 三个集合字段恒为数组, 空集合也给出, 免得各消费方各自兜底 null; Header 由写入侧保证已是数组。
// 三者均按名称定序: 缓存遍历顺序随机, 而编辑表单不提供排序开关, 顺序须由此处定稿。
func channelDetail(channel model.Channel) model.ChannelDetail {
	detail := model.ChannelDetail{ID: channel.ID, ChannelConfig: channel.ChannelConfig}

	detail.Keys = make([]model.ChannelKeyConfig, 0)
	keyNameByID := make(map[int]string)
	for _, channelKey := range channelKeyCache.GetAll() {
		if channelKey.ChannelID == channel.ID {
			detail.Keys = append(detail.Keys, channelKey.ChannelKeyConfig)
			keyNameByID[channelKey.ID] = channelKey.Name
		}
	}
	sort.Slice(detail.Keys, func(i, j int) bool { return detail.Keys[i].Name < detail.Keys[j].Name })

	detail.Models = make([]string, 0)
	modelNameByID := make(map[int]string)
	for _, channelModel := range channelModelCache.GetAll() {
		if channelModel.ChannelID == channel.ID {
			detail.Models = append(detail.Models, channelModel.Name)
			modelNameByID[channelModel.ID] = channelModel.Name
		}
	}
	sort.Strings(detail.Models)

	// 授权按模型主键归属本渠道, 两侧主键在此翻译成名称。
	grants := make([]model.ChannelGrantConfig, 0)
	for _, grant := range channelGrantCache.GetAll() {
		modelName, ok := modelNameByID[grant.ChannelModelID]
		if !ok {
			continue
		}
		keyName, ok := keyNameByID[grant.ChannelKeyID]
		if !ok {
			continue
		}
		grants = append(grants, model.ChannelGrantConfig{ModelName: modelName, KeyName: keyName, Protocols: grant.Protocols})
	}
	sort.Slice(grants, func(i, j int) bool {
		if grants[i].ModelName != grants[j].ModelName {
			return grants[i].ModelName < grants[j].ModelName
		}
		return grants[i].KeyName < grants[j].KeyName
	})
	detail.Grants = grants
	return detail
}

// channelGrantIDs 返回指定渠道下全部渠道授权的主键。
func channelGrantIDs(channelID int) []int {
	grantIDs := make([]int, 0)
	for _, grant := range channelGrantCache.GetAll() {
		if channelModel, ok := channelModelCache.Get(grant.ChannelModelID); ok && channelModel.ChannelID == channelID {
			grantIDs = append(grantIDs, grant.ID)
		}
	}
	return grantIDs
}

// syncChannelKeys 按提交的凭据集合新增, 更新与删除渠道凭据。
// 凭据在渠道内按名称唯一, 名称作为匹配依据; 删除凭据会级联删除其渠道授权。
func syncChannelKeys(tx *gorm.DB, channelID int, requested []model.ChannelKeyConfig) error {
	var existing []model.ChannelKey
	if err := tx.Where("channel_id = ?", channelID).Find(&existing).Error; err != nil {
		return fmt.Errorf("failed to load channel keys: %w", err)
	}
	existingByName := make(map[string]model.ChannelKey, len(existing))
	for _, channelKey := range existing {
		existingByName[channelKey.Name] = channelKey
	}
	for _, requestedKey := range requested {
		if current, ok := existingByName[requestedKey.Name]; ok {
			if current.Key != requestedKey.Key || current.Enabled != requestedKey.Enabled {
				if err := tx.Model(&model.ChannelKey{}).Where("id = ?", current.ID).
					Updates(map[string]any{"key": requestedKey.Key, "enabled": requestedKey.Enabled}).Error; err != nil {
					return fmt.Errorf("failed to update channel key: %w", err)
				}
			}
			delete(existingByName, requestedKey.Name)
			continue
		}
		newKey := model.ChannelKey{ChannelID: channelID, ChannelKeyConfig: requestedKey}
		if err := tx.Create(&newKey).Error; err != nil {
			return fmt.Errorf("failed to create channel key: %w", err)
		}
	}
	deletedKeyIDs := make([]int, 0, len(existingByName))
	for _, channelKey := range existingByName {
		deletedKeyIDs = append(deletedKeyIDs, channelKey.ID)
	}
	if len(deletedKeyIDs) == 0 {
		return nil
	}
	if err := clearActiveItems(tx, grantsOf(tx, "channel_key_id", deletedKeyIDs)); err != nil {
		return err
	}
	if err := tx.Delete(&model.ChannelKey{}, deletedKeyIDs).Error; err != nil {
		return fmt.Errorf("failed to delete channel keys: %w", err)
	}
	return nil
}

// syncChannelModels 按提交的模型名称集合新增与删除渠道模型。
// 模型在渠道内按名称唯一, 除名称外无可更新字段, 故提交侧直接给名称; 删除模型会级联删除其渠道授权。
func syncChannelModels(tx *gorm.DB, channelID int, requested []string) error {
	var existing []model.ChannelModel
	if err := tx.Where("channel_id = ?", channelID).Find(&existing).Error; err != nil {
		return fmt.Errorf("failed to load channel models: %w", err)
	}
	existingByName := make(map[string]model.ChannelModel, len(existing))
	for _, channelModel := range existing {
		existingByName[channelModel.Name] = channelModel
	}
	for _, requestedModel := range requested {
		if _, ok := existingByName[requestedModel]; ok {
			delete(existingByName, requestedModel)
			continue
		}
		if err := tx.Create(&model.ChannelModel{ChannelID: channelID, Name: requestedModel}).Error; err != nil {
			return fmt.Errorf("failed to create channel model: %w", err)
		}
	}
	deletedModelIDs := make([]int, 0, len(existingByName))
	for _, channelModel := range existingByName {
		deletedModelIDs = append(deletedModelIDs, channelModel.ID)
	}
	if len(deletedModelIDs) == 0 {
		return nil
	}
	if err := clearActiveItems(tx, grantsOf(tx, "channel_model_id", deletedModelIDs)); err != nil {
		return err
	}
	if err := tx.Delete(&model.ChannelModel{}, deletedModelIDs).Error; err != nil {
		return fmt.Errorf("failed to delete channel models: %w", err)
	}
	return nil
}

// syncChannelGrants 按提交的授权集合新增, 更新与删除渠道授权。
// 授权以 (模型, 凭据) 组合唯一, 该组合作为匹配依据; 提交方按名称引用, 名称在此解析为本渠道的主键。
// 凭据与模型已在本事务内先行同步, 故新增的两者在此都能查到, 一次提交即可完成建模型与授权。
func syncChannelGrants(tx *gorm.DB, channelID int, requested []model.ChannelGrantConfig) error {
	var channelModels []model.ChannelModel
	if err := tx.Where("channel_id = ?", channelID).Find(&channelModels).Error; err != nil {
		return fmt.Errorf("failed to load channel models: %w", err)
	}
	modelIDByName := make(map[string]int, len(channelModels))
	modelIDList := make([]int, 0, len(channelModels))
	for _, channelModel := range channelModels {
		modelIDByName[channelModel.Name] = channelModel.ID
		modelIDList = append(modelIDList, channelModel.ID)
	}
	var channelKeys []model.ChannelKey
	if err := tx.Where("channel_id = ?", channelID).Find(&channelKeys).Error; err != nil {
		return fmt.Errorf("failed to load channel keys: %w", err)
	}
	keyIDByName := make(map[string]int, len(channelKeys))
	for _, channelKey := range channelKeys {
		keyIDByName[channelKey.Name] = channelKey.ID
	}

	var existing []model.ChannelGrant
	if len(modelIDList) > 0 {
		if err := tx.Where("channel_model_id IN ?", modelIDList).Find(&existing).Error; err != nil {
			return fmt.Errorf("failed to load channel grants: %w", err)
		}
	}
	type grantKey struct {
		modelID int // 渠道模型主键。
		keyID   int // 渠道凭据主键。
	}
	existingByKey := make(map[grantKey]model.ChannelGrant, len(existing))
	for _, grant := range existing {
		existingByKey[grantKey{grant.ChannelModelID, grant.ChannelKeyID}] = grant
	}

	for _, requestedGrant := range requested {
		modelID, ok := modelIDByName[requestedGrant.ModelName]
		if !ok {
			return fmt.Errorf("channel model %q does not belong to channel %d", requestedGrant.ModelName, channelID)
		}
		keyID, ok := keyIDByName[requestedGrant.KeyName]
		if !ok {
			return fmt.Errorf("channel key %q does not belong to channel %d", requestedGrant.KeyName, channelID)
		}
		if requestedGrant.Protocols == 0 || requestedGrant.Protocols&^definedProtocols != 0 {
			return fmt.Errorf("channel grant protocols %d is empty or contains undefined bits", requestedGrant.Protocols)
		}
		key := grantKey{modelID, keyID}
		if current, ok := existingByKey[key]; ok {
			if current.Protocols != requestedGrant.Protocols {
				if err := tx.Model(&model.ChannelGrant{}).Where("id = ?", current.ID).
					Update("protocols", requestedGrant.Protocols).Error; err != nil {
					return fmt.Errorf("failed to update channel grant: %w", err)
				}
			}
			delete(existingByKey, key)
			continue
		}
		newGrant := model.ChannelGrant{
			ChannelModelID: modelID,
			ChannelKeyID:   keyID,
			Protocols:      requestedGrant.Protocols,
		}
		if err := tx.Create(&newGrant).Error; err != nil {
			return fmt.Errorf("failed to create channel grant: %w", err)
		}
	}

	deletedGrantIDs := make([]int, 0, len(existingByKey))
	for _, grant := range existingByKey {
		deletedGrantIDs = append(deletedGrantIDs, grant.ID)
	}
	if len(deletedGrantIDs) == 0 {
		return nil
	}
	if err := clearActiveItems(tx, deletedGrantIDs); err != nil {
		return err
	}
	if err := tx.Delete(&model.ChannelGrant{}, deletedGrantIDs).Error; err != nil {
		return fmt.Errorf("failed to delete channel grants: %w", err)
	}
	return nil
}

// normalizedPath 去掉协议路径两端空白, 留空时回退到默认路径, 并校验前导斜杠。
// 缺少前导斜杠会与地址拼成错误的上游地址, 在写入前拒绝。
func normalizedPath(value, fallback string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	if !strings.HasPrefix(value, "/") {
		return value, fmt.Errorf("channel path %q must start with /", value)
	}
	return value, nil
}

// clearActiveItems 清理引用待删除授权的分组当前项。
// grants 既可以是授权主键切片, 也可以是筛选授权主键的子查询, 由调用方按删除的是授权, 模型还是凭据给出。
func clearActiveItems(tx *gorm.DB, grants any) error {
	itemIDs := tx.Model(&model.GroupItem{}).Select("id").Where("channel_grant_id IN (?)", grants)
	if err := tx.Model(&model.Group{}).
		Where("active_item_id IN (?)", itemIDs).
		Update("active_item_id", 0).Error; err != nil {
		return fmt.Errorf("failed to clear active items: %w", err)
	}
	return nil
}

// grantsOf 返回筛选指定列命中某组主键的授权主键子查询, 供 clearActiveItems 级联定位。
func grantsOf(tx *gorm.DB, column string, ids []int) *gorm.DB {
	return tx.Model(&model.ChannelGrant{}).Select("id").Where(column+" IN ?", ids)
}

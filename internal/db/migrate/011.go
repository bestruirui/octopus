package migrate

import (
	"fmt"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

func init() {
	RegisterBeforeAutoMigration(Migration{
		Version: 11,
		Up:      migrateChannelGrants,
	})
}

// 原 channels.type 到协议位与路径的映射。
// gemini 原生协议已移除, 该类型渠道按 OpenAI Chat 接入, 地址需人工改为其 OpenAI 兼容端点。
// 路径为空表示沿用协议默认路径; volcengine 地址自带版本号, 单独给出不含版本号的路径。
// 全部旧类型的方言都是 generic: 它们在标准协议之上没有请求体或响应体差异, 仅地址与路径不同。
var legacyChannelTypes = map[string]struct {
	protocol     model.Protocol // 该类型对应的单个协议位。
	chatPath     string         // OpenAI Chat Completions 路径。
	responsePath string         // OpenAI Responses 路径。
	messagePath  string         // Anthropic Messages 路径。
}{
	"openai":           {protocol: model.ProtocolOpenAIChatCompletion},
	"openai_responses": {protocol: model.ProtocolOpenAIResponse},
	"anthropic":        {protocol: model.ProtocolAnthropicMessage},
	"gemini":           {protocol: model.ProtocolOpenAIChatCompletion},
	"volcengine": {
		protocol:     model.ProtocolOpenAIChatCompletion,
		chatPath:     "/chat/completions",
		responsePath: "/responses",
		messagePath:  "/messages",
	},
}

// migrateChannelGrants 把单地址单凭据单协议的渠道展开为渠道凭据与渠道授权结构。
// 每个渠道建一条默认凭据, 每个模型与该凭据建一条渠道授权并写入原类型对应的协议位,
// 随后把分组项由引用渠道模型改为引用渠道授权, 最后删除被取代的旧列。
// 本迁移在 AutoMigrate 之前执行, 因此新列先由此手工补齐, 之后由 AutoMigrate 收敛类型与索引。
//
// 建表, 加列, 删列一律在事务外执行。SQLite 修改表结构的方式是"建临时表 -> 复制 -> DROP 原表 -> 改名",
// 而 DROP TABLE 在 foreign_keys=ON 下会级联删除子表行。驱动的 RunWithoutForeignKey 会先关闭外键,
// 但 SQLite 在事务内静默忽略 PRAGMA foreign_keys, 使该保护失效, 从而清空 channel_models 等子表。
func migrateChannelGrants(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	// type 列存在与否作为执行标记, 已迁移的库直接跳过。
	if !db.Migrator().HasTable("channels") || !hasPhysicalColumn(db, "channels", "type") {
		return nil
	}

	// 新增列在 AutoMigrate 之前不存在, 先补齐才能回填。
	// 交由 GORM 按模型定义生成 DDL, 而非手写: 手写的类型若与 GORM 的取值不一致,
	// 主 AutoMigrate 会为收敛类型而重建 channels, 其 DROP TABLE 会级联清空 channel_models。
	newColumns := []struct {
		name  string // 列名, 用于判断是否已存在。
		field string // 模型字段名, 交给 GORM 生成该列的 DDL。
	}{
		{"dialect", "Dialect"},
		{"openai_chat_completion_path", "OpenAIChatCompletionPath"},
		{"openai_response_path", "OpenAIResponsePath"},
		{"anthropic_message_path", "AnthropicMessagePath"},
	}
	for _, column := range newColumns {
		if hasPhysicalColumn(db, "channels", column.name) {
			continue
		}
		if err := db.Migrator().AddColumn(&model.Channel{}, column.field); err != nil {
			return fmt.Errorf("failed to add channels.%s: %w", column.name, err)
		}
	}
	// 迁移 8/10 的历史快照创建的 channel_models 没有渠道外键，先补齐再写入授权。
	// 否则后续 AutoMigrate 补外键时会重建 channel_models，级联删除刚写入的授权。
	if !db.Migrator().HasConstraint(&model.Channel{}, "Models") {
		if err := db.Migrator().CreateConstraint(&model.Channel{}, "Models"); err != nil {
			return fmt.Errorf("failed to create channel_models foreign key: %w", err)
		}
	}
	// 用 CreateTable 而非 AutoMigrate: AutoMigrate 会顺着 ChannelKey 的外键回溯到父表 Channel 并一并迁移,
	// 从而重建 channels, 其 DROP TABLE 会级联清空 channel_models。CreateTable 只建这两张表, 且同样带上外键。
	for _, table := range []any{&model.ChannelKey{}, &model.ChannelGrant{}} {
		if db.Migrator().HasTable(table) {
			continue
		}
		if err := db.Migrator().CreateTable(table); err != nil {
			return fmt.Errorf("failed to create channel_keys and channel_grants: %w", err)
		}
	}

	// 中途失败重跑时清空上轮写入的凭据与授权, 这两张表只由本迁移写入, 清空是安全的。
	if err := db.Where("1 = 1").Delete(&model.ChannelGrant{}).Error; err != nil {
		return fmt.Errorf("failed to reset channel_grants: %w", err)
	}
	if err := db.Where("1 = 1").Delete(&model.ChannelKey{}).Error; err != nil {
		return fmt.Errorf("failed to reset channel_keys: %w", err)
	}

	// 模型主键到授权主键的映射, 供事务外重建 group_items 使用。
	var grantIDByModelOut map[int]int
	if err := db.Transaction(func(tx *gorm.DB) error {
		// 不显式 Select, 由 GORM 按字段名映射列, 避免 key 在各方言下的引号差异。
		type legacyChannelRow struct {
			ID   int    // 渠道主键。
			Type string // 原上游服务提供方。
			Key  string // 原渠道级凭据。
			Name string // 渠道名称。
		}
		channelRows := make([]legacyChannelRow, 0)
		if err := tx.Table("channels").Order("id ASC").Find(&channelRows).Error; err != nil {
			return fmt.Errorf("failed to read legacy channels: %w", err)
		}

		protocolByChannel := make(map[int]model.Protocol, len(channelRows))
		geminiChannels := make([]string, 0)
		keyIDByChannel := make(map[int]int, len(channelRows))
		for _, row := range channelRows {
			channelType := strings.ToLower(strings.TrimSpace(row.Type))
			mapped, ok := legacyChannelTypes[channelType]
			if !ok {
				mapped = legacyChannelTypes["openai"]
			}
			if channelType == "gemini" {
				geminiChannels = append(geminiChannels, fmt.Sprintf("%s(id=%d)", row.Name, row.ID))
			}
			protocolByChannel[row.ID] = mapped.protocol

			// 方言列的 DDL 默认值即 generic, 补列时已填, 无需回填。
			// 路径只在该类型给出非默认值时覆盖, 其余沿用补列时的默认值。
			if mapped.chatPath != "" {
				if err := tx.Table("channels").Where("id = ?", row.ID).Updates(map[string]any{
					"openai_chat_completion_path": mapped.chatPath,
					"openai_response_path":        mapped.responsePath,
					"anthropic_message_path":      mapped.messagePath,
				}).Error; err != nil {
					return fmt.Errorf("failed to backfill channel %d: %w", row.ID, err)
				}
			}

			// 原渠道级 key 为空时也建行, 保持模型与目标结构完整, 由用户后续补填。
			channelKey := model.ChannelKey{
				ChannelID:        row.ID,
				ChannelKeyConfig: model.ChannelKeyConfig{Name: "default", Key: row.Key, Enabled: true},
			}
			if err := tx.Create(&channelKey).Error; err != nil {
				return fmt.Errorf("failed to create channel_key for channel %d: %w", row.ID, err)
			}
			keyIDByChannel[row.ID] = channelKey.ID
		}

		// 渠道与渠道模型的统计列原地保留, 不做任何搬动: 三级统计各自独立累加, 互不换算。
		// channel_keys 是新增维度, 其统计从零开始。
		channelModels := make([]model.ChannelModel, 0)
		if err := tx.Order("id ASC").Find(&channelModels).Error; err != nil {
			return fmt.Errorf("failed to read channel_models: %w", err)
		}
		grantIDByModel := make(map[int]int, len(channelModels))
		for _, channelModel := range channelModels {
			keyID, ok := keyIDByChannel[channelModel.ChannelID]
			if !ok {
				continue
			}
			grant := model.ChannelGrant{
				ChannelModelID: channelModel.ID,
				ChannelKeyID:   keyID,
				Protocols:      protocolByChannel[channelModel.ChannelID],
			}
			if err := tx.Create(&grant).Error; err != nil {
				return fmt.Errorf("failed to create channel_grant for model %d: %w", channelModel.ID, err)
			}
			grantIDByModel[channelModel.ID] = grant.ID
		}

		grantIDByModelOut = grantIDByModel
		if len(geminiChannels) > 0 {
			log.Warnf("原 gemini 渠道已按 OpenAI Chat 协议迁移, 请手动将地址改为其 OpenAI 兼容端点: %s", strings.Join(geminiChannels, ", "))
		}
		return nil
	}); err != nil {
		return err
	}

	// group_items 需整表重建: 旧唯一索引与外键都建在 channel_model_id 上, 而 SQLite 原生 DROP COLUMN
	// 不允许删除仍被外键定义引用的列。重建同样必须在事务外, 否则 DROP 原表会级联清空自身。
	if err := migrateGroupItemsToGrants(db, grantIDByModelOut); err != nil {
		return err
	}

	// 删列会触发 SQLite 建临时表重建, 必须留在事务外, 否则级联清空 channel_models。
	for _, column := range []string{"key", "auto_sync"} {
		if err := dropColumnIfExists(db, &model.Channel{}, "channels", column); err != nil {
			return err
		}
	}
	if err := dropColumnIfExists(db, &model.ChannelModel{}, "channel_models", "source"); err != nil {
		return err
	}
	// type 列是本迁移的执行标记, 最后再删: 中途失败时标记仍在, 下次启动可重跑。
	return dropColumnIfExists(db, &model.Channel{}, "channels", "type")
}

// migrateGroupItemsToGrants 把 group_items 由引用渠道模型改为引用渠道授权, 保留原有主键与顺序。
// 无法定位目标或与同组已有分组项重复的旧行直接丢弃, 分组选中项指向被丢弃行时置零。
// 采用整表重建而非就地补列: 旧表的唯一索引与外键都建在 channel_model_id 上, 且 SQLite 的
// AutoMigrate 在为新列补外键时会重建表, 重建时只复制它从建表语句中解析到的列, 就地补上的列会被丢弃。
func migrateGroupItemsToGrants(db *gorm.DB, grantIDByModel map[int]int) error {
	if !db.Migrator().HasTable("group_items") || !hasPhysicalColumn(db, "group_items", "channel_model_id") {
		return nil
	}

	type legacyItemRow struct {
		ID             int // 分组项主键。
		GroupID        int // 所属分组主键。
		ChannelModelID int // 旧引用的渠道模型主键。
		Priority       int // 展示与故障转移顺序。
	}
	legacyItems := make([]legacyItemRow, 0)
	if err := db.Table("group_items").Select("id, group_id, channel_model_id, priority").
		Order("id ASC").Find(&legacyItems).Error; err != nil {
		return fmt.Errorf("failed to read legacy group_items: %w", err)
	}

	items := make([]model.GroupItem, 0, len(legacyItems))
	discardedIDs := make([]int, 0)
	seen := make(map[[2]int]struct{}, len(legacyItems))
	for _, item := range legacyItems {
		grantID, ok := grantIDByModel[item.ChannelModelID]
		itemKey := [2]int{item.GroupID, grantID}
		if _, exists := seen[itemKey]; !ok || exists {
			discardedIDs = append(discardedIDs, item.ID)
			continue
		}
		seen[itemKey] = struct{}{}
		items = append(items, model.GroupItem{
			ID:             item.ID,
			GroupID:        item.GroupID,
			ChannelGrantID: grantID,
			Priority:       item.Priority,
		})
	}

	if err := db.Migrator().DropTable(&model.GroupItem{}); err != nil {
		return fmt.Errorf("failed to drop legacy group_items: %w", err)
	}
	if err := db.AutoMigrate(&model.GroupItem{}); err != nil {
		return fmt.Errorf("failed to create group_items: %w", err)
	}
	if len(items) > 0 {
		if err := db.Create(&items).Error; err != nil {
			return fmt.Errorf("failed to create migrated group_items: %w", err)
		}
	}
	if len(discardedIDs) > 0 {
		if err := db.Model(&groupsTable{}).Where("active_item_id IN ?", discardedIDs).
			Update("active_item_id", 0).Error; err != nil {
			return fmt.Errorf("failed to clear discarded active items: %w", err)
		}
	}
	return clearStaleActiveItems(db)
}

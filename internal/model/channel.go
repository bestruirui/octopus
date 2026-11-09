package model

import (
	"encoding/json"
	"strings"
	"text/template"
)

// 渠道支持的上游线协议, 以位掩码存储, 一条渠道授权可同时支持多个协议。
type Protocol uint8

// 协议位从 1 << 1 开始, 1 << 0 保留待用; 位值会落库并与前端交互, 不可再变更。
const (
	ProtocolOpenAIChatCompletion Protocol = 1 << 1 // OpenAI Chat Completions 协议。
	ProtocolOpenAIResponse       Protocol = 1 << 2 // OpenAI Responses 协议。
	ProtocolAnthropicMessage     Protocol = 1 << 3 // Anthropic Messages 协议。
)

// 上游在标准协议之上的方言。
// 同一协议下不同服务商仍可能有请求体或响应体上的差异, 例如思考内容放 reasoning_content 还是 reasoning,
// 这类差异无法由地址与路径表达, 需按方言分支构造出站转换器。
// 地址与路径不属于方言范畴, 由前端按服务商预填到渠道字段上。
type Dialect string

const (
	DialectGeneric Dialect = "generic" // 标准协议, 不做任何厂商特化。
)

// 渠道的可编辑配置; 落库时平铺成 channels 的各列, 出入 JSON 时平铺成渠道读写接口的各字段。
// 单独成结构体是为了让库内实体, 读取副本与提交请求共用同一份定义: 这些字段在三处的形状本就一致,
// 各写一遍只会让加字段时漏改其中一处。
// 不带 binding 约束: 保存与探测都收这一份配置, 但两者的必填项不同 —— 探测发生在渠道尚未命名时,
// 故必填校验分别由 normalizeChannelConfig 与 fetchUpstreamModels 按各自的需要给出。
type ChannelConfig struct {
	Name                     string         `json:"name" gorm:"unique;not null"`                                                                        // 渠道名称。
	Dialect                  Dialect        `json:"dialect" gorm:"not null;default:generic"`                                                            // 上游方言, 决定出站转换器的厂商特化配置。
	Enabled                  bool           `json:"enabled" gorm:"default:true"`                                                                        // 渠道是否可用。
	BaseURL                  string         `json:"base_url"`                                                                                           // 上游地址, 各协议共用。
	OpenAIChatCompletionPath string         `json:"openai_chat_completion_path" gorm:"column:openai_chat_completion_path;default:/v1/chat/completions"` // OpenAI Chat Completions 请求路径; 留空由后端填默认路径。
	OpenAIResponsePath       string         `json:"openai_response_path" gorm:"column:openai_response_path;default:/v1/responses"`                      // OpenAI Responses 请求路径; 留空由后端填默认路径。
	AnthropicMessagePath     string         `json:"anthropic_message_path" gorm:"column:anthropic_message_path;default:/v1/messages"`                   // Anthropic Messages 请求路径; 留空由后端填默认路径。
	Proxy                    bool           `json:"proxy" gorm:"default:false"`                                                                         // 是否使用代理。
	ChannelProxy             string         `json:"channel_proxy"`                                                                                      // 渠道专用代理地址; 留空表示不用渠道专用代理。
	CustomHeader             []CustomHeader `json:"custom_header" gorm:"serializer:json"`                                                               // 追加到上游请求的 Header。
	ParamOverride            string         `json:"param_override"`                                                                                     // 请求参数覆盖配置; 留空表示不覆盖。
	MatchRegex               string         `json:"match_regex"`                                                                                        // 拉取模型列表时的过滤表达式; 留空表示不过滤。
}

// 单个上游渠道的共享配置; 路径按协议分别配置, 凭据由 ChannelKey 提供。
type Channel struct {
	ID            int            `json:"id" gorm:"primaryKey"` // 渠道主键。
	ChannelConfig                // 可编辑配置, 平铺为 channels 的各列。
	Keys          []ChannelKey   `json:"-" gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE"` // 渠道下的上游凭据; 不出 JSON, 读取走 ChannelDetail。
	Models        []ChannelModel `json:"-" gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE"` // 渠道提供的模型; 不出 JSON, 读取走 ChannelDetail。
	StatsMetrics                 // 渠道自身的累计统计。
}

// 渠道凭据的可编辑配置; 名称在渠道内唯一, 整体替换时以它为匹配依据。
// 与 ChannelConfig 同理, 库内实体与读写接口共用这一份定义。
type ChannelKeyConfig struct {
	Name    string `json:"name" gorm:"not null;index:idx_channel_key,unique"` // 凭据名称, 界面展示与人工识别用。
	Key     string `json:"key" gorm:"not null"`                               // 上游访问凭据。
	Enabled bool   `json:"enabled" gorm:"default:true"`                       // 是否可用, 禁用后不参与选路但保留统计。
}

// 渠道下的一份上游凭据; 不同凭据通常对应不同的额度与计费。
type ChannelKey struct {
	ID               int `json:"id" gorm:"primaryKey"`                                    // 凭据主键。
	ChannelID        int `json:"channel_id" gorm:"not null;index:idx_channel_key,unique"` // 所属渠道 ID。
	ChannelKeyConfig     // 可编辑配置, 平铺为 channel_keys 的各列。
	StatsMetrics         // 该凭据自身的累计统计。
}

// 渠道提供的单个上游模型。
type ChannelModel struct {
	ID           int    `json:"id" gorm:"primaryKey"`                                      // 渠道模型主键。
	ChannelID    int    `json:"channel_id" gorm:"not null;index:idx_channel_model,unique"` // 所属渠道 ID。
	Name         string `json:"name" gorm:"not null;index:idx_channel_model,unique"`       // 上游模型名称。
	StatsMetrics        // 该模型自身的累计统计。
}

// 渠道内的一条上游授权: 指定模型使用指定凭据时支持的协议集合, 也是转发的最小单位。
// 上游按凭据分组授权模型, 且同一凭据下不同模型可能位于不同协议的端点, 故协议归属于模型与凭据的组合。
type ChannelGrant struct {
	ID             int           `json:"id" gorm:"primaryKey"`                                                                               // 授权主键。
	ChannelModelID int           `json:"channel_model_id" gorm:"not null;index:idx_channel_grant,unique"`                                    // 渠道模型 ID。
	ChannelModel   *ChannelModel `json:"channel_model,omitempty" gorm:"foreignKey:ChannelModelID;references:ID;constraint:OnDelete:CASCADE"` // 授权引用的渠道模型。
	ChannelKeyID   int           `json:"channel_key_id" gorm:"not null;index:idx_channel_grant,unique"`                                      // 凭据 ID。
	ChannelKey     *ChannelKey   `json:"channel_key,omitempty" gorm:"foreignKey:ChannelKeyID;references:ID;constraint:OnDelete:CASCADE"`     // 授权引用的凭据。
	Protocols      Protocol      `json:"protocols" gorm:"not null"`                                                                          // 该组合支持的协议位掩码。
}

// 渠道读写副本: 编辑表单的完整形状, 读取与提交同构。
// 读写同构使前端无需为提交另建一套类型, 后端也无需按字段比对增量: 表单本就一次给出完整配置。
// 凭据与模型只给界面用得上的字段: 两者在渠道内按名称唯一, 提交时也按名称引用, 主键与统计都无从使用。
// 集合字段恒为数组, 读取侧承诺不为 null。
type ChannelDetail struct {
	ID            int                  `json:"id"` // 渠道主键; 创建时提交 0, 由数据库分配。
	ChannelConfig                      // 渠道自身的可编辑配置。
	Keys          []ChannelKeyConfig   `json:"keys"`   // 渠道下的上游凭据。
	Models        []string             `json:"models"` // 渠道提供的上游模型名称。
	Grants        []ChannelGrantConfig `json:"grants"` // 渠道下的授权。
}

// 渠道授权的可编辑形状, 两侧按名称引用。
// 名称在渠道内唯一, 本就是模型与凭据的匹配依据; 而新增的模型与凭据在同一事务内才分配主键,
// 提交方拿不到, 无法在一次请求里既建模型又给它授权。读写用同一种寻址, 界面无需在主键与名称之间翻译。
type ChannelGrantConfig struct {
	ModelName string   `json:"model_name"` // 渠道模型名称。
	KeyName   string   `json:"key_name"`   // 渠道凭据名称。
	Protocols Protocol `json:"protocols"`  // 该组合支持的协议位掩码。
}

// 渠道及其模型的累计统计, 自带展示所需的名称与启停状态。
// 这一份同时充当渠道列表项: 列表页要展示的名称, 启停与模型个数都在此, 模型个数即 Models 的长度,
// 故不再另给渠道概览接口 —— 列表页与首页榜单共用这一条响应, 整份配置在点开编辑时由 /channel/detail 单独取。
// 名称与渠道配置重复给出是有意的: 消费方由此无需再按主键回查渠道。
type ChannelStats struct {
	ChannelID    int                 `json:"channel_id"`   // 渠道主键。
	ChannelName  string              `json:"channel_name"` // 渠道名称。
	Enabled      bool                `json:"enabled"`      // 渠道是否可用, 供列表页的开关与过滤使用。
	Models       []ChannelModelStats `json:"models"`       // 该渠道各模型的独立统计, 恒为数组; 长度即渠道的模型个数。
	StatsMetrics                     // 渠道自身的累计统计。
}

// 单个渠道模型的累计统计。
type ChannelModelStats struct {
	ModelID      int    `json:"model_id"`   // 渠道模型主键。
	ModelName    string `json:"model_name"` // 上游模型名称。
	StatsMetrics        // 该模型自身的累计统计。
}

// 供分组页选取的一条候选授权, 字段与分组成员的展示字段一一对应。
// 分组页只需按渠道分组列出可选授权并判断可用性, 由此无需再拉整份渠道列表:
// 那里带着统计, 路径, 代理与凭据明文, 与选取成员无关。
type ChannelGrantCandidate struct {
	ID          int      `json:"id"`           // 授权主键, 分组成员按它引用。
	ChannelID   int      `json:"channel_id"`   // 授权所属渠道 ID。
	ChannelName string   `json:"channel_name"` // 授权所属渠道名称。
	ModelName   string   `json:"model_name"`   // 授权引用的上游模型名称。
	KeyName     string   `json:"key_name"`     // 授权引用的凭据名称。
	Protocols   Protocol `json:"protocols"`    // 授权支持的协议位掩码。
	Available   bool     `json:"available"`    // 渠道与凭据均启用且模型, 凭据均存在时为真。
}

// 追加到上游请求的单个 Header 操作; Op 为 rename/copy 时 HeaderKey 是源头、HeaderValue 是目标头, delete 时值不参与。
type CustomHeader struct {
	Op          string `json:"op,omitempty"`        // set | delete | rename | copy; 留空等价 set, 与历史数据兼容。
	HeaderKey   string `json:"header_key"`          // Header 名称; rename/copy 的源名称。
	HeaderValue string `json:"header_value"`        // Header 值, 支持 {{...}} 模板与 {client_header:xxx}; rename/copy 的目标名称。
	Condition   string `json:"condition,omitempty"` // 生效条件; 留空恒生效, 否则 Go template 渲染为 "true" 才生效。
}

// 参数覆盖操作类型。
const (
	OverrideOpSet         = "set"           // 强制写入, 存在则替换, 不存在则新增。
	OverrideOpSetIfAbsent = "set_if_absent" // 仅当路径不存在时写入。
	OverrideOpDelete      = "delete"        // 删除路径。
	OverrideOpRename      = "rename"        // Header 专用: 把 HeaderKey 改名为 HeaderValue, 保留全部值。
	OverrideOpCopy        = "copy"          // Header 专用: 把 HeaderKey 复制一份为 HeaderValue, 保留原头。
)

// 参数覆盖的单条操作; 覆盖配置为 JSON 数组时逐条执行。
type OverrideOperation struct {
	Op        string `json:"op"`                  // set | set_if_absent | delete。
	Path      string `json:"path,omitempty"`      // 目标字段路径; 新格式下点号按嵌套路径解析。
	Value     string `json:"value,omitempty"`     // set/set_if_absent 的值, 支持 {{...}} 模板。
	Condition string `json:"condition,omitempty"` // 留空恒真; 否则渲染结果为 "true" 才执行。
}

// OverrideTemplateFuncs 是条件与值模板共用的函数集, 保存校验与转发渲染须用同一份以保持一致。
var OverrideTemplateFuncs = template.FuncMap{
	"toJSON": marshalTemplateValue,
}

func marshalTemplateValue(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// ValidateOverrideTemplate 校验一段可选的 Go template 文本; 不含 {{ 视为静态文本直接通过。
func ValidateOverrideTemplate(text string) error {
	if !strings.Contains(text, "{{") {
		return nil
	}
	_, err := template.New("override").Funcs(OverrideTemplateFuncs).Parse(text)
	return err
}

// 按凭据拉取上游模型列表的请求; 渠道尚未保存时也可试拉, 故随请求携带拉取所需的渠道配置。
// 直接收整份渠道配置而不另立探测专用形状: 探测发生在编辑表单内, 提交方手上本就是完整配置,
// 且探测用的地址, 路径, 代理与过滤表达式必须与保存后生效的完全一致, 收同一份即可; 探测用不上的字段忽略即可。
// 不指定协议: 一次探测同时试 OpenAI 与 Anthropic 两侧, 协议支持情况由各侧的响应决定。
type ChannelFetchModelRequest struct {
	Channel ChannelConfig `json:"channel"`                // 用于拉取的渠道配置, 提供地址, 路径, 代理与 Header。
	Key     string        `json:"key" binding:"required"` // 拉取使用的上游凭据。
}

// 探测到的单个上游模型及其支持的协议集合。
type ChannelFetchModel struct {
	Name      string   `json:"name"`      // 上游模型名称。
	Protocols Protocol `json:"protocols"` // 由探测结果得出的协议位掩码。
}

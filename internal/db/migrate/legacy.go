package migrate

// 历史迁移使用的表结构快照。
// 已发布的迁移必须按当时的表结构执行, 不能随 internal/model 的后续演进而改变语义,
// 因此凡是历史迁移需要读写的旧结构一律在此定义, 不再引用 model 包。
// 各迁移函数内另有只读取特定旧列的局部结构, 与此处的快照互不影响。

// channels 表在版本 5 时的地址与凭据列，供 AutoMigrate 前补列使用。
type channelsV5 struct {
	ID      int    `gorm:"primaryKey"` // 渠道主键。
	BaseURL string // 唯一的上游基础地址。
	Key     string // 唯一的上游访问凭据。
}

func (channelsV5) TableName() string { return "channels" }

// channels 表快照, 供历史迁移定位表名并读写版本 8 的内嵌统计列。
type channelsTable struct {
	ID int `gorm:"primaryKey"` // 渠道主键。
	SnapshotStatsMetrics
}

func (channelsTable) TableName() string { return "channels" }

// channel_models 表在版本 8 时的结构, 含此后已删除的 source 与统计列。
type channelModelsV8 struct {
	ID        int    `gorm:"primaryKey"`                                   // 渠道模型主键。
	ChannelID int    `gorm:"not null;index:idx_channel_model_name,unique"` // 所属渠道 ID。
	Name      string `gorm:"not null;index:idx_channel_model_name,unique"` // 上游模型名称。
	Source    string `gorm:"not null;default:auto"`                        // 模型来源: auto 或 manual。
	SnapshotStatsMetrics
}

func (channelModelsV8) TableName() string { return "channel_models" }

// group_items 表在版本 8 时的结构, 引用的是渠道模型而非渠道授权。
type groupItemsV8 struct {
	ID             int `gorm:"primaryKey"`                                    // 分组项主键。
	GroupID        int `gorm:"not null;index:idx_group_channel_model,unique"` // 所属分组 ID。
	ChannelModelID int `gorm:"not null;index:idx_group_channel_model,unique"` // 引用的渠道模型 ID。
	Priority       int `gorm:"not null"`                                      // 展示与故障转移顺序。
}

func (groupItemsV8) TableName() string { return "group_items" }

// groups 表快照, 供历史迁移定位表名并读写 active_item_id。
type groupsTable struct {
	ID           int `gorm:"primaryKey"`         // 分组主键。
	ActiveItemID int `gorm:"not null;default:0"` // 手动模式下选中的成员主键。
}

func (groupsTable) TableName() string { return "groups" }

// SnapshotStatsMetrics 是历史迁移涉及的内嵌统计列，必须导出才能被 GORM 识别。
type SnapshotStatsMetrics struct {
	InputToken     int64   `gorm:"bigint"`    // 累计输入 token。
	OutputToken    int64   `gorm:"bigint"`    // 累计输出 token。
	InputCost      float64 `gorm:"type:real"` // 累计输入费用。
	OutputCost     float64 `gorm:"type:real"` // 累计输出费用。
	WaitTime       int64   `gorm:"bigint"`    // 累计等待耗时。
	RequestSuccess int64   `gorm:"bigint"`    // 累计成功请求数。
	RequestFailed  int64   `gorm:"bigint"`    // 累计失败请求数。
}

// 版本 8 引入的渠道模型来源取值。
const (
	snapshotModelSourceAuto   = "auto"   // 通过上游接口自动获取。
	snapshotModelSourceManual = "manual" // 管理员手动配置。
)

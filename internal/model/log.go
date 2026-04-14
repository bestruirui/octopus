package model

// AttemptStatus 尝试状态
type AttemptStatus string

const (
	AttemptSuccess      AttemptStatus = "success"       // 转发成功
	AttemptFailed       AttemptStatus = "failed"        // 转发失败
	AttemptCircuitBreak AttemptStatus = "circuit_break" // 熔断跳过
	AttemptSkipped      AttemptStatus = "skipped"       // 其他原因跳过（禁用、无Key、类型不兼容等）
)

// ChannelAttempt 记录单次渠道尝试的决策和结果
type ChannelAttempt struct {
	ChannelID    int           `json:"channel_id"`
	ChannelKeyID int           `json:"channel_key_id,omitempty"`
	ChannelName  string        `json:"channel_name"`
	ModelName    string        `json:"model_name"`
	AttemptNum   int           `json:"attempt_num"`
	Status       AttemptStatus `json:"status"`
	Duration     int           `json:"duration"`
	Sticky       bool          `json:"sticky,omitempty"`
	Msg          string        `json:"msg,omitempty"`
}

type RelayLog struct {
	ID                int64            `json:"id" gorm:"primaryKey;autoIncrement:false"` // Snowflake ID
	Time              int64            `json:"time"`                                     // 时间戳（秒）
	RequestModelName  string           `json:"request_model_name"`                       // 请求模型名称
	RequestAPIKeyName string           `json:"request_api_key_name"`                     // 请求使用的 API Key 名称
	ChannelId         int              `json:"channel"`                                  // 实际使用的渠道ID
	ChannelName       string           `json:"channel_name"`                             // 渠道名称
	ActualModelName   string           `json:"actual_model_name"`                        // 实际使用模型名称
	InputTokens       int              `json:"input_tokens"`                             // 输入Token
	OutputTokens      int              `json:"output_tokens"`                            // 输出 Token
	Ftut              int              `json:"ftut"`                                     // 首字时间(毫秒)
	UseTime           int              `json:"use_time"`                                 // 总用时(毫秒)
	Cost              float64          `json:"cost"`                                     // 消耗费用
	RequestContent    string           `json:"request_content"`                          // 请求内容
	ResponseContent   string           `json:"response_content"`                         // 响应内容
	Error             string           `json:"error"`                                    // 错误信息
	Attempts          []ChannelAttempt `json:"attempts" gorm:"serializer:json"`          // 所有尝试记录
	TotalAttempts     int              `json:"total_attempts"`                           // 总尝试次数
}

// RelayLogListItem 日志列表轻量条目，排除了 RequestContent 和 ResponseContent 大字段
type RelayLogListItem struct {
	ID                int64            `json:"id" gorm:"column:id"`
	Time              int64            `json:"time" gorm:"column:time"`
	RequestModelName  string           `json:"request_model_name" gorm:"column:request_model_name"`
	RequestAPIKeyName string           `json:"request_api_key_name" gorm:"column:request_api_key_name"`
	ChannelId         int              `json:"channel" gorm:"column:channel_id"`
	ChannelName       string           `json:"channel_name" gorm:"column:channel_name"`
	ActualModelName   string           `json:"actual_model_name" gorm:"column:actual_model_name"`
	InputTokens       int              `json:"input_tokens" gorm:"column:input_tokens"`
	OutputTokens      int              `json:"output_tokens" gorm:"column:output_tokens"`
	Ftut              int              `json:"ftut" gorm:"column:ftut"`
	UseTime           int              `json:"use_time" gorm:"column:use_time"`
	Cost              float64          `json:"cost" gorm:"column:cost"`
	Error             string           `json:"error" gorm:"column:error"`
	Attempts          []ChannelAttempt `json:"attempts" gorm:"column:attempts;serializer:json"`
	TotalAttempts     int              `json:"total_attempts" gorm:"column:total_attempts"`
}

// TableName 指定 RelayLogListItem 使用与 RelayLog 相同的数据库表
func (RelayLogListItem) TableName() string { return "relay_logs" }

// ToListItem 将完整的 RelayLog 转换为轻量的列表条目
func (r *RelayLog) ToListItem() RelayLogListItem {
	return RelayLogListItem{
		ID:                r.ID,
		Time:              r.Time,
		RequestModelName:  r.RequestModelName,
		RequestAPIKeyName: r.RequestAPIKeyName,
		ChannelId:         r.ChannelId,
		ChannelName:       r.ChannelName,
		ActualModelName:   r.ActualModelName,
		InputTokens:       r.InputTokens,
		OutputTokens:      r.OutputTokens,
		Ftut:              r.Ftut,
		UseTime:           r.UseTime,
		Cost:              r.Cost,
		Error:             r.Error,
		Attempts:          r.Attempts,
		TotalAttempts:     r.TotalAttempts,
	}
}

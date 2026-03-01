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
	ID               int64            `json:"id" gorm:"primaryKey;autoIncrement:false"` // Snowflake ID
	Time             int64            `json:"time"`                                     // 时间戳（秒）
	RequestModelName string           `json:"request_model_name"`                       // 请求模型名称
	ChannelId        int              `json:"channel"`                                  // 实际使用的渠道ID
	ChannelName      string           `json:"channel_name"`                             // 渠道名称
	ActualModelName  string           `json:"actual_model_name"`                        // 实际使用模型名称
	InputTokens      int              `json:"input_tokens"`                             // 输入Token
	OutputTokens     int              `json:"output_tokens"`                            // 输出 Token
	Ftut             int              `json:"ftut"`                                     // 首字时间(毫秒)
	UseTime          int              `json:"use_time"`                                 // 总用时(毫秒)
	Cost             float64          `json:"cost"`                                     // 消耗费用
	RequestContent   string           `json:"request_content"`                          // 请求内容
	ResponseContent  string           `json:"response_content"`                         // 响应内容
	Error            string           `json:"error"`                                    // 错误信息
	Attempts         []ChannelAttempt `json:"attempts" gorm:"serializer:json"`          // 所有尝试记录
	TotalAttempts    int              `json:"total_attempts"`                           // 总尝试次数
}

type RelayLogListFilter struct {
	Group   *string `json:"group,omitempty"`
	Model   *string `json:"model,omitempty"`
	Retried *bool   `json:"retried,omitempty"`
	Channel *string `json:"channel,omitempty"`
}

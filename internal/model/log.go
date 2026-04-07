package model

// AttemptStatus 尝试状态
type AttemptStatus string

const (
	AttemptSuccess      AttemptStatus = "success"       // 转发成功
	AttemptFailed       AttemptStatus = "failed"        // 转发失败
	AttemptCircuitBreak AttemptStatus = "circuit_break" // 熔断跳过
	AttemptSkipped      AttemptStatus = "skipped"       // 其他原因跳过（禁用、无Key、类型不兼容等）
)

// SkipReason 跳过原因枚举
type SkipReason string

const (
	SkipReasonRateLimited SkipReason = "rate_limited" // RPM 限制
	SkipReasonConcurrency SkipReason = "concurrency"  // 并发限制
	SkipReasonCooldown429 SkipReason = "cooldown_429" // 429 冷却
	SkipReasonCircuitOpen SkipReason = "circuit_open" // 熔断开启
	SkipReasonDisabled    SkipReason = "disabled"     // Key/渠道禁用
	SkipReasonNoKey       SkipReason = "no_key"       // 无可用 Key
)

// ChannelAttempt 记录单次渠道尝试的决策和结果。
// 这里保留 ChannelKeyID 和 SkipReason，便于把日志颗粒度下沉到 key 级别。
type ChannelAttempt struct {
	ChannelID     int           `json:"channel_id"`
	ChannelKeyID  int           `json:"channel_key_id,omitempty"`
	ChannelKeyRemark string     `json:"channel_key_remark,omitempty"`
	ChannelName   string        `json:"channel_name"`
	ModelName     string        `json:"model_name"`
	AttemptNum    int           `json:"attempt_num"`
	Status        AttemptStatus `json:"status"`
	Duration      int           `json:"duration"`
	Sticky        bool          `json:"sticky,omitempty"`
	SkipReason    SkipReason    `json:"skip_reason,omitempty"` // 标准化跳过原因，前端可直接据此展示 rate_limited/cooldown_429 等状态
	ErrorClass    string        `json:"error_class,omitempty"`
	Msg           string        `json:"msg,omitempty"`
}

type RelayLog struct {
	ID                int64           `json:"id" gorm:"primaryKey;autoIncrement:false"` // Snowflake ID
	Time              int64           `json:"time"`                                     // 时间戳（秒）
	RequestModelName  string          `json:"request_model_name"`                       // 请求模型名称
	RequestAPIKeyName string          `json:"request_api_key_name"`                     // 请求使用的 API Key 名称
	ChannelId         int             `json:"channel"`                                  // 实际使用的渠道ID
	ChannelName       string          `json:"channel_name"`                             // 渠道名称
	ActualModelName   string          `json:"actual_model_name"`                        // 实际使用模型名称
	InputTokens       int             `json:"input_tokens"`                             // 输入Token
	OutputTokens      int             `json:"output_tokens"`                            // 输出 Token
	Ftut              int             `json:"ftut"`                                     // 首字时间(毫秒)
	UseTime           int             `json:"use_time"`                                 // 总用时(毫秒)
	Cost              float64         `json:"cost"`                                     // 消耗费用
	RequestContent    string          `json:"request_content"`                          // 请求内容
	ResponseContent   string          `json:"response_content"`                         // 响应内容
	Error             string          `json:"error"`                                    // 错误信息
	Attempts          []ChannelAttempt `json:"attempts" gorm:"serializer:json"`          // 所有尝试记录
	TotalAttempts     int             `json:"total_attempts"`                           // 总尝试次数
}

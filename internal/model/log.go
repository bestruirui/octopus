package model

// ChannelAttempt 记录单次渠道尝试的信息
type ChannelAttempt struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	ModelName   string `json:"model_name"`
	Round       int    `json:"round"`       // 第几轮 (1-3)
	AttemptNum  int    `json:"attempt_num"` // 第几次尝试
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	Duration    int    `json:"duration"` // 耗时(毫秒)

	RelayStatusCode  int    `json:"relay_status_code,omitempty"`
	RelayErrorSource string `json:"relay_error_source,omitempty"`
	RelayRetryable   bool   `json:"relay_retryable,omitempty"`
	RelayTrippable   bool   `json:"relay_trippable,omitempty"`

	BreakerKey        string `json:"breaker_key,omitempty"`
	CBDecision        string `json:"cb_decision,omitempty"`
	CBStateBefore     string `json:"cb_state_before,omitempty"`
	CBStateAfter      string `json:"cb_state_after,omitempty"`
	CBTripCount       int    `json:"cb_trip_count,omitempty"`
	CBOpenUntil       string `json:"cb_open_until,omitempty"`
	ProbeInFlight     bool   `json:"probe_in_flight,omitempty"`
	EarliestRetryAt   string `json:"earliest_retry_at,omitempty"`
	RetryAfterSeconds int    `json:"retry_after_seconds,omitempty"`
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
	SuccessfulRound  int              `json:"successful_round"`                         // 成功的轮次
	CBLogLevelMax    int              `json:"cb_log_level_max"`                         // 熔断相关日志等级 (1~3)
}

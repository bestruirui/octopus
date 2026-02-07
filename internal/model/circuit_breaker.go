package model

type CircuitBreakerConfig struct {
	Enabled          bool    `json:"enabled"`
	FailureThreshold int     `json:"failure_threshold"`
	BaseCooldownMS   int     `json:"base_cooldown_ms"`
	MaxCooldownMS    int     `json:"max_cooldown_ms"`
	BackoffFactor    float64 `json:"backoff_factor"`
	JitterMin        float64 `json:"jitter_min"`
	JitterMax        float64 `json:"jitter_max"`
	DecayWindowMS    int     `json:"decay_window_ms"`
}

type CircuitBreakerState string

const (
	CircuitBreakerStateClosed   CircuitBreakerState = "CLOSED"
	CircuitBreakerStateOpen     CircuitBreakerState = "OPEN"
	CircuitBreakerStateHalfOpen CircuitBreakerState = "HALF_OPEN"
)

type GroupCircuitBreakerItemState struct {
	GroupID             int                 `json:"group_id"`
	GroupName           string              `json:"group_name"`
	ChannelID           int                 `json:"channel_id"`
	ChannelName         string              `json:"channel_name"`
	ModelName           string              `json:"model_name"`
	BreakerKey          string              `json:"breaker_key"`
	State               CircuitBreakerState `json:"state"`
	ConsecutiveFailures int                 `json:"consecutive_failures"`
	TripCount           int                 `json:"trip_count"`
	LastFailureAt       string              `json:"last_failure_at,omitempty"`
	LastFailureReason   string              `json:"last_failure_reason,omitempty"`
	LastTripAt          string              `json:"last_trip_at,omitempty"`
	OpenUntil           string              `json:"open_until,omitempty"`
	OpenRemainingSecond int                 `json:"open_remaining_second,omitempty"`
	ProbeInFlight       bool                `json:"probe_in_flight"`
}

type GroupCircuitBreakerStatesResponse struct {
	GroupID   int                            `json:"group_id"`
	GroupName string                         `json:"group_name"`
	Items     []GroupCircuitBreakerItemState `json:"items"`
}

type CircuitBreakerResetResponse struct {
	ChannelID        int      `json:"channel_id"`
	AffectedBreakers int      `json:"affected_breakers"`
	BreakerKeys      []string `json:"breaker_keys,omitempty"`
}

type CircuitBreakerAllOpenScope struct {
	GroupID   int    `json:"group_id"`
	ModelName string `json:"model_name"`
}

type CircuitBreakerAllOpenData struct {
	Reason            string                     `json:"reason"`
	EarliestRetryAt   string                     `json:"earliest_retry_at"`
	RetryAfterSeconds int                        `json:"retry_after_seconds"`
	Scope             CircuitBreakerAllOpenScope `json:"scope"`
}

package balancer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/log"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed   CircuitState = iota // 正常通行
	StateOpen                         // 熔断中，拒绝所有请求
	StateHalfOpen                     // 半开，仅允许单个试探请求
)

// circuitEntry 单个熔断器条目（粒度：渠道 + 模型）
type circuitEntry struct {
	ChannelID           int
	ModelName           string
	State               CircuitState
	ConsecutiveFailures int64
	LastFailureTime     time.Time
	TripCount           int // 累计熔断触发次数（用于指数退避）
	LastErrorTime       int64
	LastError           string
	ManualDisabled      bool // 手动禁用（管理员显式关闭该渠道的该模型）
	ChannelName         string
	mu                  sync.Mutex
}

// 全局熔断器存储
var globalBreaker sync.Map // key: string -> value: *circuitEntry

// circuitKey 生成熔断器键：channelID:modelName
func circuitKey(channelID int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelID, modelName)
}

// getOrCreateEntry 获取或创建熔断器条目
func getOrCreateEntry(channelID int, modelName string) *circuitEntry {
	key := circuitKey(channelID, modelName)
	if v, ok := globalBreaker.Load(key); ok {
		return v.(*circuitEntry)
	}
	entry := &circuitEntry{
		ChannelID: channelID,
		ModelName: modelName,
		State:     StateClosed,
	}
	actual, _ := globalBreaker.LoadOrStore(key, entry)
	return actual.(*circuitEntry)
}

// Init 从数据库加载持久化的熔断器状态（启动时调用）。
func Init(ctx context.Context) error {
	records, err := op.CircuitBreakerList(ctx)
	if err != nil {
		return fmt.Errorf("failed to load circuit breakers: %w", err)
	}
	for i := range records {
		r := &records[i]
		entry := getOrCreateEntry(r.ChannelID, r.ModelName)
		entry.mu.Lock()
		entry.State = CircuitState(r.State)
		entry.TripCount = r.TripCount
		entry.ConsecutiveFailures = r.ConsecutiveFailures
		entry.ManualDisabled = r.ManualDisabled
		entry.LastErrorTime = r.LastErrorTime
		entry.LastError = r.LastError
		entry.ChannelName = r.ChannelName
		if r.LastErrorTime > 0 {
			entry.LastFailureTime = time.Unix(r.LastErrorTime, 0)
		}
		entry.mu.Unlock()
	}
	log.Infof("loaded %d circuit breaker records", len(records))
	return nil
}

// persist 将条目状态写回数据库（内存状态为运行时权威，DB 用于展示与重启恢复）。
func (e *circuitEntry) persist() {
	if e.ChannelName == "" {
		e.ChannelName = fmt.Sprintf("channel_%d", e.ChannelID)
	}
	_ = op.CircuitBreakerUpsert(&model.CircuitBreaker{
		ChannelID:           e.ChannelID,
		ChannelName:         e.ChannelName,
		ModelName:           e.ModelName,
		State:               int(e.State),
		TripCount:           e.TripCount,
		ConsecutiveFailures: e.ConsecutiveFailures,
		ManualDisabled:      e.ManualDisabled,
		LastErrorTime:       e.LastErrorTime,
		LastError:           e.LastError,
	})
}

// getThreshold 获取熔断阈值配置
func getThreshold() int64 {
	v, err := op.SettingGetInt(model.SettingKeyCircuitBreakerThreshold)
	if err != nil || v <= 0 {
		return 5
	}
	return int64(v)
}

// GetCooldown 获取当前冷却时间（带指数退避）
func GetCooldown(tripCount int) time.Duration {
	base, err := op.SettingGetInt(model.SettingKeyCircuitBreakerCooldown)
	if err != nil || base <= 0 {
		base = 60
	}
	maxCooldown, err := op.SettingGetInt(model.SettingKeyCircuitBreakerMaxCooldown)
	if err != nil || maxCooldown <= 0 {
		maxCooldown = 600
	}

	// 指数退避：baseCooldown * 2^(tripCount-1)
	cooldown := base
	if tripCount > 1 {
		shift := tripCount - 1
		if shift > 20 { // 防止溢出
			shift = 20
		}
		cooldown = base << shift
	}
	if cooldown > maxCooldown {
		cooldown = maxCooldown
	}

	return time.Duration(cooldown) * time.Second
}

// IsTripped 检查渠道+模型是否应被跳过（手动禁用 / 熔断冷却中 / 半开试探中）。
func IsTripped(channelID int, modelName string) (tripped bool, remaining time.Duration) {
	key := circuitKey(channelID, modelName)
	v, ok := globalBreaker.Load(key)
	if !ok {
		return false, 0 // 无记录，视为 Closed
	}
	entry := v.(*circuitEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// 手动禁用：无论熔断器状态如何，一律跳过
	if entry.ManualDisabled {
		return true, 0
	}

	switch entry.State {
	case StateClosed:
		return false, 0

	case StateOpen:
		cooldown := GetCooldown(entry.TripCount)
		elapsed := time.Since(entry.LastFailureTime)
		if elapsed >= cooldown {
			entry.State = StateHalfOpen
			log.Infof("circuit breaker [%s] Open -> HalfOpen (cooldown %v elapsed)", key, cooldown)
			return false, 0
		}
		// 仍在冷却中
		return true, cooldown - elapsed

	case StateHalfOpen:
		// 已有试探请求在进行中，拒绝其他请求
		return true, 0

	default:
		return false, 0
	}
}

// IsManualDisabled 检查某个渠道+模型是否被手动禁用。
func IsManualDisabled(channelID int, modelName string) bool {
	v, ok := globalBreaker.Load(circuitKey(channelID, modelName))
	if !ok {
		return false
	}
	entry := v.(*circuitEntry)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return entry.ManualDisabled
}

// RecordSuccess 记录成功，重置熔断器状态（手动禁用状态不受影响）。
func RecordSuccess(channelID int, modelName string) {
	key := circuitKey(channelID, modelName)
	v, ok := globalBreaker.Load(key)
	if !ok {
		return
	}
	entry := v.(*circuitEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.State == StateHalfOpen {
		log.Infof("circuit breaker [%s] HalfOpen -> Closed (probe succeeded)", key)
	}

	// 重置熔断状态，但保留手动禁用标记
	entry.State = StateClosed
	entry.ConsecutiveFailures = 0
	entry.TripCount = 0
	entry.persist()
}

// RecordFailure 记录失败。若错误为额度耗尽（quota exhausted），直接触发熔断；
// 否则按连续失败次数累计，达到阈值后触发熔断。
func RecordFailure(channelID int, modelName, channelName string, statusCode int, errMsg string) {
	key := circuitKey(channelID, modelName)
	entry := getOrCreateEntry(channelID, modelName)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.ChannelName = channelName
	entry.LastErrorTime = time.Now().Unix()
	entry.LastError = errMsg

	// 手动禁用期间不累计失败状态（但仍记录错误信息）
	if entry.ManualDisabled {
		entry.persist()
		return
	}

	// 额度耗尽错误：直接进入熔断，无需等待连续失败阈值
	if IsQuotaExhausted(statusCode, errMsg) {
		entry.State = StateOpen
		entry.TripCount++
		entry.ConsecutiveFailures = 0
		entry.LastFailureTime = time.Now()
		log.Warnf("circuit breaker [%s] quota exhausted -> Open (tripCount=%d, cooldown=%v)",
			key, entry.TripCount, GetCooldown(entry.TripCount))
		entry.persist()
		return
	}

	switch entry.State {
	case StateClosed:
		entry.ConsecutiveFailures++
		threshold := getThreshold()
		if entry.ConsecutiveFailures >= threshold {
			entry.State = StateOpen
			entry.TripCount++
			entry.LastFailureTime = time.Now()
			log.Warnf("circuit breaker [%s] Closed -> Open (failures=%d >= threshold=%d, tripCount=%d, cooldown=%v)",
				key, entry.ConsecutiveFailures, threshold, entry.TripCount, GetCooldown(entry.TripCount))
		}

	case StateHalfOpen:
		// 试探失败，重新进入 Open 状态，TripCount 递增（冷却时间翻倍）
		entry.State = StateOpen
		entry.TripCount++
		entry.ConsecutiveFailures = 0 // 重新开始计数
		entry.LastFailureTime = time.Now()
		log.Warnf("circuit breaker [%s] HalfOpen -> Open (probe failed, tripCount=%d, cooldown=%v)",
			key, entry.TripCount, GetCooldown(entry.TripCount))

	case StateOpen:
		// 理论上不应该在 Open 状态下接收到失败记录（请求应被拒绝），
		// 但为安全起见仍更新失败时间
		entry.LastFailureTime = time.Now()
	}

	entry.persist()
}

// SetManualDisabled 手动启用/禁用某个渠道+模型的熔断状态。
// disabled=true 时该组合被永久跳过；false 时恢复并清除熔断状态。
func SetManualDisabled(channelID int, modelName, channelName string, disabled bool) {
	entry := getOrCreateEntry(channelID, modelName)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.ChannelName = channelName
	entry.ManualDisabled = disabled
	if !disabled {
		entry.State = StateClosed
		entry.ConsecutiveFailures = 0
		entry.TripCount = 0
	}
	entry.persist()
}

// Reset 取消熔断：清空熔断状态与手动禁用标记，并删除持久化记录。
func Reset(channelID int, modelName string) {
	key := circuitKey(channelID, modelName)
	if v, ok := globalBreaker.Load(key); ok {
		entry := v.(*circuitEntry)
		entry.mu.Lock()
		entry.State = StateClosed
		entry.ConsecutiveFailures = 0
		entry.TripCount = 0
		entry.ManualDisabled = false
		entry.LastErrorTime = 0
		entry.LastError = ""
		entry.mu.Unlock()
		globalBreaker.Delete(key)
	}
	_ = op.CircuitBreakerDelete(channelID, modelName)
}

// DeleteByChannel 渠道删除时清理该渠道的全部熔断记录。
func DeleteByChannel(channelID int) {
	globalBreaker.Range(func(key, value any) bool {
		raw := key.(string)
		idx := strings.IndexByte(raw, ':')
		if idx > 0 {
			var id int
			if _, err := fmt.Sscanf(raw[:idx], "%d", &id); err == nil && id == channelID {
				globalBreaker.Delete(key)
			}
		}
		return true
	})
	_ = op.CircuitBreakerDeleteByChannel(channelID)
}

// List 返回当前所有熔断器状态快照（内存为权威，与 DB 保持同步）。
func List(ctx context.Context) []model.CircuitBreaker {
	records := make([]model.CircuitBreaker, 0)
	globalBreaker.Range(func(key, value any) bool {
		entry := value.(*circuitEntry)
		entry.mu.Lock()
		records = append(records, model.CircuitBreaker{
			ChannelID:           entry.ChannelID,
			ChannelName:         entry.ChannelName,
			ModelName:           entry.ModelName,
			State:               int(entry.State),
			TripCount:           entry.TripCount,
			ConsecutiveFailures: entry.ConsecutiveFailures,
			ManualDisabled:      entry.ManualDisabled,
			LastErrorTime:       entry.LastErrorTime,
			LastError:           entry.LastError,
		})
		entry.mu.Unlock()
		return true
	})
	return records
}

// IsQuotaExhausted 判断错误是否为额度耗尽类错误（配额/额度用完，需要人工处理或长时间冷却）。
// 匹配常见上游错误文案（如百炼 Free quota exhausted、Code Plan 使用上限等）。
func IsQuotaExhausted(statusCode int, msg string) bool {
	if statusCode != 402 && statusCode != 403 && statusCode != 429 {
		return false
	}
	lower := strings.ToLower(msg)
	patterns := []string{
		"free quota exhausted",
		"quota exhausted",
		"quota exceeded",
		"quota has been exhausted",
		"insufficient quota",
		"quota limit reached",
		"quota is exhausted",
		"usage limit",
		"已达上限", "达到上限", "使用上限", "额度已", "额度用", "配额已", "配额用", "限额已", "已耗尽", "已用尽",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

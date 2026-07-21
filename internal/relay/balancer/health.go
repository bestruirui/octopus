package balancer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bestruirui/octopus/internal/utils/log"
)

// HealthStatus 渠道健康状态
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"   // 正常
	HealthDegraded  HealthStatus = "degraded"  // 有失败但未熔断
	HealthUnhealthy HealthStatus = "unhealthy" // 探测失败 / 熔断中
	HealthUnknown   HealthStatus = "unknown"   // 尚无数据
)

// ChannelHealth 渠道健康快照（供 API / 看板使用）
type ChannelHealth struct {
	ChannelID      int          `json:"channel_id"`
	ChannelName    string       `json:"channel_name"`
	Status         HealthStatus `json:"status"`
	LastProbeTime  int64        `json:"last_probe_time"`  // unix 秒
	LastProbeOK    bool         `json:"last_probe_ok"`
	LastProbeDelay int          `json:"last_probe_delay"` // ms
	LastProbeError string       `json:"last_probe_error,omitempty"`
	FailStreak     int          `json:"fail_streak"`
	CircuitOpen    bool         `json:"circuit_open"`
	CircuitRemain  int          `json:"circuit_remain_sec,omitempty"`
	BaseURL        string       `json:"base_url,omitempty"`
}

type probeRecord struct {
	ChannelID   int
	ChannelName string
	BaseURL     string
	OK          bool
	DelayMS     int
	Err         string
	At          time.Time
	FailStreak  int
}

var (
	healthStore sync.Map // channelID(int) -> *probeRecord

	// 可配置阈值（由探活任务/设置热更新）
	// failThreshold: 连续失败多少次判 unhealthy，默认 3
	// degradeMS: 延迟超过多少 ms 判 degraded，默认 5000
	failThreshold atomic.Int64
	degradeMS     atomic.Int64
)

func init() {
	failThreshold.Store(3)
	degradeMS.Store(5000)
}

// SetHealthThresholds 热更新健康判定阈值
func SetHealthThresholds(failThresh, degradeLatencyMS int) {
	if failThresh > 0 {
		failThreshold.Store(int64(failThresh))
	}
	if degradeLatencyMS > 0 {
		degradeMS.Store(int64(degradeLatencyMS))
	}
}

// GetHealthThresholds 供 API 返回当前配置
func GetHealthThresholds() (failThresh, degradeLatencyMS int) {
	return int(failThreshold.Load()), int(degradeMS.Load())
}

// UpdateProbeResult 由探活任务写入
func UpdateProbeResult(channelID int, channelName, baseURL string, ok bool, delayMS int, errMsg string) {
	var streak int
	if v, loaded := healthStore.Load(channelID); loaded {
		prev := v.(*probeRecord)
		if ok {
			streak = 0
		} else {
			streak = prev.FailStreak + 1
		}
	} else if !ok {
		streak = 1
	}

	healthStore.Store(channelID, &probeRecord{
		ChannelID:   channelID,
		ChannelName: channelName,
		BaseURL:     baseURL,
		OK:          ok,
		DelayMS:     delayMS,
		Err:         errMsg,
		At:          time.Now(),
		FailStreak:  streak,
	})

	// Feed probe latency into EWMA so board kRTT can fall back to real RTT
	// when eBPF non-blocking connect cannot measure handshake time.
	if delayMS > 0 {
		RecordLatency(channelID, int64(delayMS), ok)
	}
}

// GetChannelHealth 返回单个渠道健康状态
func GetChannelHealth(channelID int) ChannelHealth {
	v, ok := healthStore.Load(channelID)
	if !ok {
		return ChannelHealth{ChannelID: channelID, Status: HealthUnknown}
	}
	rec := v.(*probeRecord)
	return buildHealth(rec)
}

// ListChannelHealth 返回全部已探测渠道的健康状态
func ListChannelHealth() []ChannelHealth {
	out := make([]ChannelHealth, 0, 16)
	healthStore.Range(func(_, value any) bool {
		rec := value.(*probeRecord)
		out = append(out, buildHealth(rec))
		return true
	})
	return out
}

// AnyCircuitOpen 检查渠道是否有任意 key/model 处于熔断
func AnyCircuitOpen(channelID int) (open bool, remainSec int) {
	prefix := fmt.Sprintf("%d:", channelID)
	globalBreaker.Range(func(key, value any) bool {
		k, ok := key.(string)
		if !ok {
			return true
		}
		if len(k) < len(prefix) || k[:len(prefix)] != prefix {
			return true
		}
		entry := value.(*circuitEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if entry.State == StateOpen {
			cooldown := GetCooldown(entry.TripCount)
			elapsed := time.Since(entry.LastFailureTime)
			if elapsed < cooldown {
				open = true
				r := int((cooldown - elapsed).Seconds())
				if r > remainSec {
					remainSec = r
				}
			}
		}
		return true
	})
	return open, remainSec
}

func buildHealth(rec *probeRecord) ChannelHealth {
	h := ChannelHealth{
		ChannelID:      rec.ChannelID,
		ChannelName:    rec.ChannelName,
		LastProbeTime:  rec.At.Unix(),
		LastProbeOK:    rec.OK,
		LastProbeDelay: rec.DelayMS,
		LastProbeError: rec.Err,
		FailStreak:     rec.FailStreak,
		BaseURL:        rec.BaseURL,
	}

	circuitOpen, remain := AnyCircuitOpen(rec.ChannelID)
	h.CircuitOpen = circuitOpen
	h.CircuitRemain = remain

	ft := int(failThreshold.Load())
	if ft < 1 {
		ft = 3
	}
	dms := int(degradeMS.Load())
	if dms < 1 {
		dms = 5000
	}

	switch {
	case circuitOpen:
		h.Status = HealthUnhealthy
	case !rec.OK && rec.FailStreak >= ft:
		// 连续失败达到阈值才 unhealthy（避免一次抖动误杀）
		h.Status = HealthUnhealthy
	case !rec.OK || rec.FailStreak > 0 || rec.DelayMS > dms:
		h.Status = HealthDegraded
	case rec.OK:
		h.Status = HealthHealthy
	default:
		h.Status = HealthUnknown
	}
	return h
}

// ForceTrip 探活确认挂掉时强制打开熔断（keyID=0, model="__probe__"）
// 默认探活失败不调用此函数；仅 health_probe_trip_on_fail=true 时启用。
func ForceTrip(channelID int, reason string) {
	key := circuitKey(channelID, 0, "__probe__")
	entry := getOrCreateEntry(key)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.LastFailureTime = time.Now()
	if entry.State != StateOpen {
		entry.State = StateOpen
		entry.TripCount++
		log.Warnf("circuit breaker [%s] forced Open by health probe: %s (tripCount=%d, cooldown=%v)",
			key, reason, entry.TripCount, GetCooldown(entry.TripCount))
	}
}

// ForceRecover 探活成功时恢复探活键的熔断状态
func ForceRecover(channelID int) {
	RecordSuccess(channelID, 0, "__probe__")
}

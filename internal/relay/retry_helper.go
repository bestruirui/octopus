package relay

import (
	"context"
	"fmt"
	"time"

	dbmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/relay/balancer"
	"github.com/lingyuins/octopus/internal/utils/log"
)

// PrepareCandidateResult 准备候选的结果
type PrepareCandidateResult struct {
	Channel       *dbmodel.Channel
	UsedKey       dbmodel.ChannelKey
	SkipReason    string                // 不为空表示应跳过该候选
	SkipStatus    dbmodel.AttemptStatus // 跳过时的状态（用于 Iterator 追踪）
	ResolvedModel string                // 解析后的上游模型名
}

// PrepareCandidate 准备候选：熔断检查、Key选择、类型兼容检查
// 这是一个公共函数，供 relay.go 和 media_relay.go 复用
//
// 参数：
//   - ctx: 请求上下文
//   - item: 当前候选 GroupItem
//   - iter: Iterator 用于熔断检查和追踪
//   - ratelimitCooldown: 429 冷却时间（秒）
//   - requestModel: 请求模型名
//   - zenPreferredCheck: Zen 模型前缀类型兼容性检查函数（可选）
//
// 返回：PrepareCandidateResult
func PrepareCandidate(
	ctx context.Context,
	item dbmodel.GroupItem,
	iter *balancer.Iterator,
	ratelimitCooldown int,
	requestModel string,
	zenPreferredCheck func(channelType int) bool,
) PrepareCandidateResult {
	result := PrepareCandidateResult{}

	// 1. 获取通道
	channel, err := op.ChannelGet(item.ChannelID, ctx)
	if err != nil {
		log.Warnf("failed to get channel %d: %v", item.ChannelID, err)
		result.SkipReason = fmt.Sprintf("channel not found: %v", err)
		result.SkipStatus = dbmodel.AttemptSkipped
		return result
	}
	result.Channel = channel

	// 2. 检查通道是否启用
	if !channel.Enabled {
		result.SkipReason = "channel disabled"
		result.SkipStatus = dbmodel.AttemptSkipped
		return result
	}

	// 3. 选择 Key（含 429 cooldown）
	usedKey := channel.GetChannelKeyWithCooldown(ratelimitCooldown)
	if usedKey.ChannelKey == "" {
		result.SkipReason = "no available key"
		result.SkipStatus = dbmodel.AttemptSkipped
		return result
	}
	result.UsedKey = usedKey

	// 4. 解析上游模型名
	resolvedModel := resolveCandidateModelName(requestModel, item)
	if resolvedModel == "" {
		result.SkipReason = "resolved upstream model is empty"
		result.SkipStatus = dbmodel.AttemptSkipped
		return result
	}
	result.ResolvedModel = resolvedModel

	// 5. 熔断检查
	if iter.SkipCircuitBreak(channel.ID, usedKey.ID, channel.Name, resolvedModel) {
		result.SkipReason = "circuit breaker tripped"
		result.SkipStatus = dbmodel.AttemptCircuitBreak
		return result
	}

	// 6. Zen 模型前缀类型兼容性检查（如果提供）
	if zenPreferredCheck != nil && !zenPreferredCheck(int(channel.Type)) {
		result.SkipReason = "channel type not preferred for zen model prefix"
		result.SkipStatus = dbmodel.AttemptSkipped
		return result
	}

	return result
}

// PrepareCandidateForRetry 内层重试时准备 Key
// 与 PrepareCandidate 不同，这里只处理换 Key 和熔断检查
//
// 参数：
//   - channel: 当前通道
//   - failedKeyIDs: 已失败的 Key ID 列表
//   - iter: Iterator
//   - ratelimitCooldown: 429 冷却时间（秒）
//
// 返回：
//   - usedKey: 可用的 Key（如果为空表示无更多 Key）
//   - skipReason: 跳过原因（如果不为空）
func PrepareCandidateForRetry(
	channel *dbmodel.Channel,
	failedKeyIDs []int,
	iter *balancer.Iterator,
	ratelimitCooldown int,
	modelName string,
) (dbmodel.ChannelKey, string) {
	// 换 Key（排除已失败的 + 429 cooldown）
	usedKey := channel.GetChannelKeyExcludingWithCooldown(failedKeyIDs, ratelimitCooldown)
	if usedKey.ChannelKey == "" {
		return dbmodel.ChannelKey{}, "no more keys to retry"
	}

	// 熔断检查
	if iter.SkipCircuitBreak(channel.ID, usedKey.ID, channel.Name, modelName) {
		return usedKey, "circuit breaker tripped on retry key"
	}

	return usedKey, ""
}

// RecordSuccessSideEffects 记录成功后的所有副作用
// 统一处理：Key 状态更新、统计、熔断器重置、Auto 策略、粘性会话
//
// 参数：
//   - channel: 通道
//   - usedKey: 使用的 Key
//   - span: AttemptSpan（用于结束计时）
//   - statusCode: HTTP 状态码
//   - modelName: 上游模型名
//   - apiKeyID: API Key ID（用于粘性会话）
//   - requestModel: 请求模型名
//   - cost: 本次请求成本（可选，媒体端点可能为 0）
func RecordSuccessSideEffects(
	channel *dbmodel.Channel,
	usedKey dbmodel.ChannelKey,
	span *balancer.AttemptSpan,
	statusCode int,
	modelName string,
	apiKeyID int,
	requestModel string,
	cost float64,
) {
	// 更新 Key 状态
	usedKey.StatusCode = statusCode
	usedKey.LastUseTimeStamp = time.Now().Unix()
	usedKey.TotalCost += cost
	op.ChannelKeyUpdate(usedKey)

	// 结束 Span
	span.End(dbmodel.AttemptSuccess, statusCode, "")

	// Channel 维度统计
	op.StatsChannelUpdate(channel.ID, dbmodel.StatsMetrics{
		WaitTime:       span.Duration().Milliseconds(),
		RequestSuccess: 1,
	})

	// 熔断器：记录成功
	balancer.RecordSuccess(channel.ID, usedKey.ID, modelName)

	// Auto 策略：记录成功
	balancer.RecordAutoSuccess(channel.ID, modelName)

	// 粘性会话：更新记录
	balancer.SetSticky(apiKeyID, requestModel, channel.ID, usedKey.ID)
}

// RecordFailureSideEffects 记录失败后的所有副作用
// 根据决策决定是否触发熔断计数（换 Key 不触发熔断，换候选才触发）
//
// 参数：
//   - channel: 通道
//   - usedKey: 使用的 Key
//   - span: AttemptSpan
//   - statusCode: HTTP 状态码
//   - modelName: 上游模型名
//   - decision: 重试决策
//   - tryIndex: 当前尝试索引（用于日志）
//   - tryTotal: 总尝试次数（用于日志）
func RecordFailureSideEffects(
	channel *dbmodel.Channel,
	usedKey dbmodel.ChannelKey,
	span *balancer.AttemptSpan,
	statusCode int,
	modelName string,
	decision RetryDecision,
	tryIndex int,
	tryTotal int,
) {
	// 更新 Key 状态（无论什么决策都更新）
	usedKey.StatusCode = statusCode
	usedKey.LastUseTimeStamp = time.Now().Unix()
	op.ChannelKeyUpdate(usedKey)

	// 构造日志消息
	msg := decision.String()
	if tryTotal > 1 {
		msg = fmt.Sprintf("retry %d/%d failed: %s", tryIndex, tryTotal, msg)
	}

	// 结束 Span
	span.End(dbmodel.AttemptFailed, statusCode, msg)

	// Channel 维度统计
	op.StatsChannelUpdate(channel.ID, dbmodel.StatsMetrics{
		WaitTime:      span.Duration().Milliseconds(),
		RequestFailed: 1,
	})

	// 熔断器和 Auto 策略：只在换候选或停止时记录失败
	// 换 Key 重试不触发熔断计数，避免误熔断（同一 channel 的其他 key 可能正常）
	if decision.Scope == ScopeNextChannel || decision.Scope == ScopeAbortAll {
		balancer.RecordFailure(channel.ID, usedKey.ID, modelName)
		balancer.RecordAutoFailure(channel.ID, modelName)
	}

	// 日志记录决策
	if decision.IsError {
		log.Warnf("channel %s failed: %s (decision: %s)", channel.Name, msg, decision.Scope.String())
	}
}

// IsRetryAllowed 根据决策判断是否允许继续重试
// 返回：
//   - continueRetry: 是否继续重试
//   - switchChannel: 是否切换候选（true = 换候选，false = 换 Key）
func IsRetryAllowed(decision RetryDecision) (continueRetry bool, switchChannel bool) {
	switch decision.Scope {
	case ScopeNone:
		// 不重试，请求结束
		return false, false
	case ScopeSameChannel:
		// 同候选换 Key 重试
		return true, false
	case ScopeNextChannel:
		// 换下一个候选重试
		return true, true
	case ScopeAbortAll:
		// 停止所有重试
		return false, false
	default:
		// 未知决策，保守停止
		return false, false
	}
}

package relay

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/looplj/axonhub/llm"
)

// 客户端请求在转发过程中的当前状态。
type Status string

const (
	StatusRunning   Status = "running"   // 循环中: 正在选目标, 等待或请求上游。
	StatusCommitted Status = "committed" // 首字节已写出客户端, 此后不可再重试。
	StatusSuccess   Status = "success"   // 响应已完整交付客户端。
	StatusFailed    Status = "failed"    // 请求以错误结束。
	StatusCanceled  Status = "canceled"  // 客户端提前断开或取消。
)

// 客户端请求的完整进程内状态, 同时作为状态流的消息形状; 上半部分在请求到达时写入并在结束时定稿, 下半部分每轮循环覆盖。
type RequestState struct {
	ID                uint64        `json:"id"`                  // 请求在当前进程内的唯一标识。
	Status            Status        `json:"status"`              // 请求当前状态。
	StartedAt         time.Time     `json:"started_at"`          // 请求到达时间。
	Duration          time.Duration `json:"duration"`            // 请求总耗时, 未结束时为零。
	FirstTextDuration time.Duration `json:"first_text_duration"` // 首次向客户端写出有效文本的耗时。
	Model             string        `json:"model"`               // 客户端请求的模型名称, 即分组名称。
	Usage             llm.Usage     `json:"usage"`               // 请求结束时写入的展示用量。
	Cost              float64       `json:"cost"`                // 请求结束时写入的累计费用。

	Round         int    `json:"round"`           // 最新一轮循环的递增序号, 人工中止按此匹配以免误杀下一轮。
	TargetChannel string `json:"target_channel"`  // 最新一轮选中的渠道名称。
	TargetModel   string `json:"target_model"`    // 最新一轮实际请求上游的模型名称。
	Sending       bool   `json:"sending"`         // 最新一轮是否仍在等待上游响应。
	Error         string `json:"error,omitempty"` // 最新一轮的失败原因, 请求结束后即为最终错误。

	body         string             // 客户端原始请求体, 体积大故不进状态流, 由独立接口按需拉取。
	responseBody string             // 聚合后的完整最终响应体, 同样按需拉取。
	apiKeyID     int                // 发起请求的 API Key ID, 用于请求完成后的归属统计。
	cancel       context.CancelFunc // 中止最新一轮上游请求, 仅在该轮等待响应期间非空。
}

const streamBuffer = 16 // 单个状态流连接的非阻塞消息缓冲容量。
const maxFinished = 50  // 进程内最多保留的已结束请求数量。

var (
	idSeq    atomic.Uint64                          // 进程内严格递增的请求 ID。
	mu       sync.Mutex                             // 全部共享状态的互斥锁。
	requests = make(map[uint64]*RequestState)       // 按请求 ID 保存的全部请求状态。
	watchers = make(map[chan RequestState]struct{}) // 全部状态流 SSE 连接。
)

// newRequestState 分配请求 ID 并登记初始运行状态; 返回的记录是本请求后续全部状态写入的入口。
func newRequestState(model, body string, apiKeyID int) *RequestState {
	mu.Lock()
	defer mu.Unlock()

	request := &RequestState{
		ID:        idSeq.Add(1),
		Status:    StatusRunning,
		StartedAt: time.Now(),
		Model:     model,
		body:      body,
		apiKeyID:  apiKeyID,
	}
	requests[request.ID] = request
	publishRequestLocked(request)
	return request
}

// startRound 记录本轮选中的目标并进入上游请求, cancel 供人工中止本轮, 返回递增的轮次序号。
func (r *RequestState) startRound(cancel context.CancelFunc, channel, model string) int {
	mu.Lock()
	defer mu.Unlock()

	r.Round++
	r.TargetChannel = channel
	r.TargetModel = model
	r.Sending = true
	r.Error = ""
	r.cancel = cancel
	publishRequestLocked(r)
	return r.Round
}

// finishRound 记录本轮上游结果, errText 为空表示已取得可提交响应。
func (r *RequestState) finishRound(errText string) {
	mu.Lock()
	defer mu.Unlock()

	r.Sending = false
	r.Error = errText
	r.cancel = nil
	publishRequestLocked(r)
}

// Interrupt 中止指定请求仍在等待响应且轮次匹配的上游请求; 轮次不匹配说明该轮已结束, 不影响后续轮次。
func Interrupt(id uint64, round int) {
	mu.Lock()
	request := requests[id]
	if request == nil || request.Round != round || request.cancel == nil {
		mu.Unlock()
		return
	}
	cancel := request.cancel
	request.cancel = nil
	mu.Unlock()

	cancel()
}

// wait 在重新选择目标之前退避 seconds 秒; 客户端在退避期间断开时以取消终态定稿并返回 false。
func (r *RequestState) wait(ctx context.Context, seconds int) bool {
	select {
	case <-ctx.Done():
		r.markCanceled(ctx.Err(), "", nil)
		return false
	case <-time.After(time.Duration(seconds) * time.Second):
		return true
	}
}

// markCommitted 标记响应已提交; 流式响应在此之后仍会持续转发, 故必须先于提交动作调用。
func (r *RequestState) markCommitted() {
	mu.Lock()
	defer mu.Unlock()

	r.Status = StatusCommitted
	publishRequestLocked(r)
}

// markFirstText 记录首个有效文本已经刷新给客户端; 元数据事件不算首字。
func (r *RequestState) markFirstText() {
	mu.Lock()
	defer mu.Unlock()

	if r.FirstTextDuration != 0 {
		return
	}
	r.FirstTextDuration = time.Since(r.StartedAt)
	publishRequestLocked(r)
}

// markSucceeded 以成功终态定稿请求。
func (r *RequestState) markSucceeded(responseBody string, usage *llm.Usage) {
	mu.Lock()
	defer mu.Unlock()

	r.Status = StatusSuccess
	r.Error = ""
	r.responseBody = responseBody
	r.finishLocked(usage)
}

// markFailed 以失败终态定稿请求, 最终错误取自本次失败原因。
func (r *RequestState) markFailed(err error, responseBody string, usage *llm.Usage) {
	mu.Lock()
	defer mu.Unlock()

	r.Status = StatusFailed
	r.Error = err.Error()
	if responseBody != "" {
		r.responseBody = responseBody
	}
	r.finishLocked(usage)
}

// markCanceled 以取消终态定稿请求, 用于客户端提前断开或主动取消。
func (r *RequestState) markCanceled(err error, responseBody string, usage *llm.Usage) {
	mu.Lock()
	defer mu.Unlock()

	r.Status = StatusCanceled
	r.Error = err.Error()
	if responseBody != "" {
		r.responseBody = responseBody
	}
	r.finishLocked(usage)
}

// finishLocked 写入用量和费用, 发布终态, 更新请求级统计并裁剪历史; 调用方必须持有锁。
func (r *RequestState) finishLocked(usage *llm.Usage) {
	r.Sending = false
	r.cancel = nil
	if usage != nil {
		r.Usage = *usage
	}
	metrics := usageMetrics(r.TargetModel, usage)
	r.Cost = metrics.InputCost + metrics.OutputCost
	r.Duration = time.Since(r.StartedAt)
	metrics.WaitTime = r.Duration.Milliseconds()
	if r.Status == StatusSuccess {
		metrics.RequestSuccess = 1
	} else {
		metrics.RequestFailed = 1
	}
	_ = op.StatsTotalUpdate(metrics)
	_ = op.StatsHourlyUpdate(metrics)
	_ = op.StatsDailyUpdate(context.Background(), metrics)
	if r.apiKeyID > 0 {
		_ = op.StatsAPIKeyUpdate(r.apiKeyID, metrics)
	}
	publishRequestLocked(r)

	finished := 0
	oldest := uint64(0)
	for id, request := range requests {
		if request.Status == StatusRunning || request.Status == StatusCommitted {
			continue
		}
		finished++
		if oldest == 0 || id < oldest {
			oldest = id
		}
	}
	if finished > maxFinished {
		delete(requests, oldest)
	}
}

// usageMetrics 将统一用量按模型单价转换为 Token 与费用统计; 无用量或价格时对应费用为零。
func usageMetrics(modelName string, usage *llm.Usage) model.StatsMetrics {
	if usage == nil {
		return model.StatsMetrics{}
	}
	metrics := model.StatsMetrics{InputToken: usage.PromptTokens, OutputToken: usage.CompletionTokens}
	price, err := op.LLMGet(modelName)
	if err != nil {
		return metrics
	}
	cachedTokens, writeCachedTokens := int64(0), int64(0)
	if usage.PromptTokensDetails != nil {
		cachedTokens = usage.PromptTokensDetails.CachedTokens
		writeCachedTokens = usage.PromptTokensDetails.WriteCachedTokens
	}
	inputTokens := max(int64(0), usage.PromptTokens-cachedTokens-writeCachedTokens)
	metrics.InputCost = (float64(inputTokens)*price.Input + float64(cachedTokens)*price.CacheRead + float64(writeCachedTokens)*price.CacheWrite) / 1_000_000
	metrics.OutputCost = float64(usage.CompletionTokens) * price.Output / 1_000_000
	return metrics
}

// publishRequestLocked 非阻塞发布最新请求状态, 连接拥塞时关闭它并交给客户端重连获取全量快照; 调用方必须持有锁。
func publishRequestLocked(request *RequestState) {
	for stream := range watchers {
		select {
		case stream <- *request:
		default:
			delete(watchers, stream)
			close(stream)
		}
	}
}

// OpenRequestStream 注册请求状态流连接, 返回按请求 ID 倒序的全部快照和后续增量通道。
func OpenRequestStream() ([]RequestState, chan RequestState) {
	mu.Lock()
	defer mu.Unlock()

	stream := make(chan RequestState, streamBuffer)
	watchers[stream] = struct{}{}

	snapshot := make([]RequestState, 0, len(requests))
	for _, request := range requests {
		snapshot = append(snapshot, *request)
	}
	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].ID > snapshot[j].ID })
	return snapshot, stream
}

// CloseRequestStream 注销并关闭指定请求状态流连接。
func CloseRequestStream(stream chan RequestState) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := watchers[stream]; exists {
		delete(watchers, stream)
		close(stream)
	}
}

// RequestBody 返回指定请求保存的原始请求体, 记录不存在时返回空串。
func RequestBody(id uint64) string {
	mu.Lock()
	defer mu.Unlock()

	if request := requests[id]; request != nil {
		return request.body
	}
	return ""
}

// ResponseBody 返回指定请求当前保存的响应体, 记录不存在或响应未完成时返回空串。
func ResponseBody(id uint64) string {
	mu.Lock()
	defer mu.Unlock()

	if request := requests[id]; request != nil {
		return request.responseBody
	}
	return ""
}

// Clear 删除全部已结束的请求记录。
func Clear() {
	mu.Lock()
	defer mu.Unlock()

	for id, request := range requests {
		if request.Status != StatusRunning && request.Status != StatusCommitted {
			delete(requests, id)
		}
	}
}

package relay

import (
	"sort"
	"sync"
	"time"

	"github.com/looplj/axonhub/llm"
)

// RequestState 表示客户端请求在 Relay 中的当前状态。
type RequestState string

const (
	RequestStateRunning   RequestState = "running"
	RequestStateCommitted RequestState = "committed"
	RequestStateSuccess   RequestState = "success"
	RequestStateFailed    RequestState = "failed"
	RequestStateCanceled  RequestState = "canceled"
)

// LogEventType 表示 Relay 提交给日志状态的生命周期事件类型。
type LogEventType string

const (
	LogEventRequestStarted    LogEventType = "request.started"
	LogEventAttemptStarted    LogEventType = "attempt.started"
	LogEventAttemptFinished   LogEventType = "attempt.finished"
	LogEventTargetWaiting     LogEventType = "target.waiting"
	LogEventResponseCommitted LogEventType = "response.committed"
	LogEventRequestFinished   LogEventType = "request.finished"
)

// LogAttempt 记录一次渠道尝试。
type LogAttempt struct {
	Type        LogEventType `json:"-"`             // 尝试阶段事件类型。
	Index       int          `json:"attempt_index"` // 当前请求中的尝试序号。
	ChannelName string       `json:"channel_name"`  // 渠道名称。
	ModelName   string       `json:"model_name"`    // 实际请求的模型名称。
	Error       string       `json:"error"`         // 本次尝试的失败原因。
}

// LogOverview 表示概览流中一条可持续更新的请求日志。
type LogOverview struct {
	ID                uint64        `json:"id"`                             // 请求在当前进程内的唯一标识。
	State             RequestState  `json:"state"`                          // 请求当前状态。
	StartedAt         time.Time     `json:"started_at"`                     // 请求到达时间。
	CompletedAt       time.Time     `json:"completed_at"`                   // 请求完成时间。
	Duration          time.Duration `json:"duration"`                       // 请求总耗时。
	RequestModel      string        `json:"request_model"`                  // 客户端请求的模型名称。
	RequestAPIKeyName string        `json:"request_api_key_name,omitempty"` // 请求使用的 API Key 名称。
	ActualModel       string        `json:"actual_model"`                   // 最终实际请求的模型名称。
	ClientProtocol    llm.APIFormat `json:"client_protocol"`                // 客户端使用的请求协议。
	Stream            bool          `json:"stream"`                         // 是否为流式请求。
	FinalChannelName  string        `json:"final_channel_name"`             // 成功渠道或最后尝试渠道的名称。
	InputTokens       int64         `json:"input_tokens"`                   // 请求完成后补充的输入 Token 数量。
	OutputTokens      int64         `json:"output_tokens"`                  // 请求完成后补充的输出 Token 数量。
	CacheReadTokens   int64         `json:"cache_read_tokens"`              // 请求完成后补充的缓存读取 Token 数量。
	CacheWriteTokens  int64         `json:"cache_write_tokens"`             // 请求完成后补充的缓存写入 Token 数量。
	TotalCost         float64       `json:"total_cost"`                     // 请求完成后补充的总费用。
	Error             string        `json:"error,omitempty"`                // 请求当前或最终错误。
}

// LogRecord 保存一个请求的概览和正文快照。
type LogRecord struct {
	LogOverview
	RequestBody    string      `json:"request_body"`  // 客户端原始请求体。
	ResponseBody   string      `json:"response_body"` // 聚合后的完整最终响应体。
	currentAttempt *LogAttempt // 当前仍在执行的渠道尝试(内部游标,未导出)。
}

const logStreamBufferSize = 16
const maxLogHistoryRecords = 50

var (
	logMu              sync.Mutex                   // logMu 保护全部共享日志状态。
	logRecords         = make(map[uint64]LogRecord) // logRecords 按请求 ID 保存全部请求记录。
	logOverview        chan LogOverview             // logOverview 保存最新概览连接的消息通道。
	logDetailRequestID uint64                       // logDetailRequestID 保存最新详情连接订阅的请求 ID。
	logDetail          chan LogAttempt              // logDetail 保存最新详情连接的单条尝试阶段更新通道。
)

// applyLog 应用 Relay 提交的完整生命周期快照并通知当前连接。
func applyLog(eventType LogEventType, record LogRecord, attempt *LogAttempt) {
	logMu.Lock()
	defer logMu.Unlock()

	if attempt != nil {
		attempt.Type = eventType
		switch eventType {
		case LogEventAttemptStarted:
			record.currentAttempt = attempt
		case LogEventAttemptFinished, LogEventResponseCommitted:
			record.currentAttempt = nil
		}
	}
	if eventType == LogEventRequestFinished {
		record.currentAttempt = nil
	}
	logRecords[record.ID] = record
	if eventType == LogEventRequestFinished {
		trimLogRecordsLocked()
	}

	sendLogOverviewLocked(record.LogOverview)
	if logDetail != nil && logDetailRequestID == record.ID {
		if attempt != nil {
			sendLogDetailLocked(*attempt)
		}
		if eventType == LogEventRequestFinished {
			close(logDetail)
			logDetail = nil
			logDetailRequestID = 0
		}
	}
}

// trimLogRecordsLocked 删除 ID 最小的已完成记录，调用方必须持有锁。
func trimLogRecordsLocked() {
	completed := 0
	oldestID := uint64(0)
	for id, record := range logRecords {
		if !isLogFinished(record.State) {
			continue
		}
		completed++
		if oldestID == 0 || id < oldestID {
			oldestID = id
		}
	}
	if completed > maxLogHistoryRecords {
		delete(logRecords, oldestID)
	}
}

// isLogFinished 判断请求是否已经离开运行状态。
func isLogFinished(state RequestState) bool {
	return state == RequestStateSuccess || state == RequestStateFailed || state == RequestStateCanceled
}

// ClearLogs 删除全部完成记录。
func ClearLogs() {
	logMu.Lock()
	defer logMu.Unlock()

	for id, record := range logRecords {
		if isLogFinished(record.State) {
			delete(logRecords, id)
		}
	}
}

// GetLogRequestBody 返回指定请求保存的原始请求体。
func GetLogRequestBody(id uint64) (string, bool) {
	logMu.Lock()
	defer logMu.Unlock()

	record, ok := logRecords[id]
	if !ok {
		return "", false
	}
	return record.RequestBody, true
}

// GetLogResponseBody 返回指定请求当前保存的响应体。
func GetLogResponseBody(id uint64) (string, bool) {
	logMu.Lock()
	defer logMu.Unlock()

	record, ok := logRecords[id]
	if !ok {
		return "", false
	}
	return record.ResponseBody, true
}

// OpenLogOverview 替换当前概览连接，并返回建立连接时的完整快照和消息通道。
func OpenLogOverview() ([]LogOverview, chan LogOverview) {
	logMu.Lock()
	defer logMu.Unlock()

	if logOverview != nil {
		close(logOverview)
	}
	logOverview = make(chan LogOverview, logStreamBufferSize)
	return logOverviewSnapshotLocked(), logOverview
}

// CloseLogOverview 在指定通道仍是当前连接时关闭概览流。
func CloseLogOverview(stream chan LogOverview) {
	logMu.Lock()
	defer logMu.Unlock()

	if logOverview == stream {
		close(logOverview)
		logOverview = nil
	}
}

// OpenLogDetail 查找运行中的请求，替换实时重试连接并补发当前运行尝试。
func OpenLogDetail(id uint64) (chan LogAttempt, bool) {
	logMu.Lock()
	defer logMu.Unlock()

	record, found := logRecords[id]
	if !found || isLogFinished(record.State) {
		return nil, false
	}

	if logDetail != nil {
		close(logDetail)
		logDetail = nil
		logDetailRequestID = 0
	}
	logDetailRequestID = id
	logDetail = make(chan LogAttempt, logStreamBufferSize)
	if attempt := record.currentAttempt; attempt != nil {
		logDetail <- *attempt
	} else if record.State == RequestStateCommitted {
		logDetail <- LogAttempt{Type: LogEventResponseCommitted, ChannelName: record.FinalChannelName}
	}
	return logDetail, true
}

// CloseLogDetail 在指定通道仍是当前连接时关闭详情流。
func CloseLogDetail(stream chan LogAttempt) {
	logMu.Lock()
	defer logMu.Unlock()

	if logDetail == stream {
		close(logDetail)
		logDetail = nil
		logDetailRequestID = 0
	}
}

// logOverviewSnapshotLocked 返回按请求 ID 倒序排列的全部概览；调用方必须持有锁。
func logOverviewSnapshotLocked() []LogOverview {
	overviews := make([]LogOverview, 0, len(logRecords))
	for _, record := range logRecords {
		overviews = append(overviews, record.LogOverview)
	}
	sort.Slice(overviews, func(i, j int) bool {
		return overviews[i].ID > overviews[j].ID
	})
	return overviews
}

// sendLogOverviewLocked 非阻塞发送概览，拥塞时关闭连接并交给客户端重连获取最新状态；调用方必须持有锁。
func sendLogOverviewLocked(message LogOverview) {
	if logOverview == nil {
		return
	}
	select {
	case logOverview <- message:
		return
	default:
	}

	close(logOverview)
	logOverview = nil
}

// sendLogDetailLocked 非阻塞发送单条尝试阶段更新，拥塞时只保留最新状态；调用方必须持有锁。
func sendLogDetailLocked(update LogAttempt) {
	select {
	case logDetail <- update:
		return
	default:
	}

	for {
		select {
		case <-logDetail:
			continue
		default:
		}
		break
	}
	logDetail <- update
}

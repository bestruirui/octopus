package handlers

import (
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/bestruirui/octopus/internal/relay"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/log").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/overview/stream", http.MethodGet).
				Handle(streamOverview),
		).
		AddRoute(
			router.NewRoute("/:id/request-body", http.MethodGet).
				Handle(getRequestBody),
		).
		AddRoute(
			router.NewRoute("/:id/response-body", http.MethodGet).
				Handle(getResponseBody),
		).
		AddRoute(
			router.NewRoute("/:request_id/:round/stop", http.MethodPost).
				Handle(interruptRound),
		).
		AddRoute(
			router.NewRoute("/clear", http.MethodDelete).
				Handle(clearLog),
		)
}

// interruptRound 中止请求当前轮次匹配的上游调用。
func interruptRound(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("request_id"), 10, 64)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid request id")
		return
	}
	round, err := strconv.Atoi(c.Param("round"))
	if err != nil || round < 1 {
		resp.Error(c, http.StatusBadRequest, "invalid round")
		return
	}
	relay.Interrupt(requestID, round)
	c.Status(http.StatusNoContent)
}

// clearLog 删除全部已完成的请求记录，并在释放记录引用后主动执行垃圾回收。
func clearLog(c *gin.Context) {
	relay.Clear()
	runtime.GC()
	c.Status(http.StatusNoContent)
}

// getRequestBody 返回指定请求的原始请求体。
func getRequestBody(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid request id")
		return
	}
	resp.Success(c, relay.RequestBody(id))
}

// getResponseBody 返回指定请求当前保存的响应体。
func getResponseBody(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid request id")
		return
	}
	resp.Success(c, relay.ResponseBody(id))
}

// streamOverview 逐条发送建立连接时的概览及后续请求更新。
func streamOverview(c *gin.Context) {
	prepareSSE(c)
	snapshot, updates := relay.OpenRequestStream()
	defer relay.CloseRequestStream(updates)
	if len(snapshot) == 0 {
		// Flush a real SSE comment so proxies forward the empty-state response immediately.
		if _, err := c.Writer.Write([]byte(": connected\n\n")); err != nil {
			return
		}
		c.Writer.Flush()
	}
	for _, request := range snapshot {
		if err := sse.Encode(c.Writer, sse.Event{Event: "log", Data: request}); err != nil {
			return
		}
		c.Writer.Flush()
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := c.Writer.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			c.Writer.Flush()
		case request, ok := <-updates:
			if !ok {
				return
			}
			if err := sse.Encode(c.Writer, sse.Event{Event: "log", Data: request}); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

// prepareSSE 设置实时日志连接需要的响应头。
func prepareSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativePassthroughOrdinaryResponsesPreservesUnknownFieldsAndMapsModel(t *testing.T) {
	requestBody := []byte(`{
  "model": "gpt-test",
  "stream": false,
  "input": [{"type":"future_unknown_item","foo":{"bar":1}}],
  "metadata": {"trace":"kept"},
  "prompt_cache_key": "cache-key"
}`)
	responseBody := []byte(`{"id":"resp_1","object":"response","created_at":1700000000,"output":[],"usage":{"input_tokens":12,"input_tokens_details":{"cached_tokens":5},"output_tokens":3,"output_tokens_details":{"reasoning_tokens":2},"total_tokens":15}}`)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/responses", r.URL.Path)
		require.Equal(t, "Bearer upstream-secret", r.Header.Get("Authorization"))
		require.Equal(t, "codex-test", r.Header.Get("X-Openai-Subagent"))
		require.Empty(t, r.Header.Get("Connection"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var actual map[string]any
		require.NoError(t, json.Unmarshal(body, &actual))
		assert.Equal(t, "upstream-model", actual["model"])
		assert.Equal(t, []any{map[string]any{
			"type": "future_unknown_item",
			"foo":  map[string]any{"bar": float64(1)},
		}}, actual["input"])
		assert.Equal(t, map[string]any{"trace": "kept"}, actual["metadata"])
		assert.Equal(t, "cache-key", actual["prompt_cache_key"])
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Codex-Turn-State", "turn-state")
		_, _ = w.Write(responseBody)
	}))
	defer upstream.Close()

	attempt, recorder := newNativeTestAttempt(t, upstream.URL, "/responses", requestBody)
	attempt.rawRequest.Headers.Set("Authorization", "Bearer client-octopus-key")
	attempt.rawRequest.Headers.Set("X-OpenAI-Subagent", "codex-test")
	attempt.rawRequest.Headers.Set("Connection", "keep-alive")
	status, err := attempt.forwardPassthrough()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, responseBody, recorder.Body.Bytes())
	assert.Equal(t, "turn-state", recorder.Header().Get("X-Codex-Turn-State"))
	assert.EqualValues(t, 12, attempt.metrics.Stats.InputToken)
	assert.EqualValues(t, 3, attempt.metrics.Stats.OutputToken)
}

func TestNativePassthroughRemoteCompactionV2AndUnknownSSEEvent(t *testing.T) {
	requestBody := []byte(`{
  "model":"test-model",
  "stream":true,
  "input":[
    {"role":"user","content":"remember A17"},
    {"type":"compaction_trigger"}
  ]
}`)
	sse := "data: {\"type\":\"response.created\",\"sequence_number\":0,\"response\":{\"id\":\"resp_compact\",\"created_at\":1700000000}}\n\n" +
		"event: response.future_event\ndata: {\"type\":\"response.future_event\",\"future\":{\"kept\":true}}\n\n" +
		"data: {\"type\":\"response.output_item.done\",\"sequence_number\":1,\"item\":{\"id\":\"cmp_1\",\"type\":\"compaction\",\"encrypted_content\":\"TEST\"}}\n\n" +
		"data: {\"type\":\"response.completed\",\"sequence_number\":2,\"response\":{\"id\":\"resp_compact\",\"created_at\":1700000000,\"output\":[{\"id\":\"cmp_1\",\"type\":\"compaction\",\"encrypted_content\":\"TEST\"}],\"usage\":{\"input_tokens\":100,\"input_tokens_details\":{\"cached_tokens\":40},\"output_tokens\":8,\"output_tokens_details\":{\"reasoning_tokens\":6},\"total_tokens\":108}}}\n\n"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var request struct {
			Input []map[string]any `json:"input"`
		}
		require.NoError(t, json.Unmarshal(body, &request))
		require.Equal(t, "compaction_trigger", request.Input[len(request.Input)-1]["type"])
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sse)
	}))
	defer upstream.Close()

	attempt, recorder := newNativeTestAttempt(t, upstream.URL, "/responses", requestBody)
	status, err := attempt.forwardPassthrough()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, sse, recorder.Body.String())
	assert.Contains(t, recorder.Body.String(), `"type":"compaction","encrypted_content":"TEST"`)
	assert.Contains(t, recorder.Body.String(), `"type":"response.future_event"`)
	assert.EqualValues(t, 100, attempt.metrics.Stats.InputToken)
	assert.EqualValues(t, 8, attempt.metrics.Stats.OutputToken)
	assert.EqualValues(t, 40, attempt.metrics.CachedToken)
	assert.EqualValues(t, 6, attempt.metrics.ReasoningToken)
	assert.False(t, attempt.metrics.FirstTokenTime.IsZero())
}

func TestNativePassthroughAlphaSearch(t *testing.T) {
	requestBody := []byte(`{"id":"session","model":"gpt-test","commands":{"weather":[{"location":"US, CA, San Francisco"}]},"settings":{"external_web_access":true}}`)
	responseBody := []byte(`{"encrypted_output":"ciphertext","output":"sunny","results":[{"type":"weather_result","future_field":true}]}`)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/alpha/search", r.URL.Path)
		require.Equal(t, "turn-metadata", r.Header.Get("X-Codex-Turn-Metadata"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var actual map[string]any
		require.NoError(t, json.Unmarshal(body, &actual))
		assert.Equal(t, "upstream-model", actual["model"])
		assert.NotNil(t, actual["commands"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(responseBody)
	}))
	defer upstream.Close()

	attempt, recorder := newNativeTestAttempt(t, upstream.URL, "/alpha/search", requestBody)
	attempt.rawRequest.Headers.Set("X-Codex-Turn-Metadata", "turn-metadata")
	status, err := attempt.forwardPassthrough()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, responseBody, recorder.Body.Bytes())
}

func TestNativePassthroughLegacyResponsesCompact(t *testing.T) {
	requestBody := []byte(`{"model":"gpt-test","input":[{"role":"user","content":"remember A17"}],"unknown_option":{"kept":true}}`)
	responseBody := []byte(`{"id":"cmp_resp","object":"response.compaction","output":[{"type":"compaction","encrypted_content":"LEGACY"}]}`)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/responses/compact", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"unknown_option":{"kept":true}`)
		assert.Contains(t, string(body), `"model":"upstream-model"`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(responseBody)
	}))
	defer upstream.Close()

	attempt, recorder := newNativeTestAttempt(t, upstream.URL, "/responses/compact", requestBody)
	status, err := attempt.forwardPassthrough()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, responseBody, recorder.Body.Bytes())
}

func TestNativePassthroughFailoverBeforeStreamStarts(t *testing.T) {
	requestBody := []byte(`{"model":"gpt-test","stream":true,"input":"hello"}`)
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporary failure", http.StatusBadGateway)
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_b\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n")
	}))
	defer second.Close()

	attemptA, recorder := newNativeTestAttempt(t, first.URL, "/responses", requestBody)
	_, err := attemptA.forwardPassthrough()
	require.Error(t, err)
	assert.False(t, attemptA.c.Writer.Written())

	attemptB := cloneNativeAttemptForUpstream(attemptA, second.URL)
	_, err = attemptB.forwardPassthrough()
	require.NoError(t, err)
	assert.True(t, attemptB.c.Writer.Written())
	assert.Contains(t, recorder.Body.String(), `"id":"resp_b"`)
}

func TestNativePassthroughDoesNotFailoverAfterStreamStarts(t *testing.T) {
	requestBody := []byte(`{"model":"gpt-test","stream":true,"input":"hello"}`)
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_a\"}}\n\n")
	}))
	defer first.Close()
	var secondCalls atomic.Int64
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		secondCalls.Add(1)
		_, _ = fmt.Fprint(w, `{"id":"resp_b"}`)
	}))
	defer second.Close()

	attemptA, recorder := newNativeTestAttempt(t, first.URL, "/responses", requestBody)
	_, err := attemptA.forwardPassthrough()
	require.Error(t, err)
	assert.True(t, attemptA.c.Writer.Written())

	if !attemptA.c.Writer.Written() {
		attemptB := cloneNativeAttemptForUpstream(attemptA, second.URL)
		_, _ = attemptB.forwardPassthrough()
	}
	assert.Zero(t, secondCalls.Load())
	assert.Contains(t, recorder.Body.String(), `"id":"resp_a"`)
	assert.NotContains(t, recorder.Body.String(), `"id":"resp_b"`)
}

func TestNativePassthroughRequiresResponsesChannelOptIn(t *testing.T) {
	run := &relayRun{inboundType: llm.APIFormatOpenAIResponse}
	assert.False(t, run.nativePassthroughEnabled(&model.Channel{
		Type:              llm.APIFormatOpenAIResponse,
		NativePassthrough: false,
	}))
	assert.False(t, run.nativePassthroughEnabled(&model.Channel{
		Type:              llm.APIFormatAnthropicMessage,
		NativePassthrough: true,
	}))
	assert.True(t, run.nativePassthroughEnabled(&model.Channel{
		Type:              llm.APIFormatOpenAIResponse,
		NativePassthrough: true,
	}))
}

func TestNativePassthroughUnknownResponseItemIsPreserved(t *testing.T) {
	body := []byte(`{"model":"gpt-test","input":[{"type":"future_unknown_item","foo":{"bar":1}}]}`)
	attempt, _ := newNativeTestAttempt(t, "http://127.0.0.1:1", "/responses", body)
	request, err := attempt.buildPassthroughRequest(t.Context())
	require.NoError(t, err)
	encoded, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":"upstream-model","input":[{"type":"future_unknown_item","foo":{"bar":1}}]}`, string(encoded))
}

func TestNativePassthroughUnknownSSEEventIsByteTransparent(t *testing.T) {
	stream := "event: response.future_event\r\ndata: {\"type\":\"response.future_event\",\"opaque\":true}\r\n\r\n"
	observer := newPassthroughObserver()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	writer := &passthroughStreamWriter{writer: context.Writer, observer: observer}

	n, err := writer.Write([]byte(stream))
	require.NoError(t, err)
	assert.Equal(t, len(stream), n)
	assert.Equal(t, stream, recorder.Body.String())
}

func TestNativePassthroughUsageObserverRecordsCachedAndReasoningTokens(t *testing.T) {
	observer := newPassthroughObserver()
	observer.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":20,\"input_tokens_details\":{\"cached_tokens\":7},\"output_tokens\":4,\"output_tokens_details\":{\"reasoning_tokens\":3},\"total_tokens\":24}}}\n\n"))

	require.NotNil(t, observer.usage)
	assert.EqualValues(t, 20, observer.usage.PromptTokens)
	assert.EqualValues(t, 7, observer.usage.PromptTokensDetails.CachedTokens)
	assert.EqualValues(t, 4, observer.usage.CompletionTokens)
	assert.EqualValues(t, 3, observer.usage.CompletionTokensDetails.ReasoningTokens)
}

func newNativeTestAttempt(t *testing.T, upstreamURL, path string, body []byte) (*relayAttempt, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/v1"+path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	context.Request = request
	rawRequest, err := httpclient.ReadHTTPRequest(request)
	require.NoError(t, err)

	var options *PassthroughOptions
	if path != "/responses" {
		options = &PassthroughOptions{UpstreamPath: path, ExtractModel: true, RewriteModel: true}
	}
	internalRequest := &llm.Request{Model: "gpt-test", RequestType: llm.RequestTypeChat, RawRequest: rawRequest}
	run := &relayRun{
		c:               context,
		inboundType:     llm.APIFormatOpenAIResponse,
		internalRequest: internalRequest,
		rawRequest:      rawRequest,
		requestModel:    "gpt-test",
		passthrough:     options,
		group:           model.Group{FirstTokenTimeOut: 2},
		metrics: &RelayMetrics{
			RequestModel:    "gpt-test",
			ActualModel:     "upstream-model",
			StartTime:       time.Now(),
			InternalRequest: internalRequest,
			RawRequestBody:  append([]byte(nil), body...),
		},
	}
	attempt := &relayAttempt{
		relayRun: run,
		channel: &model.Channel{
			ID:                1,
			Name:              "test-channel",
			Type:              llm.APIFormatOpenAIResponse,
			NativePassthrough: true,
			BaseUrls:          []model.BaseUrl{{URL: upstreamURL + "/v1"}},
		},
		usedKey:       model.ChannelKey{ID: 1, ChannelID: 1, ChannelKey: "upstream-secret"},
		native:        true,
		upstreamModel: "upstream-model",
	}
	return attempt, recorder
}

func cloneNativeAttemptForUpstream(source *relayAttempt, upstreamURL string) *relayAttempt {
	clone := *source
	channel := *source.channel
	channel.BaseUrls = []model.BaseUrl{{URL: upstreamURL + "/v1"}}
	clone.channel = &channel
	return &clone
}

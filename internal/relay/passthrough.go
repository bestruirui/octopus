package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
	"github.com/looplj/axonhub/llm/transformer"
	"github.com/tidwall/sjson"
)

// PassthroughOptions controls a raw JSON relay endpoint. Only routing fields are
// inspected; all other request fields and response bytes remain opaque.
type PassthroughOptions struct {
	UpstreamPath string
	ExtractModel bool
	RewriteModel bool
}

// PassthroughHandler creates an OpenAI Responses-compatible raw relay endpoint.
func PassthroughHandler(options PassthroughOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		run, err := newRelayRun(c, llm.APIFormatOpenAIResponse, nil, &options)
		if err != nil {
			return
		}
		run.run()
	}
}

type routingEnvelope struct {
	Model string `json:"model"`
}

func parseRoutingRequest(c *gin.Context, extractModel bool) (*httpclient.Request, string, error) {
	rawRequest, err := httpclient.ReadHTTPRequest(c.Request)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return nil, "", err
	}
	if !extractModel {
		return rawRequest, "", nil
	}

	var envelope routingEnvelope
	if err := json.Unmarshal(rawRequest.Body, &envelope); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return nil, "", err
	}
	if strings.TrimSpace(envelope.Model) == "" {
		err := errors.New("model is required")
		resp.Error(c, http.StatusBadRequest, err.Error())
		return nil, "", err
	}
	return rawRequest, envelope.Model, nil
}

func (ra *relayAttempt) forwardPassthrough() (int, error) {
	ctx := ra.c.Request.Context()
	request, err := ra.buildPassthroughRequest(ctx)
	if err != nil {
		return 0, err
	}

	httpClient, err := helper.ChannelHttpClient(ra.channel)
	if err != nil {
		return 0, err
	}
	upstreamResponse, err := httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer upstreamResponse.Body.Close()

	if upstreamResponse.StatusCode < http.StatusOK || upstreamResponse.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(upstreamResponse.Body)
		if readErr != nil {
			return upstreamResponse.StatusCode, fmt.Errorf("upstream returned %d and response read failed: %w", upstreamResponse.StatusCode, readErr)
		}
		return upstreamResponse.StatusCode, &passthroughUpstreamError{
			StatusCode: upstreamResponse.StatusCode,
			Body:       body,
		}
	}

	contentType := strings.ToLower(upstreamResponse.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") {
		return upstreamResponse.StatusCode, ra.forwardPassthroughSSE(ctx, upstreamResponse)
	}
	return upstreamResponse.StatusCode, ra.forwardPassthroughResponse(upstreamResponse)
}

func (ra *relayAttempt) buildPassthroughRequest(ctx context.Context) (*http.Request, error) {
	if ra.rawRequest == nil {
		return nil, errors.New("missing raw request")
	}
	options := PassthroughOptions{
		UpstreamPath: "/responses",
		ExtractModel: true,
		RewriteModel: true,
	}
	if ra.passthrough != nil {
		options = *ra.passthrough
	}

	body := append([]byte(nil), ra.rawRequest.Body...)
	if options.RewriteModel {
		rewrittenBody, rewriteErr := sjson.SetBytes(body, "model", ra.upstreamModel)
		if rewriteErr != nil {
			return nil, fmt.Errorf("failed to rewrite model: %w", rewriteErr)
		}
		body = rewrittenBody
	}

	headers := cloneHeaders(ra.rawRequest.Headers)
	removeHopByHopHeaders(headers)
	headers.Del("Authorization")
	headers.Del("X-Api-Key")
	headers.Del("Host")
	headers.Del("Content-Length")

	requestOptions := &httpclient.Request{
		Headers:     headers,
		Body:        body,
		ContentType: headers.Get("Content-Type"),
	}
	ra.applyChannelRequestOptions(requestOptions)
	body = requestOptions.Body
	headers = requestOptions.Headers
	headers.Set("Authorization", "Bearer "+ra.usedKey.ChannelKey)
	headers.Del("Content-Length")

	upstreamURL, err := buildPassthroughURL(channelBaseURL(ra.channel.GetBaseUrl()), options.UpstreamPath, ra.c.Request.URL.Query())
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header = headers
	ra.metrics.RawRequestBody = append(ra.metrics.RawRequestBody[:0], body...)
	return request, nil
}

func channelBaseURL(baseURL string) string {
	return strings.TrimSpace(baseURL)
}

func buildPassthroughURL(baseURL, upstreamPath string, query url.Values) (string, error) {
	if baseURL == "" {
		return "", errors.New("channel base URL is empty")
	}
	if !strings.HasPrefix(upstreamPath, "/") {
		return "", fmt.Errorf("upstream path must start with /: %s", upstreamPath)
	}

	var target string
	if strings.HasSuffix(baseURL, "##") {
		target = strings.TrimSuffix(baseURL, "##")
		parsed, err := url.Parse(target)
		if err != nil {
			return "", err
		}
		if !strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), upstreamPath) {
			return "", fmt.Errorf("raw channel URL does not match passthrough path %s", upstreamPath)
		}
	} else {
		target = transformer.NormalizeBaseURL(baseURL, "v1") + upstreamPath
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	upstreamQuery := parsed.Query()
	for key, values := range query {
		for _, value := range values {
			upstreamQuery.Add(key, value)
		}
	}
	parsed.RawQuery = upstreamQuery.Encode()
	return parsed.String(), nil
}

func (ra *relayAttempt) forwardPassthroughResponse(upstream *http.Response) error {
	body, err := io.ReadAll(upstream.Body)
	if err != nil {
		return err
	}
	observer := newPassthroughObserver()
	observer.observeJSON(body)
	ra.recordPassthroughObservation(observer, body)
	copyResponseHeaders(ra.c.Writer.Header(), upstream.Header)
	ra.c.Data(upstream.StatusCode, upstream.Header.Get("Content-Type"), body)
	return nil
}

func (ra *relayAttempt) forwardPassthroughSSE(ctx context.Context, upstream *http.Response) error {
	firstChunk, err := readFirstChunk(ctx, upstream.Body, ra.group.FirstTokenTimeOut)
	if err != nil {
		return err
	}

	copyResponseHeaders(ra.c.Writer.Header(), upstream.Header)
	ra.c.Writer.Header().Del("Content-Length")
	ra.c.Writer.WriteHeader(upstream.StatusCode)
	ra.metrics.FirstTokenTime = time.Now()

	observer := newPassthroughObserver()
	writer := &passthroughStreamWriter{writer: ra.c.Writer, observer: observer}
	if _, err := writer.Write(firstChunk); err != nil {
		return err
	}
	_, copyErr := io.Copy(writer, upstream.Body)
	observer.finish()
	ra.recordPassthroughObservation(observer, nil)
	if copyErr != nil {
		return fmt.Errorf("upstream stream interrupted: %w", copyErr)
	}
	if observer.sawFailed {
		return errors.New("upstream emitted response.failed")
	}
	if strings.HasSuffix(ra.passthroughPath(), "/responses") && !observer.sawCompleted {
		return errors.New("upstream stream closed before response.completed")
	}
	return nil
}

func (ra *relayAttempt) passthroughPath() string {
	if ra.passthrough != nil {
		return ra.passthrough.UpstreamPath
	}
	return "/responses"
}

func readFirstChunk(ctx context.Context, reader io.Reader, timeoutSeconds int) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	results := make(chan result, 1)
	go func() {
		buffer := make([]byte, 32*1024)
		n, err := reader.Read(buffer)
		results <- result{data: buffer[:n], err: err}
	}()

	var timer *time.Timer
	var timeout <-chan time.Time
	if timeoutSeconds > 0 {
		timer = time.NewTimer(time.Duration(timeoutSeconds) * time.Second)
		timeout = timer.C
		defer timer.Stop()
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timeout:
		return nil, fmt.Errorf("first token timeout (%ds)", timeoutSeconds)
	case result := <-results:
		if len(result.data) == 0 {
			if result.err == nil {
				return nil, errors.New("upstream stream returned no bytes")
			}
			return nil, fmt.Errorf("upstream stream returned no bytes: %w", result.err)
		}
		if result.err != nil && !errors.Is(result.err, io.EOF) {
			return nil, result.err
		}
		return result.data, nil
	}
}

type passthroughStreamWriter struct {
	writer   gin.ResponseWriter
	observer *passthroughObserver
}

func (w *passthroughStreamWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	w.observer.Write(data)
	n, err := w.writer.Write(data)
	w.writer.Flush()
	return n, err
}

type passthroughUsage struct {
	InputTokens       int64 `json:"input_tokens"`
	OutputTokens      int64 `json:"output_tokens"`
	TotalTokens       int64 `json:"total_tokens"`
	InputTokenDetails struct {
		CachedTokens     int64 `json:"cached_tokens"`
		CacheWriteTokens int64 `json:"cache_write_tokens"`
	} `json:"input_tokens_details"`
	OutputTokenDetails struct {
		ReasoningTokens int64 `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

func (u *passthroughUsage) toLLMUsage() *llm.Usage {
	if u == nil {
		return nil
	}
	return &llm.Usage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      u.TotalTokens,
		PromptTokensDetails: &llm.PromptTokensDetails{
			CachedTokens:      u.InputTokenDetails.CachedTokens,
			WriteCachedTokens: u.InputTokenDetails.CacheWriteTokens,
		},
		CompletionTokensDetails: &llm.CompletionTokensDetails{
			ReasoningTokens: u.OutputTokenDetails.ReasoningTokens,
		},
	}
}

type passthroughObserver struct {
	buffer        []byte
	usage         *llm.Usage
	completedData []byte
	lastData      []byte
	sawCompleted  bool
	sawFailed     bool
}

func newPassthroughObserver() *passthroughObserver {
	return &passthroughObserver{buffer: make([]byte, 0, 4096)}
}

func (o *passthroughObserver) Write(data []byte) {
	o.buffer = append(o.buffer, data...)
	for {
		index, separatorLength := nextSSEFrame(o.buffer)
		if index < 0 {
			return
		}
		frame := append([]byte(nil), o.buffer[:index]...)
		o.buffer = o.buffer[index+separatorLength:]
		o.observeSSEFrame(frame)
	}
}

func (o *passthroughObserver) finish() {
	if len(bytes.TrimSpace(o.buffer)) > 0 {
		o.observeSSEFrame(o.buffer)
	}
	o.buffer = nil
}

func nextSSEFrame(data []byte) (int, int) {
	lf := bytes.Index(data, []byte("\n\n"))
	crlf := bytes.Index(data, []byte("\r\n\r\n"))
	switch {
	case lf < 0:
		return crlf, 4
	case crlf < 0:
		return lf, 2
	case lf < crlf:
		return lf, 2
	default:
		return crlf, 4
	}
}

func (o *passthroughObserver) observeSSEFrame(frame []byte) {
	lines := strings.Split(strings.ReplaceAll(string(frame), "\r\n", "\n"), "\n")
	dataLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if len(dataLines) == 0 {
		return
	}
	payload := []byte(strings.Join(dataLines, "\n"))
	if bytes.Equal(bytes.TrimSpace(payload), []byte("[DONE]")) {
		return
	}
	o.lastData = append(o.lastData[:0], payload...)

	var event struct {
		Type     string `json:"type"`
		Response struct {
			Usage *passthroughUsage `json:"usage"`
		} `json:"response"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return
	}
	switch event.Type {
	case "response.completed":
		o.sawCompleted = true
		o.completedData = append(o.completedData[:0], payload...)
		o.usage = event.Response.Usage.toLLMUsage()
	case "response.failed", "response.incomplete":
		o.sawFailed = true
	}
}

func (o *passthroughObserver) observeJSON(body []byte) {
	var response struct {
		Usage    *passthroughUsage `json:"usage"`
		Response *struct {
			Usage *passthroughUsage `json:"usage"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return
	}
	usage := response.Usage
	if usage == nil && response.Response != nil {
		usage = response.Response.Usage
	}
	o.usage = usage.toLLMUsage()
}

func (ra *relayAttempt) recordPassthroughObservation(observer *passthroughObserver, fallback []byte) {
	if observer == nil {
		return
	}
	ra.metrics.RecordUsage(observer.usage)
	switch {
	case len(observer.completedData) > 0:
		ra.metrics.InternalResponse = append(ra.metrics.InternalResponse[:0], observer.completedData...)
	case len(fallback) > 0:
		ra.metrics.InternalResponse = append(ra.metrics.InternalResponse[:0], fallback...)
	case len(observer.lastData) > 0:
		ra.metrics.InternalResponse = append(ra.metrics.InternalResponse[:0], observer.lastData...)
	}
}

type passthroughUpstreamError struct {
	StatusCode int
	Body       []byte
}

func (e *passthroughUpstreamError) Error() string {
	body := strings.TrimSpace(string(e.Body))
	if body == "" {
		return fmt.Sprintf("upstream returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("upstream returned HTTP %d: %s", e.StatusCode, body)
}

func cloneHeaders(source http.Header) http.Header {
	result := make(http.Header, len(source))
	for key, values := range source {
		result[key] = append([]string(nil), values...)
	}
	return result
}

func removeHopByHopHeaders(headers http.Header) {
	for _, value := range headers.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			headers.Del(strings.TrimSpace(token))
		}
	}
	for _, key := range []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Proxy-Connection",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		headers.Del(key)
	}
}

func copyResponseHeaders(destination, source http.Header) {
	filtered := cloneHeaders(source)
	removeHopByHopHeaders(filtered)
	for key, values := range filtered {
		destination.Del(key)
		for _, value := range values {
			destination.Add(key, value)
		}
	}
}

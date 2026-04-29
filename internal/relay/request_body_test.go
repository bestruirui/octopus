package relay

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	dbmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/transformer/inbound"
)

func TestParseRequestRejectsOversizeBody(t *testing.T) {
	originalLimit := maxRelayJSONBodyBytes
	maxRelayJSONBodyBytes = 32
	defer func() {
		maxRelayJSONBodyBytes = originalLimit
	}()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, _, err := parseRequest(inbound.InboundTypeOpenAIChat, ctx)
	if !errors.Is(err, errRelayRequestBodyTooLarge) {
		t.Fatalf("parseRequest() error = %v, want %v", err, errRelayRequestBodyTooLarge)
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestExtractModelFromJSONRejectsOversizeBody(t *testing.T) {
	originalLimit := maxRelayJSONBodyBytes
	maxRelayJSONBodyBytes = 16
	defer func() {
		maxRelayJSONBodyBytes = originalLimit
	}()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-1"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	_, _, _, err := extractModelFromJSON(ctx)
	if !errors.Is(err, errRelayRequestBodyTooLarge) {
		t.Fatalf("extractModelFromJSON() error = %v, want %v", err, errRelayRequestBodyTooLarge)
	}
}

func TestExtractModelFromMultipartRejectsOversizeBody(t *testing.T) {
	originalLimit := maxRelayMultipartBodyBytes
	maxRelayMultipartBodyBytes = 64
	defer func() {
		maxRelayMultipartBodyBytes = originalLimit
	}()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "whisper-1"); err != nil {
		t.Fatalf("write model field: %v", err)
	}
	part, err := writer.CreateFormFile("file", "audio.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, strings.NewReader(strings.Repeat("a", 256))); err != nil {
		t.Fatalf("write file body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(body.Bytes()))
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())

	_, _, _, err = extractModelFromMultipart(ctx)
	if !errors.Is(err, errRelayRequestBodyTooLarge) {
		t.Fatalf("extractModelFromMultipart() error = %v, want %v", err, errRelayRequestBodyTooLarge)
	}
}

func TestForwardMediaRequestMultipartRewritesModelAndStreamsFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotModel string
	var gotFileBody string
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse upstream multipart form: %v", err)
		}
		gotModel = r.FormValue("model")
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("read upstream file: %v", err)
		}
		defer file.Close()

		payload, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read upstream file body: %v", err)
		}
		gotFileBody = string(payload)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"ok"}`))
	}))
	defer upstream.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "whisper-1"); err != nil {
		t.Fatalf("write model field: %v", err)
	}
	if err := writer.WriteField("language", "zh"); err != nil {
		t.Fatalf("write language field: %v", err)
	}
	part, err := writer.CreateFormFile("file", "audio.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, strings.NewReader("hello audio")); err != nil {
		t.Fatalf("write file body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(body.Bytes()))
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())

	modelName, _, streamRequested, err := extractModelFromMultipart(ctx)
	if err != nil {
		t.Fatalf("extractModelFromMultipart() error = %v", err)
	}
	if modelName != "whisper-1" {
		t.Fatalf("modelName = %q, want whisper-1", modelName)
	}
	if streamRequested {
		t.Fatal("streamRequested = true, want false")
	}
	if ctx.Request.MultipartForm != nil {
		defer ctx.Request.MultipartForm.RemoveAll()
	}

	status, err := forwardMediaRequestMultipart(
		ctx,
		getMediaEndpointConfig(MediaEndpointAudioTranscription),
		&dbmodel.Channel{BaseUrls: []dbmodel.BaseUrl{{URL: upstream.URL}}},
		"sk-test",
		"whisper-1",
		"whisper-1-rewritten",
		false,
	)
	if err != nil {
		t.Fatalf("forwardMediaRequestMultipart() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("Authorization = %q, want Bearer sk-test", gotAuth)
	}
	if gotModel != "whisper-1-rewritten" {
		t.Fatalf("upstream model = %q, want whisper-1-rewritten", gotModel)
	}
	if gotFileBody != "hello audio" {
		t.Fatalf("upstream file body = %q, want hello audio", gotFileBody)
	}
	if recorder.Body.String() != `{"text":"ok"}` {
		t.Fatalf("response body = %q, want %q", recorder.Body.String(), `{"text":"ok"}`)
	}
}

func TestExtractModelFromJSONReturnsStreamFlag(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-1","stream":true}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	modelName, body, streamRequested, err := extractModelFromJSON(ctx)
	if err != nil {
		t.Fatalf("extractModelFromJSON() error = %v", err)
	}
	if modelName != "gpt-image-1" {
		t.Fatalf("modelName = %q, want %q", modelName, "gpt-image-1")
	}
	if !streamRequested {
		t.Fatal("streamRequested = false, want true")
	}
	if string(body) != `{"model":"gpt-image-1","stream":true}` {
		t.Fatalf("body = %q, want original payload", string(body))
	}
}

func TestBuildMediaUpstreamURLKeepsSingleV1Prefix(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{
			name:    "base url already includes v1",
			baseURL: "https://api.example.com/v1",
			path:    "/v1/rerank",
			want:    "https://api.example.com/v1/rerank",
		},
		{
			name:    "nested base path already includes v1",
			baseURL: "https://api.example.com/openai/v1/",
			path:    "/v1/images/generations",
			want:    "https://api.example.com/openai/v1/images/generations",
		},
		{
			name:    "base url without path keeps endpoint prefix",
			baseURL: "https://api.example.com",
			path:    "/v1/search",
			want:    "https://api.example.com/v1/search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildMediaUpstreamURL(tt.baseURL, tt.path)
			if err != nil {
				t.Fatalf("buildMediaUpstreamURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("buildMediaUpstreamURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleSSEResponseFlushesLines(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("event: message\n" + "data: {\"ok\":true}\n\n")),
	}
	response.Header.Set("Content-Type", "text/event-stream")

	status, err := handleSSEResponse(ctx, response)
	if err != nil {
		t.Fatalf("handleSSEResponse() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if !recorder.Flushed {
		t.Fatal("recorder.Flushed = false, want true")
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if recorder.Body.String() != "event: message\n"+"data: {\"ok\":true}\n\n" {
		t.Fatalf("body = %q, want original SSE payload", recorder.Body.String())
	}
}

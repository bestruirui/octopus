package helper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	appmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/transformer/outbound"
)

func TestBuildGroupProbeRequest_ConversationEndpointsUseMessages(t *testing.T) {
	for _, endpointType := range []string{
		appmodel.EndpointTypeAll,
		appmodel.EndpointTypeChat,
		appmodel.EndpointTypeDeepSeek,
		appmodel.EndpointTypeResponses,
		appmodel.EndpointTypeMessages,
	} {
		req, err := buildGroupProbeRequest(endpointType, "gpt-4o-mini")
		if err != nil {
			t.Fatalf("buildGroupProbeRequest(%q) error = %v", endpointType, err)
		}
		if req.EmbeddingInput != nil {
			t.Fatalf("buildGroupProbeRequest(%q) embedding_input = %#v, want nil", endpointType, req.EmbeddingInput)
		}
		if len(req.Messages) != 1 {
			t.Fatalf("buildGroupProbeRequest(%q) messages len = %d, want 1", endpointType, len(req.Messages))
		}
		if req.Messages[0].Content.Content == nil || *req.Messages[0].Content.Content != "hi" {
			t.Fatalf("buildGroupProbeRequest(%q) message content = %#v, want hi", endpointType, req.Messages[0].Content.Content)
		}
		if req.Stream == nil || *req.Stream {
			t.Fatalf("buildGroupProbeRequest(%q) stream = %#v, want false", endpointType, req.Stream)
		}
	}
}

func TestBuildGroupProbeRequest_EmbeddingsUseInput(t *testing.T) {
	req, err := buildGroupProbeRequest(appmodel.EndpointTypeEmbeddings, "text-embedding-3-small")
	if err != nil {
		t.Fatalf("buildGroupProbeRequest() error = %v", err)
	}
	if len(req.Messages) != 0 {
		t.Fatalf("buildGroupProbeRequest() messages len = %d, want 0", len(req.Messages))
	}
	if req.EmbeddingInput == nil || req.EmbeddingInput.Single == nil || *req.EmbeddingInput.Single != "hi" {
		t.Fatalf("buildGroupProbeRequest() embedding_input = %#v, want single hi", req.EmbeddingInput)
	}
}

func TestBuildGroupProbeRequest_RejectsUnsupportedEndpoint(t *testing.T) {
	if _, err := buildGroupProbeRequest(appmodel.EndpointTypeImageGeneration, "gpt-image-1"); err == nil {
		t.Fatal("buildGroupProbeRequest() error = nil, want unsupported endpoint error")
	}
}

func TestValidateGroupProbeChannelType_RejectsMismatchedEmbeddingChannel(t *testing.T) {
	err := validateGroupProbeChannelType(appmodel.EndpointTypeEmbeddings, outbound.OutboundTypeOpenAIChat)
	if err == nil {
		t.Fatal("validateGroupProbeChannelType() error = nil, want mismatch error")
	}
}

func TestSendGroupProbeRequest_EmbeddingsUseEmbeddingPayload(t *testing.T) {
	type outboundRequest struct {
		Model string `json:"model"`
		Input string `json:"input"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("request path = %q, want /embeddings", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("Authorization = %q, want Bearer sk-test", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var got outboundRequest
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if got.Input != "hi" {
			t.Fatalf("input = %q, want hi", got.Input)
		}
		if got.Model != "text-embedding-3-small" {
			t.Fatalf("model = %q, want text-embedding-3-small", got.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"emb-1","object":"list","created":1,"model":"text-embedding-3-small","data":[{"object":"embedding","index":0,"embedding":[0.1]}]}`))
	}))
	defer server.Close()

	channel := &appmodel.Channel{
		Type:     outbound.OutboundTypeOpenAIEmbedding,
		BaseUrls: []appmodel.BaseUrl{{URL: server.URL}},
	}

	statusCode, responseText, err := sendGroupProbeRequest(
		context.Background(),
		outbound.Get(outbound.OutboundTypeOpenAIEmbedding),
		channel,
		"sk-test",
		appmodel.EndpointTypeEmbeddings,
		"text-embedding-3-small",
	)
	if err != nil {
		t.Fatalf("sendGroupProbeRequest() error = %v", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("sendGroupProbeRequest() status = %d, want 200", statusCode)
	}
	if responseText == "" {
		t.Fatal("sendGroupProbeRequest() responseText = empty, want upstream payload")
	}
}

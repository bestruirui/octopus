package openai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/transformer/model"
)

func TestCompactResponseOutbound_TransformRequest(t *testing.T) {
	outbound := NewCompactResponseOutbound()
	hello := "hello"
	req, err := outbound.TransformRequest(context.Background(), &model.InternalLLMRequest{
		Model: "gpt-4.1",
		Messages: []model.Message{
			{
				Role: "user",
				Content: model.MessageContent{
					Content: &hello,
				},
			},
		},
		TransformerMetadata: map[string]string{
			"previous_response_id": "resp_prev",
			"prompt_cache_key":     "cache_key",
		},
	}, "https://api.openai.com/v1", "test-key")
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}

	if got, want := req.URL.String(), "https://api.openai.com/v1/responses/compact"; got != want {
		t.Fatalf("url = %s, want %s", got, want)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `"previous_response_id":"resp_prev"`) {
		t.Fatalf("request body missing previous_response_id: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"prompt_cache_key":"cache_key"`) {
		t.Fatalf("request body missing prompt_cache_key: %s", bodyStr)
	}
}

func TestCompactResponseOutbound_TransformResponse_Passthrough(t *testing.T) {
	outbound := NewCompactResponseOutbound()
	raw := `{"object":"response.compaction","id":"resp_cpt","created_at":1}`
	resp, err := outbound.TransformResponse(context.Background(), &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(raw)),
	})
	if err != nil {
		t.Fatalf("TransformResponse() error = %v", err)
	}
	if string(resp.RawResponse) != raw {
		t.Fatalf("RawResponse = %s, want %s", string(resp.RawResponse), raw)
	}
}

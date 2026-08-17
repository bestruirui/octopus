package relay

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

func TestImageProtocols(t *testing.T) {
	tests := []struct {
		name   string
		format llm.APIFormat
		route  string
	}{
		{name: "generation", format: llm.APIFormatOpenAIImageGeneration, route: "/images/generations"},
		{name: "edit", format: llm.APIFormatOpenAIImageEdit, route: "/images/edits"},
		{name: "variation", format: llm.APIFormatOpenAIImageVariation, route: "/images/variations"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			protocol := newImageProtocol(tc.format)
			if protocol.format != tc.format {
				t.Fatalf("format = %q, want %q", protocol.format, tc.format)
			}
			if protocol.route != tc.route {
				t.Fatalf("route = %q, want %q", protocol.route, tc.route)
			}
			if protocol.authType != httpclient.AuthTypeBearer {
				t.Fatalf("auth type = %q, want bearer", protocol.authType)
			}
			if protocol.inbound == nil {
				t.Fatal("inbound transformer is nil")
			}
			if !protocol.omitBodyLog {
				t.Fatal("image protocol must omit request and response bodies from logs")
			}
		})
	}
}

func TestImageGenerationProtocolParsesRequest(t *testing.T) {
	protocol := newImageProtocol(llm.APIFormatOpenAIImageGeneration)
	request, err := protocol.inbound.TransformRequest(t.Context(), &httpclient.Request{
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Body:    []byte(`{"model":"gpt-image-1","prompt":"draw an octopus"}`),
	})
	if err != nil {
		t.Fatalf("TransformRequest() error = %v", err)
	}
	if request.Model != "gpt-image-1" {
		t.Fatalf("model = %q, want gpt-image-1", request.Model)
	}
	if request.RequestType != llm.RequestTypeImage {
		t.Fatalf("request type = %q, want %q", request.RequestType, llm.RequestTypeImage)
	}
	if request.APIFormat != llm.APIFormatOpenAIImageGeneration {
		t.Fatalf("API format = %q, want %q", request.APIFormat, llm.APIFormatOpenAIImageGeneration)
	}
}

func TestImageGenerationConvertedRoundTrip(t *testing.T) {
	type observedRequest struct {
		path          string
		authorization string
		body          string
	}
	observed := make(chan observedRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		observed <- observedRequest{
			path:          r.URL.Path,
			authorization: r.Header.Get("Authorization"),
			body:          string(body),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":1713833628,"data":[{"url":"https://example.com/generated.png"}]}`))
	}))
	defer upstream.Close()

	request := &relayRequest{raw: &httpclient.Request{
		Method:  http.MethodPost,
		Headers: http.Header{"Content-Type": []string{"application/json"}},
		Body:    []byte(`{"model":"public-image","prompt":"draw an octopus"}`),
	}}
	channel := &model.Channel{
		Type:    model.ChannelProviderOpenAI,
		BaseURL: upstream.URL + "/v1",
		Key:     "upstream-secret",
	}

	result := (&forwarder{
		protocol: newImageProtocol(llm.APIFormatOpenAIImageGeneration),
		request:  request,
		client:   upstream.Client(),
	}).executeUpstream(context.Background(), "gpt-image-1", channel)
	if result.err != nil {
		t.Fatalf("executeUpstream() error = %v", result.err)
	}
	if result.response == nil {
		t.Fatal("executeUpstream() response is nil")
	}
	defer result.response.Close()

	got := <-observed
	if got.path != "/v1/images/generations" {
		t.Fatalf("upstream path = %q, want /v1/images/generations", got.path)
	}
	if got.authorization != "Bearer upstream-secret" {
		t.Fatalf("Authorization = %q, want bearer upstream key", got.authorization)
	}
	if !strings.Contains(got.body, `"model":"gpt-image-1"`) {
		t.Fatalf("upstream body does not contain routed model: %s", got.body)
	}

	buffered, ok := result.response.(*bufferedResponse)
	if !ok {
		t.Fatalf("response type = %T, want *bufferedResponse", result.response)
	}
	if !strings.Contains(string(buffered.body), "https://example.com/generated.png") {
		t.Fatalf("response body does not contain generated image URL: %s", buffered.body)
	}
}

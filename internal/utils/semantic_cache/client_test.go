package semantic_cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEmbeddingClient_CreateEmbedding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer srv.Close()

	client := NewEmbeddingClient(RuntimeConfig{
		EmbeddingBaseURL: srv.URL,
		EmbeddingModel:   "text-embedding-3-small",
		EmbeddingTimeout: 5 * time.Second,
	})

	got, err := client.CreateEmbedding(context.Background(), "hello")
	if err != nil {
		t.Fatalf("CreateEmbedding() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("embedding length = %d", len(got))
	}
}

func TestEmbeddingClient_CreateEmbedding_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer srv.Close()

	client := NewEmbeddingClient(RuntimeConfig{
		EmbeddingBaseURL: srv.URL,
		EmbeddingModel:   "text-embedding-3-small",
		EmbeddingTimeout: 5 * time.Second,
	})

	_, err := client.CreateEmbedding(context.Background(), "hello")
	if err == nil {
		t.Fatal("CreateEmbedding() error = nil, want upstream error")
	}
	if !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "bad gateway") {
		t.Fatalf("CreateEmbedding() error = %v, want status code and body", err)
	}
}

func TestEmbeddingClient_CreateEmbedding_EmptyEmbedding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[]}]}`))
	}))
	defer srv.Close()

	client := NewEmbeddingClient(RuntimeConfig{
		EmbeddingBaseURL: srv.URL,
		EmbeddingModel:   "text-embedding-3-small",
		EmbeddingTimeout: 5 * time.Second,
	})

	_, err := client.CreateEmbedding(context.Background(), "hello")
	if err == nil {
		t.Fatal("CreateEmbedding() error = nil, want missing embedding error")
	}
	if !strings.Contains(err.Error(), "missing data[0].embedding") {
		t.Fatalf("CreateEmbedding() error = %v, want missing embedding message", err)
	}
}

package semantic_cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type EmbeddingClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

func NewEmbeddingClient(cfg RuntimeConfig) *EmbeddingClient {
	timeout := cfg.EmbeddingTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &EmbeddingClient{
		baseURL: strings.TrimRight(cfg.EmbeddingBaseURL, "/"),
		apiKey:  cfg.EmbeddingAPIKey,
		model:   cfg.EmbeddingModel,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *EmbeddingClient) CreateEmbedding(ctx context.Context, text string) ([]float64, error) {
	body, err := json.Marshal(embeddingRequest{
		Model: c.model,
		Input: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return nil, fmt.Errorf("embedding upstream error: %d: %s", resp.StatusCode, string(payload))
	}

	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedding response missing data[0].embedding")
	}

	return parsed.Data[0].Embedding, nil
}

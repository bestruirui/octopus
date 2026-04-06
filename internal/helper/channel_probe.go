package helper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	appmodel "github.com/bestruirui/octopus/internal/model"
)

type ChannelTestResult struct {
	BaseURL     string `json:"base_url"`
	KeyRemark   string `json:"key_remark,omitempty"`
	KeyMasked   string `json:"key_masked,omitempty"`
	StatusCode  int    `json:"status_code"`
	Passed      bool   `json:"passed"`
	LatencyMS   int64  `json:"latency_ms"`
	Message     string `json:"message,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`
}

type ChannelTestSummary struct {
	Passed  bool                `json:"passed"`
	Results []ChannelTestResult `json:"results"`
}

func TestChannel(ctx context.Context, request appmodel.Channel) (*ChannelTestSummary, error) {
	client, err := ChannelHttpClient(&request)
	if err != nil {
		return nil, err
	}

	baseURLs := make([]string, 0, len(request.BaseUrls))
	for _, item := range request.BaseUrls {
		url := strings.TrimSpace(item.URL)
		if url != "" {
			baseURLs = append(baseURLs, url)
		}
	}
	if len(baseURLs) == 0 {
		return nil, fmt.Errorf("at least one base url is required")
	}

	keys := make([]appmodel.ChannelKey, 0, len(request.Keys))
	for _, key := range request.Keys {
		if strings.TrimSpace(key.ChannelKey) == "" {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("at least one api key is required")
	}

	summary := &ChannelTestSummary{Results: make([]ChannelTestResult, 0, len(baseURLs)*len(keys))}
	for _, baseURL := range baseURLs {
		for _, key := range keys {
			result := ChannelTestResult{
				BaseURL:   baseURL,
				KeyRemark: strings.TrimSpace(key.Remark),
				KeyMasked: maskSecret(key.ChannelKey),
			}
			startedAt := time.Now()
			statusCode, bodyText, testErr := performChannelTestRequest(ctx, client, request, baseURL, key.ChannelKey)
			result.LatencyMS = time.Since(startedAt).Milliseconds()
			result.StatusCode = statusCode
			result.ResponseBody = bodyText
			result.Passed = statusCode == http.StatusOK || statusCode == http.StatusTooManyRequests
			if testErr != nil {
				result.Message = testErr.Error()
			} else if result.Passed {
				result.Message = "ok"
			}
			if result.Passed {
				summary.Passed = true
			}
			summary.Results = append(summary.Results, result)
		}
	}

	return summary, nil
}

func performChannelTestRequest(ctx context.Context, client *http.Client, request appmodel.Channel, baseURL, apiKey string) (int, string, error) {
	if request.Type == 3 {
		return performGeminiConnectivityRequest(ctx, client, request, baseURL, apiKey)
	}
	return performOpenAICompatibleConnectivityRequest(ctx, client, request, baseURL, apiKey)
}

func performOpenAICompatibleConnectivityRequest(ctx context.Context, client *http.Client, request appmodel.Channel, baseURL, apiKey string) (int, string, error) {
	url := strings.TrimRight(baseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	for _, header := range request.CustomHeader {
		if strings.TrimSpace(header.HeaderKey) != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}
	return doChannelProbeRequest(client, req)
}

func performGeminiConnectivityRequest(ctx context.Context, client *http.Client, request appmodel.Channel, baseURL, apiKey string) (int, string, error) {
	url := strings.TrimRight(baseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("X-Goog-Api-Key", apiKey)
	for _, header := range request.CustomHeader {
		if strings.TrimSpace(header.HeaderKey) != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}
	return doChannelProbeRequest(client, req)
}

func doChannelProbeRequest(client *http.Client, req *http.Request) (int, string, error) {
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyText := strings.TrimSpace(string(body))
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTooManyRequests {
		return resp.StatusCode, bodyText, nil
	}
	if bodyText == "" {
		bodyText = resp.Status
	}
	return resp.StatusCode, bodyText, fmt.Errorf("upstream error: %d", resp.StatusCode)
}

func maskSecret(secret string) string {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 8 {
		return trimmed
	}
	return trimmed[:4] + "..." + trimmed[len(trimmed)-4:]
}

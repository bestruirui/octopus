package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bestruirui/octopus/internal/model"
)

func FetchModel(ctx context.Context, request model.Channel) ([]string, error) {
	client, err := NewHTTPClient(request.Proxy)
	if err != nil {
		return nil, err
	}
	base_url := fmt.Sprintf("%s/v1/models", request.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base_url, nil)
	if err != nil {
		return nil, err
	}
	switch request.Type {
	case model.ChannelTypeOpenAIChat, model.ChannelTypeOpenAIResponse, model.ChannelTypeOneAPI:
		req.Header.Set("Authorization", "Bearer "+request.Key)
	case model.ChannelTypeAnthropic:
		req.Header.Set("Authorization", "Bearer "+request.Key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var models []string
	for _, model := range result.Data {
		models = append(models, model.ID)
	}
	return models, nil
}

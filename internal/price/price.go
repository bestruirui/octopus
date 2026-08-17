package price

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/client"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/charmbracelet/log"
)

const defaultLLMPriceURL = "https://models.dev/api.json"

// developerFamilies 定义研发商及其自研模型系列前缀。
var developerFamilies = map[string][]string{
	"openai":     {"gpt", "o"},
	"anthropic":  {"claude"},
	"google":     {"gemini", "gemma", "lyria", "veo"},
	"deepseek":   {"deepseek"},
	"xai":        {"grok"},
	"alibaba":    {"qwen", "qvq"},
	"zhipuai":    {"glm"},
	"minimax":    {"minimax"},
	"moonshotai": {"kimi"},
	"v0":         {"v0"},
}

var lastUpdateTime time.Time // lastUpdateTime 记录最近一次成功更新时间。

// UpdateLLMPrice 从 models.dev 更新自研文本输出模型的价格。
func UpdateLLMPrice(ctx context.Context) error {
	log.Debugf("update LLM price task started")
	startTime := time.Now()
	defer func() {
		log.Debugf("update LLM price task finished, update time: %s", time.Since(startTime))
	}()
	var body []byte
	httpClient, err := client.GetHTTPClientSystemProxy(false)
	if err == nil {
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, configuredLLMPriceURL(), nil)
		if requestErr != nil {
			return requestErr
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		resp, requestErr := httpClient.Do(req)
		if requestErr != nil {
			err = requestErr
		} else {
			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("failed to fetch LLM info: %s", resp.Status)
			} else {
				body, err = io.ReadAll(resp.Body)
				if err != nil {
					err = fmt.Errorf("failed to read response body: %w", err)
				}
			}
			resp.Body.Close()
		}
	}
	if err != nil {
		log.Warnf("direct request failed, trying with proxy: %v", err)
		httpClient, err = client.GetHTTPClientSystemProxy(true)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, configuredLLMPriceURL(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch LLM info: %s", resp.Status)
		}
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
	}
	var rawPrice map[string]struct {
		Models map[string]struct {
			ID         string `json:"id"`     // ID 是模型标识。
			Family     string `json:"family"` // Family 是模型所属系列。
			Modalities struct {
				Output []string `json:"output"` // Output 是模型支持的输出类型。
			} `json:"modalities"`
			Cost model.LLMPrice `json:"cost"` // Cost 是模型价格。
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &rawPrice); err != nil {
		return fmt.Errorf("failed to parse LLM info: %w", err)
	}
	llmPriceLock.Lock()
	for provider, familyPrefixes := range developerFamilies {
		for _, priceModel := range rawPrice[provider].Models {
			modelID := strings.ToLower(priceModel.ID)
			modelFamily := strings.ToLower(priceModel.Family)

			// 仅保留包含文本输出的非嵌入模型。
			if modelID == "" || !slices.Contains(priceModel.Modalities.Output, "text") || strings.Contains(modelID, "embed") || strings.Contains(modelFamily, "embed") {
				continue
			}

			// 云平台可能同时托管第三方模型，仅接受该研发商的自研系列。
			isDeveloperModel := false
			for _, familyPrefix := range familyPrefixes {
				if strings.HasPrefix(modelFamily, familyPrefix) {
					isDeveloperModel = true
					break
				}
			}
			if !isDeveloperModel {
				continue
			}

			llmPrice[modelID] = priceModel.Cost
		}
	}
	llmPriceLock.Unlock()
	lastUpdateTime = time.Now()
	return nil
}

func configuredLLMPriceURL() string {
	value, err := op.SettingGetString(model.SettingKeyModelPriceURL)
	if err != nil {
		return defaultLLMPriceURL
	}
	return selectLLMPriceURL(value)
}

func selectLLMPriceURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultLLMPriceURL
	}
	return value
}

// GetLastUpdateTime 返回最近一次价格更新时间。
func GetLastUpdateTime() time.Time {
	return lastUpdateTime
}

// GetLLMPrice 返回指定模型的自定义价格或预设价格。
func GetLLMPrice(modelName string) *model.LLMPrice {
	modelName = strings.ToLower(modelName)
	price, err := op.LLMGet(modelName)
	if err == nil {
		return &price
	}
	llmPriceLock.RLock()
	defer llmPriceLock.RUnlock()
	price, ok := llmPrice[modelName]
	if !ok {
		return nil
	}
	return &price
}

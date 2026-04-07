package anthropic

import (
	"github.com/lingyuins/octopus/internal/transformer/model"
)

func convertToAnthropicCacheControl(c *model.CacheControl) *CacheControl {
	if c == nil {
		return nil
	}

	return &CacheControl{
		Type: c.Type,
		TTL:  c.TTL,
	}
}

func convertToLLMCacheControl(c *CacheControl) *model.CacheControl {
	if c == nil {
		return nil
	}

	return &model.CacheControl{
		Type: c.Type,
		TTL:  c.TTL,
	}
}

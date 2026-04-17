package op

import (
	"context"
	"fmt"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/utils/cache"
)

var apiKeyCache = cache.New[int, model.APIKey](16)
var apiKeyIDMap = cache.New[string, int](16)

func APIKeyCreate(key *model.APIKey, ctx context.Context) error {
	if err := db.GetDB().WithContext(ctx).Create(key).Error; err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	apiKeyCache.Set(key.ID, *key)
	apiKeyIDMap.Set(key.APIKey, key.ID)
	return nil
}

func APIKeyUpdate(key *model.APIKey, ctx context.Context) error {
	existing, ok := apiKeyCache.Get(key.ID)
	if !ok {
		return fmt.Errorf("API key not found")
	}
	if err := db.GetDB().WithContext(ctx).Omit("api_key").Save(key).Error; err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}
	key.APIKey = existing.APIKey
	apiKeyCache.Set(key.ID, *key)
	return nil
}

func APIKeyList(ctx context.Context) ([]model.APIKey, error) {
	keys := make([]model.APIKey, 0, apiKeyCache.Len())
	for _, apiKey := range apiKeyCache.GetAll() {
		keys = append(keys, apiKey)
	}
	return keys, nil
}

func APIKeyGet(id int, ctx context.Context) (model.APIKey, error) {
	apiKey, ok := apiKeyCache.Get(id)
	if !ok {
		return model.APIKey{}, fmt.Errorf("API key not found")
	}
	return apiKey, nil
}

func APIKeyGetByAPIKey(apiKey string, ctx context.Context) (model.APIKey, error) {
	id, ok := apiKeyIDMap.Get(apiKey)
	if !ok {
		return model.APIKey{}, fmt.Errorf("API key not found")
	}
	return APIKeyGet(id, ctx)
}

func APIKeyDelete(id int, ctx context.Context) error {
	apiKey, err := APIKeyGet(id, ctx)
	if err != nil {
		return err
	}
	tx := db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	result := tx.Delete(&model.APIKey{ID: id})
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("API key not found")
	}
	if err := tx.Where("api_key_id = ?", id).Delete(&model.StatsAPIKey{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete stats API key: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit API key deletion: %w", err)
	}
	statsAPIKeyMutationLock.Lock()
	statsAPIKeyCache.Del(id)
	statsAPIKeyCacheNeedUpdateLock.Lock()
	delete(statsAPIKeyCacheNeedUpdate, id)
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	statsAPIKeyMutationLock.Unlock()
	apiKeyCache.Del(id)
	apiKeyIDMap.Del(apiKey.APIKey)
	return nil
}

func apiKeyRefreshCache(ctx context.Context) error {
	apiKeys := []model.APIKey{}
	if err := db.GetDB().WithContext(ctx).Find(&apiKeys).Error; err != nil {
		return err
	}
	apiKeyCache.Clear()
	apiKeyIDMap.Clear()
	for _, apiKey := range apiKeys {
		apiKeyCache.Set(apiKey.ID, apiKey)
		apiKeyIDMap.Set(apiKey.APIKey, apiKey.ID)
	}
	return nil
}

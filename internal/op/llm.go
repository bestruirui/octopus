package op

import (
	"context"
	"fmt"
	"strings"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"github.com/charmbracelet/log"
	"gorm.io/gorm/clause"
)

var llmModelCache = cache.New[string, model.LLMPrice](16) // llmModelCache 保存数据库中的模型价格。

// channelModels 读取渠道模型列表，数据库读取失败时返回空列表以继续清理流程。
func channelModels(ctx context.Context) []model.LLMChannel {
	channelLLMs, err := ChannelLLMList(ctx)
	if err != nil {
		log.Warnf("failed to list channel models: %v", err)
		return nil
	}
	return channelLLMs
}

// LLMList 返回缓存中的全部模型价格。
func LLMList() []model.LLMInfo {
	models := make([]model.LLMInfo, 0, llmModelCache.Len())
	for m, cost := range llmModelCache.GetAll() {
		models = append(models, model.LLMInfo{
			Name:     m,
			LLMPrice: cost,
		})
	}
	return models
}

// LLMUpdate 更新已经存在的模型价格并同步缓存。
func LLMUpdate(model model.LLMInfo, ctx context.Context) error {
	_, ok := llmModelCache.Get(model.Name)
	if !ok {
		return fmt.Errorf("model not found")
	}
	if err := db.GetDB().WithContext(ctx).Save(model).Error; err != nil {
		return err
	}
	llmModelCache.Set(model.Name, model.LLMPrice)
	return nil
}

// LLMDelete 删除未被任何渠道引用的模型价格。
func LLMDelete(modelName string, ctx context.Context) error {
	_, ok := llmModelCache.Get(modelName)
	if !ok {
		return fmt.Errorf("model not found")
	}
	for _, channelModel := range channelModels(ctx) {
		if strings.ToLower(channelModel.Name) == modelName {
			return fmt.Errorf("model is referenced by channel")
		}
	}
	if err := db.GetDB().WithContext(ctx).Delete(&model.LLMInfo{Name: modelName}).Error; err != nil {
		return err
	}
	llmModelCache.Del(modelName)
	return nil
}

// LLMCleanupGhosts 删除已经不被任何渠道引用的模型价格。
func LLMCleanupGhosts(ctx context.Context) error {
	channelModels := channelModels(ctx)
	referencedModelNames := make(map[string]struct{}, len(channelModels))
	// 价格表使用小写模型名作为键，渠道模型名转换为相同键后再判断引用关系。
	for _, channelModel := range channelModels {
		referencedModelNames[strings.ToLower(channelModel.Name)] = struct{}{}
	}

	ghostModelNames := make([]string, 0)
	for modelName := range llmModelCache.GetAll() {
		if _, ok := referencedModelNames[modelName]; !ok {
			ghostModelNames = append(ghostModelNames, modelName)
		}
	}
	if len(ghostModelNames) == 0 {
		return nil
	}
	if err := db.GetDB().WithContext(ctx).Where("name IN ?", ghostModelNames).Delete(&model.LLMInfo{}).Error; err != nil {
		return err
	}
	llmModelCache.Del(ghostModelNames...)
	return nil
}

// LLMCreate 写入已在外部入口规范化的模型价格。
func LLMCreate(model model.LLMInfo, ctx context.Context) error {
	_, ok := llmModelCache.Get(model.Name)
	if ok {
		return fmt.Errorf("model already exists")
	}
	if err := db.GetDB().WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}
	llmModelCache.Set(model.Name, model.LLMPrice)
	return nil
}

// LLMBatchCreate 批量写入已规范化且去重的模型价格，并跳过已有模型。
func LLMBatchCreate(llmInfos []model.LLMInfo, ctx context.Context) error {
	if len(llmInfos) == 0 {
		return nil
	}
	newLLMInfos := make([]model.LLMInfo, 0, len(llmInfos))
	for _, llmInfo := range llmInfos {
		if _, ok := llmModelCache.Get(llmInfo.Name); ok {
			continue
		}
		newLLMInfos = append(newLLMInfos, llmInfo)
	}
	if len(newLLMInfos) == 0 {
		return nil
	}
	if err := db.GetDB().WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&newLLMInfos).Error; err != nil {
		return err
	}
	names := make([]string, len(newLLMInfos))
	for i, llmInfo := range newLLMInfos {
		names[i] = llmInfo.Name
	}
	var savedLLMInfos []model.LLMInfo
	if err := db.GetDB().WithContext(ctx).Where("name IN ?", names).Find(&savedLLMInfos).Error; err != nil {
		return err
	}
	for _, llmInfo := range savedLLMInfos {
		llmModelCache.Set(llmInfo.Name, llmInfo.LLMPrice)
	}
	return nil
}

// LLMBatchSave 批量更新或新增模型价格，并在写入成功后同步价格缓存。
func LLMBatchSave(llmInfos []model.LLMInfo, ctx context.Context) error {
	if len(llmInfos) == 0 {
		return nil
	}
	if err := db.GetDB().WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&llmInfos).Error; err != nil {
		return err
	}
	for _, llmInfo := range llmInfos {
		llmModelCache.Set(llmInfo.Name, llmInfo.LLMPrice)
	}
	return nil
}

// LLMGet 按价格表统一使用的小写模型名读取数据库价格缓存。
func LLMGet(name string) (model.LLMPrice, error) {
	price, ok := llmModelCache.Get(strings.ToLower(name))
	if !ok {
		return model.LLMPrice{}, fmt.Errorf("model not found")
	}
	return price, nil
}

// llmRefreshCache 从数据库刷新模型价格缓存。
func llmRefreshCache(ctx context.Context) error {
	models := []model.LLMInfo{}
	if err := db.GetDB().WithContext(ctx).Find(&models).Error; err != nil {
		return err
	}
	for _, model := range models {
		llmModelCache.Set(model.Name, model.LLMPrice)
	}
	return nil
}

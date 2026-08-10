package op

import (
	"context"
	"fmt"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm/clause"
)

// CircuitBreakerList 返回所有熔断器持久化记录（供管理界面展示）。
func CircuitBreakerList(ctx context.Context) ([]model.CircuitBreaker, error) {
	records := make([]model.CircuitBreaker, 0)
	if err := db.GetDB().WithContext(ctx).Order("channel_id ASC, model_name ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list circuit breakers: %w", err)
	}
	return records, nil
}

// CircuitBreakerUpsert 按 (channel_id, model_name) 唯一键写入熔断器记录。
func CircuitBreakerUpsert(record *model.CircuitBreaker) error {
	if record == nil || record.ChannelID == 0 || record.ModelName == "" {
		return fmt.Errorf("invalid circuit breaker record")
	}
	return db.GetDB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_id"}, {Name: "model_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"channel_name", "state", "trip_count", "consecutive_failures", "manual_disabled", "last_error_time", "last_error", "updated_at"}),
	}).Create(record).Error
}

// CircuitBreakerDelete 删除指定渠道+模型的熔断记录（取消熔断并清空历史）。
func CircuitBreakerDelete(channelID int, modelName string) error {
	return db.GetDB().Where("channel_id = ? AND model_name = ?", channelID, modelName).Delete(&model.CircuitBreaker{}).Error
}

// CircuitBreakerDeleteByChannel 删除指定渠道的全部熔断记录（渠道删除时清理）。
func CircuitBreakerDeleteByChannel(channelID int) error {
	return db.GetDB().Where("channel_id = ?", channelID).Delete(&model.CircuitBreaker{}).Error
}

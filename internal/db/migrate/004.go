package migrate

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 4,
		Up:      ensureCircuitBreakerIndex,
	})
}

// 004: 兼容旧版 sqlite 数据库。
// AutoMigrate 会自动创建 circuit_breakers 表及唯一索引；这里兜底确保 (channel_id, model_name)
// 复合唯一索引存在，避免旧库中该表结构不完整导致重复记录。
func ensureCircuitBreakerIndex(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable(&model.CircuitBreaker{}) {
		return nil
	}
	if !db.Migrator().HasIndex(&model.CircuitBreaker{}, "idx_cb_channel_model") {
		if err := db.Migrator().CreateIndex(&model.CircuitBreaker{}, "idx_cb_channel_model"); err != nil {
			return fmt.Errorf("failed to ensure circuit_breakers index: %w", err)
		}
	}
	return nil
}

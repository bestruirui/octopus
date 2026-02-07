package migrate

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 3,
		Up:      ensureCircuitBreakerColumns,
	})
}

func ensureCircuitBreakerColumns(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	if !db.Migrator().HasColumn(&model.RelayLog{}, "cb_log_level_max") {
		if err := db.Migrator().AddColumn(&model.RelayLog{}, "cb_log_level_max"); err != nil {
			return fmt.Errorf("failed to add relay_logs.cb_log_level_max: %w", err)
		}
	}
	return nil
}

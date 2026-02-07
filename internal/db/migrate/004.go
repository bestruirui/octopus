package migrate

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 4,
		Up:      ensureGroupItemCircuitBreakerColumn,
	})
}

func ensureGroupItemCircuitBreakerColumn(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if db.Migrator().HasColumn(&model.GroupItem{}, "cb_enabled") {
		return nil
	}
	if err := db.Migrator().AddColumn(&model.GroupItem{}, "cb_enabled"); err != nil {
		return fmt.Errorf("failed to add group_items.cb_enabled: %w", err)
	}
	return nil
}


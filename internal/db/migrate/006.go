package migrate

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 6,
		Up:      upgradeCircuitBreakerBaseCooldownDefaultTo180s,
	})
}

// upgradeCircuitBreakerBaseCooldownDefaultTo180s upgrades legacy default 60s to 180s.
// User customized values are preserved.
func upgradeCircuitBreakerBaseCooldownDefaultTo180s(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := db.Model(&model.Setting{}).
		Where("key = ? AND value = ?", model.SettingKeyCBBaseCooldownMS, "60000").
		Update("value", "180000").Error; err != nil {
		return fmt.Errorf("failed to upgrade setting %s: %w", model.SettingKeyCBBaseCooldownMS, err)
	}
	return nil
}

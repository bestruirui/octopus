package migrate

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 5,
		Up:      upgradeCircuitBreakerDefaultSettings,
	})
}

// upgradeCircuitBreakerDefaultSettings upgrades legacy default values only.
// It keeps user-customized values untouched.
func upgradeCircuitBreakerDefaultSettings(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	legacyToNew := map[model.SettingKey]struct {
		oldVal string
		newVal string
	}{
		model.SettingKeyCBFailureThreshold: {oldVal: "5", newVal: "3"},
		model.SettingKeyCBBaseCooldownMS:   {oldVal: "30000", newVal: "60000"},
		model.SettingKeyCBMaxCooldownMS:    {oldVal: "300000", newVal: "3600000"},
	}

	for key, pair := range legacyToNew {
		if err := db.Model(&model.Setting{}).
			Where("key = ? AND value = ?", key, pair.oldVal).
			Update("value", pair.newVal).Error; err != nil {
			return fmt.Errorf("failed to upgrade setting %s: %w", key, err)
		}
	}
	return nil
}

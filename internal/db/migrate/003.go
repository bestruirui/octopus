package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 3,
		Up:      addChannelKeyModeAndWeight,
	})
}

// 003: add channels.key_mode and channel_keys.weight columns
func addChannelKeyModeAndWeight(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	dialect := db.Dialector.Name()

	// column existence helper
	hasColumn := func(table, column string) bool {
		switch dialect {
		case "sqlite":
			var name string
			db.Raw("SELECT name FROM pragma_table_info(?) WHERE name = ? LIMIT 1", table, column).Scan(&name)
			return name == column
		case "mysql":
			var count int64
			db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?", table, column).Scan(&count)
			return count > 0
		case "postgres":
			var count int64
			db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?", table, column).Scan(&count)
			return count > 0
		default:
			return db.Migrator().HasColumn(table, column)
		}
	}

	// add channels.key_mode
	if !hasColumn("channels", "key_mode") {
		var sql string
		switch dialect {
		case "sqlite":
			sql = "ALTER TABLE channels ADD COLUMN key_mode INTEGER DEFAULT 0"
		case "mysql":
			sql = "ALTER TABLE channels ADD COLUMN key_mode INT DEFAULT 0"
		case "postgres":
			sql = "ALTER TABLE channels ADD COLUMN key_mode INTEGER DEFAULT 0"
		default:
			sql = "ALTER TABLE channels ADD COLUMN key_mode INTEGER DEFAULT 0"
		}
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to add channels.key_mode: %w", err)
		}
	}

	// add channel_keys.weight
	if !hasColumn("channel_keys", "weight") {
		var sql string
		switch dialect {
		case "sqlite":
			sql = "ALTER TABLE channel_keys ADD COLUMN weight INTEGER DEFAULT 1"
		case "mysql":
			sql = "ALTER TABLE channel_keys ADD COLUMN weight INT DEFAULT 1"
		case "postgres":
			sql = "ALTER TABLE channel_keys ADD COLUMN weight INTEGER DEFAULT 1"
		default:
			sql = "ALTER TABLE channel_keys ADD COLUMN weight INTEGER DEFAULT 1"
		}
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to add channel_keys.weight: %w", err)
		}
	}

	return nil
}
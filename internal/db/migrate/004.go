package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 4,
		Up:      addTOTPColumnsToUsers,
	})
}

// 004: add totp_enabled and totp_secret columns to users table
func addTOTPColumnsToUsers(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	dialect := db.Dialector.Name()

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

	addColumn := func(table, column, columnType string) error {
		var sql string
		switch dialect {
		case "sqlite":
			sql = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType)
		case "mysql":
			sql = fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", table, column, columnType)
		case "postgres":
			sql = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType)
		default:
			sql = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnType)
		}
		return db.Exec(sql).Error
	}

	if !hasColumn("users", "totp_enabled") {
		if err := addColumn("users", "totp_enabled", "boolean default false"); err != nil {
			return fmt.Errorf("failed to add totp_enabled column: %w", err)
		}
	}

	if !hasColumn("users", "totp_secret") {
		if err := addColumn("users", "totp_secret", "text default ''"); err != nil {
			return fmt.Errorf("failed to add totp_secret column: %w", err)
		}
	}

	return nil
}

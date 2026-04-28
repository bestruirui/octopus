package migrate

import (
	"fmt"

	"github.com/lingyuins/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 4,
		Up:      addAPIKeyQuotaColumns,
	})
}

func addAPIKeyQuotaColumns(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable("api_keys") {
		return nil
	}

	columns := []struct {
		name string
		typ  string
	}{
		{"rate_limit_rpm", "INT"},
		{"rate_limit_tpm", "INT"},
		{"per_model_quota_json", "TEXT"},
	}

	hasColumn := func(table, column string) (bool, error) {
		switch db.Dialector.Name() {
		case "sqlite":
			var name string
			if err := db.Raw("SELECT name FROM pragma_table_info(?) WHERE name = ? LIMIT 1", table, column).
				Scan(&name).Error; err != nil {
				return false, fmt.Errorf("failed to check sqlite column %s.%s: %w", table, column, err)
			}
			return name == column, nil
		default:
			return db.Migrator().HasColumn(table, column), nil
		}
	}

	for _, col := range columns {
		exists, err := hasColumn("api_keys", col.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := setColumnDefault(db, "api_keys", col.name, col.typ, "0"); err != nil {
			return fmt.Errorf("failed to add column api_keys.%s: %w", col.name, err)
		}
	}
	return nil
}

func setColumnDefault(db *gorm.DB, table, column, typ, defaultVal string) error {
	switch db.Dialector.Name() {
	case "sqlite":
		return db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s DEFAULT %s", table, column, typ, defaultVal)).Error
	case "mysql":
		return db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s DEFAULT %s", table, column, typ, defaultVal)).Error
	default:
		return db.Migrator().AddColumn(&model.APIKey{}, column)
	}
}

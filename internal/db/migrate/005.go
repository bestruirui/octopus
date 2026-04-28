package migrate

import (
	"fmt"

	"github.com/lingyuins/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 5,
		Up:      addGroupConditionColumn,
	})
}

func addGroupConditionColumn(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable("groups") {
		return nil
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

	exists, err := hasColumn("groups", "condition")
	if err != nil {
		return err
	}
	if !exists {
		if err := db.Migrator().AddColumn(&model.Group{}, "Condition"); err != nil {
			return fmt.Errorf("failed to add groups.condition: %w", err)
		}
	}
	return nil
}

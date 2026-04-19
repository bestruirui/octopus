package migrate

import (
	"fmt"

	"github.com/lingyuins/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 3,
		Up:      ensureGroupsEndpointTypeColumn,
	})
}

// 003: ensure legacy groups tables have endpoint_type with a usable default.
func ensureGroupsEndpointTypeColumn(db *gorm.DB) error {
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

	exists, err := hasColumn("groups", "endpoint_type")
	if err != nil {
		return err
	}
	if !exists {
		if err := db.Migrator().AddColumn(&model.Group{}, "EndpointType"); err != nil {
			return fmt.Errorf("failed to add groups.endpoint_type: %w", err)
		}
	}

	if err := db.Exec(
		"UPDATE groups SET endpoint_type = ? WHERE endpoint_type IS NULL OR TRIM(endpoint_type) = ''",
		model.EndpointTypeAll,
	).Error; err != nil {
		return fmt.Errorf("failed to backfill groups.endpoint_type: %w", err)
	}

	if !db.Migrator().HasIndex(&model.Group{}, "EndpointType") {
		if err := db.Migrator().CreateIndex(&model.Group{}, "EndpointType"); err != nil {
			return fmt.Errorf("failed to create groups.endpoint_type index: %w", err)
		}
	}

	return nil
}

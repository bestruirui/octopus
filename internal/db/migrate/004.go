package migrate

import (
	"fmt"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterBeforeAutoMigration(Migration{
		Version: 4,
		Up:      migrateAddRouteTables,
	})
}

// 004: 创建 routes 和 route_groups 表，api_keys 添加 supported_routes 字段
func migrateAddRouteTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	// AutoMigrate 创建 routes 和 route_groups 表
	if err := db.AutoMigrate(&model.Route{}, &model.RouteGroup{}); err != nil {
		return fmt.Errorf("failed to auto migrate route tables: %w", err)
	}

	// api_keys 添加 supported_routes 字段（如果表存在）
	if db.Migrator().HasTable(&model.APIKey{}) {
		if !db.Migrator().HasColumn(&model.APIKey{}, "supported_routes") {
			if err := db.Migrator().AddColumn(&model.APIKey{}, "SupportedRoutes"); err != nil {
				return fmt.Errorf("failed to add supported_routes column to api_keys: %w", err)
			}
		}

		// 如果存在旧的 route_id 字段则删除
		if db.Migrator().HasColumn(&model.APIKey{}, "route_id") {
			if err := db.Migrator().DropColumn(&model.APIKey{}, "RouteID"); err != nil {
				return fmt.Errorf("failed to drop route_id column from api_keys: %w", err)
			}
		}
	}

	return nil
}

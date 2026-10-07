package migrate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 5,
		Up:      migrateChannelToSingleURLAndKey,
	})
}

// migrateChannelToSingleURLAndKey 将多地址、多凭据渠道收敛为单地址、单凭据，并删除旧结构。
func migrateChannelToSingleURLAndKey(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable("channels") {
		return nil
	}
	// channels.key 在当前架构中已不存在: 凭据已收敛到 channel_keys.key, 由 Migration 11 迁移。
	// 因此两列的有无只决定对应回填是否执行, 不再视为错误。
	// 用 hasPhysicalColumn 而非 Migrator().HasColumn: 后者在 SQLite 下按建表语句模糊匹配,
	// channels 的 "id integer PRIMARY KEY AUTOINCREMENT" 会被误判成存在 key 列。
	hasBaseURL := hasPhysicalColumn(db, "channels", "base_url")
	hasKey := hasPhysicalColumn(db, "channels", "key")

	if hasBaseURL && hasPhysicalColumn(db, "channels", "base_urls") {
		type legacyBaseURL struct {
			URL string `json:"url"` // 旧地址值。
		}
		type legacyChannel struct {
			ID       int    `gorm:"column:id"`        // 渠道主键。
			BaseURLs string `gorm:"column:base_urls"` // 旧地址数组 JSON。
		}

		rows := make([]legacyChannel, 0)
		if err := db.Table("channels").Select("id, base_urls").Find(&rows).Error; err != nil {
			return fmt.Errorf("failed to read channels.base_urls: %w", err)
		}
		for _, row := range rows {
			if strings.TrimSpace(row.BaseURLs) == "" || strings.TrimSpace(row.BaseURLs) == "null" {
				continue
			}
			urls := make([]legacyBaseURL, 0)
			if err := json.Unmarshal([]byte(row.BaseURLs), &urls); err != nil {
				return fmt.Errorf("failed to decode channels.base_urls for id=%d: %w", row.ID, err)
			}
			if len(urls) == 0 || strings.TrimSpace(urls[0].URL) == "" {
				continue
			}
			if err := db.Table("channels").Where("id = ? AND (base_url IS NULL OR base_url = '')", row.ID).Update("base_url", urls[0].URL).Error; err != nil {
				return fmt.Errorf("failed to migrate channels.base_urls for id=%d: %w", row.ID, err)
			}
		}
	}

	// 目标列 channels.key 不存在时必须跳过: 该列已由 Migration 11 删除, 凭据改由 channel_keys 承载。
	if hasKey && db.Migrator().HasTable("channel_keys") && hasPhysicalColumn(db, "channel_keys", "channel_key") {
		type legacyChannelKey struct {
			ChannelID  int    `gorm:"column:channel_id"`  // 所属渠道主键。
			ChannelKey string `gorm:"column:channel_key"` // 旧凭据值。
		}

		keys := make([]legacyChannelKey, 0)
		// 按旧记录主键顺序读取，每个渠道只保留第一项。
		if err := db.Table("channel_keys").
			Select("channel_id, channel_key").
			Where("channel_key <> ''").
			Order("channel_id ASC, id ASC").
			Find(&keys).Error; err != nil {
			return fmt.Errorf("failed to read channel_keys: %w", err)
		}
		selected := make(map[int]struct{})
		var quotedKey strings.Builder
		db.Dialector.QuoteTo(&quotedKey, "key")
		emptyKeyCondition := fmt.Sprintf("(%s IS NULL OR %s = '')", quotedKey.String(), quotedKey.String())
		for _, key := range keys {
			if _, ok := selected[key.ChannelID]; ok || strings.TrimSpace(key.ChannelKey) == "" {
				continue
			}
			if err := db.Table("channels").Where("id = ? AND "+emptyKeyCondition, key.ChannelID).Update("key", key.ChannelKey).Error; err != nil {
				return fmt.Errorf("failed to migrate channel key for channel_id=%d: %w", key.ChannelID, err)
			}
			selected[key.ChannelID] = struct{}{}
		}
	}

	// Migration 11 重新引入了 channel_keys 表用于多凭据架构，只有在旧架构（channels.type 存在）时才删除该表。
	// Migration 11 会删除 channels.type 列，若该列不存在说明 Migration 11 已执行，channel_keys 表不应删除。
	if db.Migrator().HasTable("channel_keys") && hasPhysicalColumn(db, "channels", "type") {
		if err := db.Migrator().DropTable("channel_keys"); err != nil {
			return fmt.Errorf("failed to drop channel_keys: %w", err)
		}
	}
	if hasPhysicalColumn(db, "channels", "base_urls") {
		if db.Dialector.Name() == "sqlite" {
			if err := db.Exec(`ALTER TABLE "channels" DROP COLUMN "base_urls"`).Error; err != nil {
				return fmt.Errorf("failed to drop channels.base_urls: %w", err)
			}
		} else if err := db.Migrator().DropColumn(&model.Channel{}, "base_urls"); err != nil {
			return fmt.Errorf("failed to drop channels.base_urls: %w", err)
		}
	}
	return nil
}

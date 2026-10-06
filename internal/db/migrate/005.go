package migrate

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func init() {
	// 必须在迁移 11 转存并删除 channels.key 之前收敛旧地址与凭据。
	RegisterBeforeAutoMigration(Migration{
		Version: 5,
		Up:      migrateChannelToSingleURLAndKey,
	})
}

// migrateChannelToSingleURLAndKey 将多地址、多凭据渠道收敛为单地址、单凭据，并删除旧结构。
func migrateChannelToSingleURLAndKey(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	// 迁移 11 最后删除 type 列；新库和已迁移的库不再使用渠道级 key。
	if !db.Migrator().HasTable("channels") || !hasPhysicalColumn(db, "channels", "type") {
		return nil
	}
	// 在 AutoMigrate 前执行，按历史结构补齐回填列，不能依赖当前 Channel 模型。
	for _, column := range []string{"base_url", "key"} {
		if hasPhysicalColumn(db, "channels", column) {
			continue
		}
		if err := db.Migrator().AddColumn(&channelsV5{}, column); err != nil {
			return fmt.Errorf("failed to add channels.%s: %w", column, err)
		}
	}

	if hasPhysicalColumn(db, "channels", "base_urls") {
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

	hasLegacyKeys := db.Migrator().HasTable("channel_keys") && hasPhysicalColumn(db, "channel_keys", "channel_key")
	if hasLegacyKeys {
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

	// 仅删除已转存的旧凭据表，保留迁移 11 创建的同名新表。
	if hasLegacyKeys {
		if err := db.Migrator().DropTable("channel_keys"); err != nil {
			return fmt.Errorf("failed to drop channel_keys: %w", err)
		}
	}
	return dropColumnIfExists(db, &channelsV5{}, "channels", "base_urls")
}

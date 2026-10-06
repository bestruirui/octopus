package migrate

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	RegisterBeforeAutoMigration(Migration{
		Version: 8,
		Up:      migrateChannelModels,
	})
}

// channelModelKey 用渠道和名称唯一定位一条渠道模型记录。
type channelModelKey struct {
	ChannelID int    // 所属渠道主键。
	Name      string // 渠道模型名称。
}

// migrateChannelModels 将旧渠道模型和分组项转换为渠道模型外键结构，并清理旧表。
func migrateChannelModels(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable("channels") {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := migrateLegacyChannelStats(tx); err != nil {
			return err
		}
		if !tx.Migrator().HasTable("channel_models") {
			if err := tx.AutoMigrate(&channelModelsV8{}); err != nil {
				return fmt.Errorf("failed to create channel_models: %w", err)
			}
		}

		type legacyChannel struct {
			ID          int            // 渠道主键。
			AutoSync    bool           // 是否自动同步模型。
			Models      sql.NullString `gorm:"column:model"`        // 旧渠道模型列表。
			CustomModel sql.NullString `gorm:"column:custom_model"` // 旧手动模型列表。
		}
		channels := make([]legacyChannel, 0)
		channelFields := []string{"id"}
		if tx.Migrator().HasColumn(&channelsTable{}, "auto_sync") {
			channelFields = append(channelFields, "auto_sync")
		}
		if hasPhysicalColumn(tx, "channels", "model") {
			channelFields = append(channelFields, "model")
		}
		if hasPhysicalColumn(tx, "channels", "custom_model") {
			channelFields = append(channelFields, "custom_model")
		}
		if err := tx.Table("channels").Select(channelFields).Order("id ASC").Find(&channels).Error; err != nil {
			return fmt.Errorf("failed to read legacy channels: %w", err)
		}

		channelAutoSync := make(map[int]bool, len(channels))
		for _, channel := range channels {
			channelAutoSync[channel.ID] = channel.AutoSync
		}

		channelModels := make([]channelModelsV8, 0)
		if err := tx.Order("id ASC").Find(&channelModels).Error; err != nil {
			return fmt.Errorf("failed to read channel_models: %w", err)
		}
		modelsByKey := make(map[channelModelKey]*channelModelsV8, len(channelModels))
		existingModelKeys := make(map[channelModelKey]struct{}, len(channelModels))
		modelOrder := make([]channelModelKey, 0)
		for i := range channelModels {
			key := channelModelKey{ChannelID: channelModels[i].ChannelID, Name: channelModels[i].Name}
			modelsByKey[key] = &channelModels[i]
			existingModelKeys[key] = struct{}{}
			modelOrder = append(modelOrder, key)
		}

		// 同名模型只保留一行，手动来源优先于自动来源。
		// 旧 group_items 对 channel_id 没有外键，渠道删除后会留下孤立行，
		// 这些行引用的渠道已不存在，插入 channel_models 会违反外键，必须跳过。
		addModel := func(channelID int, name string, source string) {
			name = strings.TrimSpace(name)
			if channelID == 0 || name == "" {
				return
			}
			if _, ok := channelAutoSync[channelID]; !ok {
				return
			}
			key := channelModelKey{ChannelID: channelID, Name: name}
			current, ok := modelsByKey[key]
			if !ok {
				current = &channelModelsV8{ChannelID: channelID, Name: name, Source: source}
				modelsByKey[key] = current
				modelOrder = append(modelOrder, key)
			} else if source == snapshotModelSourceManual {
				current.Source = snapshotModelSourceManual
			}
		}

		for _, channel := range channels {
			if channel.Models.Valid {
				for _, name := range strings.Split(channel.Models.String, ",") {
					addModel(channel.ID, name, snapshotModelSourceAuto)
				}
			}
			if channel.CustomModel.Valid {
				for _, name := range strings.Split(channel.CustomModel.String, ",") {
					addModel(channel.ID, name, snapshotModelSourceManual)
				}
			}
		}

		if tx.Migrator().HasTable("group_items") &&
			hasPhysicalColumn(tx, "group_items", "channel_id") &&
			hasPhysicalColumn(tx, "group_items", "model_name") {
			type legacyGroupItemName struct {
				ChannelID int    // 旧渠道主键。
				ModelName string // 旧模型名称。
			}
			legacyNames := make([]legacyGroupItemName, 0)
			if err := tx.Table("group_items").Select("channel_id, model_name").Order("id ASC").Find(&legacyNames).Error; err != nil {
				return fmt.Errorf("failed to read legacy group item models: %w", err)
			}
			for _, item := range legacyNames {
				key := channelModelKey{ChannelID: item.ChannelID, Name: strings.TrimSpace(item.ModelName)}
				if _, exists := modelsByKey[key]; exists {
					continue
				}
				source := snapshotModelSourceManual
				if channelAutoSync[item.ChannelID] {
					source = snapshotModelSourceAuto
				}
				addModel(item.ChannelID, item.ModelName, source)
			}
		}

		for _, key := range modelOrder {
			channelModel := modelsByKey[key]
			if _, exists := existingModelKeys[key]; !exists {
				continue
			}
			if err := tx.Model(&channelModelsV8{}).Where("id = ?", channelModel.ID).Updates(channelModel).Error; err != nil {
				return fmt.Errorf("failed to update channel_model %d: %w", channelModel.ID, err)
			}
		}
		// MySQL 默认排序规则不区分大小写，仅大小写不同的模型会撞 idx_channel_model_name 唯一索引，
		// 直接 Create 会中断整个迁移，这里冲突跳过，随后统一从库里回读真实主键。
		for _, key := range modelOrder {
			channelModel := modelsByKey[key]
			if _, exists := existingModelKeys[key]; exists {
				continue
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(channelModel).Error; err != nil {
				return fmt.Errorf("failed to create channel_model %s: %w", channelModel.Name, err)
			}
		}

		// 冲突跳过的行不会回写主键，且区分大小写的库里同名模型可能只留下一行，
		// 统一重新读取并按精确名优先、小写名兜底的方式回填主键，避免 group_items 写入 0。
		saved := make([]channelModelsV8, 0)
		if err := tx.Order("id ASC").Find(&saved).Error; err != nil {
			return fmt.Errorf("failed to reload channel_models: %w", err)
		}
		savedIDByKey := make(map[channelModelKey]int, len(saved))
		savedIDByLowerKey := make(map[channelModelKey]int, len(saved))
		for _, channelModel := range saved {
			name := strings.TrimSpace(channelModel.Name)
			exact := channelModelKey{ChannelID: channelModel.ChannelID, Name: name}
			if _, ok := savedIDByKey[exact]; !ok {
				savedIDByKey[exact] = channelModel.ID
			}
			lower := channelModelKey{ChannelID: channelModel.ChannelID, Name: strings.ToLower(name)}
			if _, ok := savedIDByLowerKey[lower]; !ok {
				savedIDByLowerKey[lower] = channelModel.ID
			}
		}
		for _, key := range modelOrder {
			channelModel := modelsByKey[key]
			if id, ok := savedIDByKey[channelModelKey{ChannelID: key.ChannelID, Name: strings.TrimSpace(key.Name)}]; ok {
				channelModel.ID = id
				continue
			}
			if id, ok := savedIDByLowerKey[channelModelKey{ChannelID: key.ChannelID, Name: strings.ToLower(strings.TrimSpace(key.Name))}]; ok {
				channelModel.ID = id
			}
		}

		if err := migrateLegacyGroupItems(tx, modelsByKey); err != nil {
			return err
		}
		for _, column := range []string{"model", "custom_model", "auto_group"} {
			if err := dropColumnIfExists(tx, &channelsTable{}, "channels", column); err != nil {
				return err
			}
		}
		// 渠道统计已并入 channels，未实际使用的模型统计直接丢弃。
		for _, table := range []string{"stats_models", "stats_channels"} {
			if tx.Migrator().HasTable(table) {
				if err := tx.Migrator().DropTable(table); err != nil {
					return fmt.Errorf("failed to drop %s: %w", table, err)
				}
			}
		}
		return nil
	})
}

// migrateLegacyChannelStats 将旧渠道统计原值迁移到 channels 的内嵌统计字段。
func migrateLegacyChannelStats(db *gorm.DB) error {
	for _, field := range []string{"InputToken", "OutputToken", "InputCost", "OutputCost", "WaitTime", "RequestSuccess", "RequestFailed"} {
		if db.Migrator().HasColumn(&channelsTable{}, field) {
			continue
		}
		if err := db.Migrator().AddColumn(&channelsTable{}, field); err != nil {
			return fmt.Errorf("failed to add channels stats column %s: %w", field, err)
		}
	}
	if !db.Migrator().HasTable("stats_channels") {
		return nil
	}

	type legacyChannelStats struct {
		ChannelID int `gorm:"column:channel_id"` // 所属渠道主键。
		SnapshotStatsMetrics
	}
	stats := make([]legacyChannelStats, 0)
	if err := db.Table("stats_channels").Order("channel_id ASC").Find(&stats).Error; err != nil {
		return fmt.Errorf("failed to read stats_channels: %w", err)
	}
	for _, channelStats := range stats {
		if err := db.Model(&channelsTable{}).Where("id = ?", channelStats.ChannelID).Updates(map[string]interface{}{
			"input_token":     channelStats.InputToken,
			"output_token":    channelStats.OutputToken,
			"input_cost":      channelStats.InputCost,
			"output_cost":     channelStats.OutputCost,
			"wait_time":       channelStats.WaitTime,
			"request_success": channelStats.RequestSuccess,
			"request_failed":  channelStats.RequestFailed,
		}).Error; err != nil {
			return fmt.Errorf("failed to migrate stats for channel_id=%d: %w", channelStats.ChannelID, err)
		}
	}
	return nil
}

// migrateLegacyGroupItems 将旧分组项重建为只保存渠道模型外键的表。
func migrateLegacyGroupItems(db *gorm.DB, modelsByKey map[channelModelKey]*channelModelsV8) error {
	if !db.Migrator().HasTable("group_items") ||
		!hasPhysicalColumn(db, "group_items", "channel_id") ||
		!hasPhysicalColumn(db, "group_items", "model_name") {
		return nil
	}

	type legacyGroupItem struct {
		ID        int    // 分组项主键。
		GroupID   int    // 所属分组主键。
		ChannelID int    // 旧渠道主键。
		ModelName string // 旧模型名称。
		Priority  int    // 展示和故障转移顺序。
	}
	legacyItems := make([]legacyGroupItem, 0)
	if err := db.Table("group_items").Order("id ASC").Find(&legacyItems).Error; err != nil {
		return fmt.Errorf("failed to read group_items: %w", err)
	}

	// 区分大小写的库里 modelsByKey 按原名建索引，不区分大小写的库里同名模型只落库一行，
	// 精确名找不到时按小写名兜底，避免把合法旧行误判为无效。
	modelsByLowerKey := make(map[channelModelKey]*channelModelsV8, len(modelsByKey))
	for key, channelModel := range modelsByKey {
		lower := channelModelKey{ChannelID: key.ChannelID, Name: strings.ToLower(strings.TrimSpace(key.Name))}
		if _, ok := modelsByLowerKey[lower]; !ok {
			modelsByLowerKey[lower] = channelModel
		}
	}

	items := make([]groupItemsV8, 0, len(legacyItems))
	invalidIDs := make([]int, 0)
	seen := make(map[[2]int]struct{}, len(legacyItems))
	for _, item := range legacyItems {
		name := strings.TrimSpace(item.ModelName)
		channelModel, ok := modelsByKey[channelModelKey{ChannelID: item.ChannelID, Name: name}]
		if !ok {
			channelModel, ok = modelsByLowerKey[channelModelKey{ChannelID: item.ChannelID, Name: strings.ToLower(name)}]
		}
		// 渠道已删除或模型未能落库时主键为 0，写入会违反 channel_model_id 外键，直接丢弃该行。
		if !ok || channelModel.ID == 0 {
			invalidIDs = append(invalidIDs, item.ID)
			continue
		}
		itemKey := [2]int{item.GroupID, channelModel.ID}
		if _, exists := seen[itemKey]; exists {
			invalidIDs = append(invalidIDs, item.ID)
			continue
		}
		seen[itemKey] = struct{}{}
		items = append(items, groupItemsV8{
			ID:             item.ID,
			GroupID:        item.GroupID,
			ChannelModelID: channelModel.ID,
			Priority:       item.Priority,
		})
	}

	if err := db.Migrator().DropTable(&groupItemsV8{}); err != nil {
		return fmt.Errorf("failed to drop legacy group_items: %w", err)
	}
	if err := db.AutoMigrate(&groupItemsV8{}); err != nil {
		return fmt.Errorf("failed to create group_items: %w", err)
	}
	if len(items) > 0 {
		if err := db.Create(&items).Error; err != nil {
			return fmt.Errorf("failed to create migrated group_items: %w", err)
		}
	}
	if len(invalidIDs) > 0 && db.Migrator().HasTable("groups") {
		if err := db.Model(&groupsTable{}).Where("active_item_id IN ?", invalidIDs).Update("active_item_id", 0).Error; err != nil {
			return fmt.Errorf("failed to clear invalid active items: %w", err)
		}
	}
	if db.Migrator().HasTable("groups") {
		if err := clearStaleActiveItems(db); err != nil {
			return err
		}
	}
	return nil
}

// clearStaleActiveItems 把 active_item_id 指向已不存在或不属于本分组的成员的分组重置为 0。
// groups 在 MySQL 8.0.2+ 和 TiDB 里是保留字，改用 GORM 模型让各方言自行加引号，避免拼接原始 SQL。
func clearStaleActiveItems(db *gorm.DB) error {
	type groupActiveItem struct {
		ID           int // 分组主键。
		ActiveItemID int // 分组当前选中的成员主键。
	}
	groups := make([]groupActiveItem, 0)
	if err := db.Model(&groupsTable{}).Where("active_item_id <> 0").
		Select("id, active_item_id").Find(&groups).Error; err != nil {
		return fmt.Errorf("failed to read group active items: %w", err)
	}
	if len(groups) == 0 {
		return nil
	}

	type groupItemRef struct {
		ID      int // 成员主键。
		GroupID int // 成员所属分组主键。
	}
	items := make([]groupItemRef, 0)
	if err := db.Model(&groupItemsV8{}).Select("id, group_id").Find(&items).Error; err != nil {
		return fmt.Errorf("failed to read group items: %w", err)
	}
	itemGroup := make(map[int]int, len(items))
	for _, item := range items {
		itemGroup[item.ID] = item.GroupID
	}

	staleIDs := make([]int, 0)
	for _, group := range groups {
		if groupID, ok := itemGroup[group.ActiveItemID]; !ok || groupID != group.ID {
			staleIDs = append(staleIDs, group.ID)
		}
	}
	if len(staleIDs) == 0 {
		return nil
	}
	if err := db.Model(&groupsTable{}).Where("id IN ?", staleIDs).
		Update("active_item_id", 0).Error; err != nil {
		return fmt.Errorf("failed to clear stale active items: %w", err)
	}
	return nil
}

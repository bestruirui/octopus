package op

import (
	"context"
	"fmt"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const dbDumpVersion = 2

func DBExportAll(ctx context.Context, includeStats bool) (*model.DBDump, error) {
	conn := db.GetDB().WithContext(ctx)

	d := &model.DBDump{
		Version:      dbDumpVersion,
		ExportedAt:   time.Now().UTC(),
		IncludeStats: includeStats,
	}

	if err := conn.Find(&d.Channels).Error; err != nil {
		return nil, fmt.Errorf("export channels: %w", err)
	}
	if err := conn.Find(&d.Groups).Error; err != nil {
		return nil, fmt.Errorf("export groups: %w", err)
	}
	if err := conn.Find(&d.GroupItems).Error; err != nil {
		return nil, fmt.Errorf("export group_items: %w", err)
	}
	if err := conn.Find(&d.LLMInfos).Error; err != nil {
		return nil, fmt.Errorf("export llm_infos: %w", err)
	}
	if err := conn.Find(&d.APIKeys).Error; err != nil {
		return nil, fmt.Errorf("export api_keys: %w", err)
	}
	if err := conn.Find(&d.Settings).Error; err != nil {
		return nil, fmt.Errorf("export settings: %w", err)
	}

	if includeStats {
		if err := conn.Find(&d.StatsTotal).Error; err != nil {
			return nil, fmt.Errorf("export stats_total: %w", err)
		}
		if err := conn.Find(&d.StatsDaily).Error; err != nil {
			return nil, fmt.Errorf("export stats_daily: %w", err)
		}
		if err := conn.Find(&d.StatsHourly).Error; err != nil {
			return nil, fmt.Errorf("export stats_hourly: %w", err)
		}
		if err := conn.Find(&d.StatsModel).Error; err != nil {
			return nil, fmt.Errorf("export stats_model: %w", err)
		}
		if err := conn.Find(&d.StatsChannel).Error; err != nil {
			return nil, fmt.Errorf("export stats_channel: %w", err)
		}
		if err := conn.Find(&d.StatsAPIKey).Error; err != nil {
			return nil, fmt.Errorf("export stats_api_key: %w", err)
		}
	}

	return d, nil
}

func DBImportIncremental(ctx context.Context, dump *model.DBDump) (*model.DBImportResult, error) {
	if dump == nil {
		return nil, fmt.Errorf("empty dump")
	}

	if dump.Version != 0 && dump.Version != dbDumpVersion {
		return nil, fmt.Errorf("unsupported dump version: %d", dump.Version)
	}

	conn := db.GetDB().WithContext(ctx)
	res := &model.DBImportResult{RowsAffected: map[string]int64{}}

	err := conn.Transaction(func(tx *gorm.DB) error {
		// 先导入无外键依赖的基础表：channels、groups、api_keys，只有这些表就位后才可查询有效主键集合来过滤孤儿引用。
		if n, err := createDoNothing(tx, dump.Channels); err != nil {
			return fmt.Errorf("import channels: %w", err)
		} else {
			res.RowsAffected["channels"] = n
		}
		if n, err := createDoNothing(tx, dump.Groups); err != nil {
			return fmt.Errorf("import groups: %w", err)
		} else {
			res.RowsAffected["groups"] = n
		}
		if n, err := createDoNothing(tx, dump.APIKeys); err != nil {
			return fmt.Errorf("import api_keys: %w", err)
		} else {
			res.RowsAffected["api_keys"] = n
		}

		// 过滤掉引用已不存在主键的孤儿记录（如渠道被删除后残留的 stats_channel / stats_model，导入时会触发外键约束错误）。
		if err := sanitizeOrphanRows(tx, dump); err != nil {
			return err
		}

		if n, err := createDoNothing(tx, dump.GroupItems); err != nil {
			return fmt.Errorf("import group_items: %w", err)
		} else {
			res.RowsAffected["group_items"] = n
		}
		if n, err := createUpsertAll(tx, dump.LLMInfos, []clause.Column{{Name: "name"}}); err != nil {
			return fmt.Errorf("import llm_infos: %w", err)
		} else {
			res.RowsAffected["llm_infos"] = n
		}
		if n, err := createUpsertSettings(tx, dump.Settings); err != nil {
			return fmt.Errorf("import settings: %w", err)
		} else {
			res.RowsAffected["settings"] = n
		}

		if dump.IncludeStats {
			if n, err := createUpsertAll(tx, dump.StatsTotal, []clause.Column{{Name: "id"}}); err != nil {
				return fmt.Errorf("import stats_total: %w", err)
			} else {
				res.RowsAffected["stats_total"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsDaily, []clause.Column{{Name: "date"}}); err != nil {
				return fmt.Errorf("import stats_daily: %w", err)
			} else {
				res.RowsAffected["stats_daily"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsHourly, []clause.Column{{Name: "hour"}}); err != nil {
				return fmt.Errorf("import stats_hourly: %w", err)
			} else {
				res.RowsAffected["stats_hourly"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsModel, []clause.Column{{Name: "id"}}); err != nil {
				return fmt.Errorf("import stats_model: %w", err)
			} else {
				res.RowsAffected["stats_model"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsChannel, []clause.Column{{Name: "channel_id"}}); err != nil {
				return fmt.Errorf("import stats_channel: %w", err)
			} else {
				res.RowsAffected["stats_channel"] = n
			}
			if n, err := createUpsertAll(tx, dump.StatsAPIKey, []clause.Column{{Name: "api_key_id"}}); err != nil {
				return fmt.Errorf("import stats_api_key: %w", err)
			} else {
				res.RowsAffected["stats_api_key"] = n
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// sanitizeOrphanRows 过滤 dump 中引用已不存在主键的孤儿记录。
// stats_channel / stats_model 引用 channels.id，stats_api_key 引用 api_keys.id。
// 若源库中这些渠道/密钥已被删除，导出文件会残留对应统计行，导入时触发外键约束错误（如 PostgreSQL 的 fk_channels_stats / SQLSTATE 23503）。
// 过滤在事务内、基础表写入后进行，此时已能查询到有效主键集合。
func sanitizeOrphanRows(tx *gorm.DB, dump *model.DBDump) error {
	// 渠道主键集合（来自导入数据 + 数据库中已存在的渠道）。
	channelIDs := make(map[int]struct{}, len(dump.Channels))
	for _, c := range dump.Channels {
		if c.ID != 0 {
			channelIDs[c.ID] = struct{}{}
		}
	}
	// 合并数据库中已有的渠道 id（增量导入时 dump 可能只含部分渠道）。
	var dbChannelIDs []int
	if err := tx.Model(&model.Channel{}).Pluck("id", &dbChannelIDs).Error; err != nil {
		return fmt.Errorf("query channels for orphan filtering: %w", err)
	}
	for _, id := range dbChannelIDs {
		channelIDs[id] = struct{}{}
	}

	if len(dump.StatsChannel) > 0 {
		kept := dump.StatsChannel[:0]
		for _, s := range dump.StatsChannel {
			if _, ok := channelIDs[s.ChannelID]; ok {
				kept = append(kept, s)
			}
		}
		dropped := len(dump.StatsChannel) - len(kept)
		dump.StatsChannel = kept
		if dropped > 0 {
			fmt.Printf("[import] dropped %d orphaned stats_channel row(s)\n", dropped)
		}
	}
	if len(dump.StatsModel) > 0 {
		kept := dump.StatsModel[:0]
		for _, s := range dump.StatsModel {
			if _, ok := channelIDs[s.ChannelID]; ok {
				kept = append(kept, s)
			}
		}
		dropped := len(dump.StatsModel) - len(kept)
		dump.StatsModel = kept
		if dropped > 0 {
			fmt.Printf("[import] dropped %d orphaned stats_model row(s)\n", dropped)
		}
	}

	// API key 主键集合（来自导入数据 + 数据库中已存在的密钥）。
	apiKeyIDs := make(map[int]struct{}, len(dump.APIKeys))
	for _, k := range dump.APIKeys {
		if k.ID != 0 {
			apiKeyIDs[k.ID] = struct{}{}
		}
	}
	var dbAPIKeyIDs []int
	if err := tx.Model(&model.APIKey{}).Pluck("id", &dbAPIKeyIDs).Error; err != nil {
		return fmt.Errorf("query api_keys for orphan filtering: %w", err)
	}
	for _, id := range dbAPIKeyIDs {
		apiKeyIDs[id] = struct{}{}
	}
	if len(dump.StatsAPIKey) > 0 {
		kept := dump.StatsAPIKey[:0]
		for _, s := range dump.StatsAPIKey {
			if _, ok := apiKeyIDs[s.APIKeyID]; ok {
				kept = append(kept, s)
			}
		}
		dropped := len(dump.StatsAPIKey) - len(kept)
		dump.StatsAPIKey = kept
		if dropped > 0 {
			fmt.Printf("[import] dropped %d orphaned stats_api_key row(s)\n", dropped)
		}
	}

	return nil
}

func createDoNothing[T any](tx *gorm.DB, rows []T) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertAll[T any](tx *gorm.DB, rows []T, columns []clause.Column) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   columns,
		UpdateAll: true,
	}).Create(&rows)
	return result.RowsAffected, result.Error
}

func createUpsertSettings(tx *gorm.DB, rows []model.Setting) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&rows)
	return result.RowsAffected, result.Error
}

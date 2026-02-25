package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/google/uuid"
)

type ImportStatus string

const (
	ImportStatusPending   ImportStatus = "pending"   // 等待中
	ImportStatusRunning   ImportStatus = "running"   // 运行中
	ImportStatusCompleted ImportStatus = "completed" // 已完成
	ImportStatusFailed    ImportStatus = "failed"    // 失败
)

type ImportJob struct {
	ID           string       `json:"id"`
	ChannelID    int          `json:"channel_id"`
	Status       ImportStatus `json:"status"`
	Total        int          `json:"total"`         // 总数
	Processed    int          `json:"processed"`     // 已处理数量
	SuccessCount int          `json:"success_count"` // 成功数量
	FailCount    int          `json:"fail_count"`    // 失败数量
	Errors       []string     `json:"errors"`        // 详细错误信息
	Duplicates   []string     `json:"duplicates"`    // 重复的key列表
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	mu           sync.Mutex
	cancel       context.CancelFunc // 取消函数
}

var (
	jobs   = make(map[string]*ImportJob) // 存储所有导入任务
	jobsMu sync.RWMutex                  // 读写锁保护 jobs map
)

// CreateImportJob 创建一个新的导入任务
func CreateImportJob(channelID int, keys []string) string {
	jobID := uuid.New().String()
	job := &ImportJob{
		ID:        jobID,
		ChannelID: channelID,
		Status:    ImportStatusPending,
		Total:     len(keys),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jobsMu.Lock()
	jobs[jobID] = job
	jobsMu.Unlock()

	// 异步开始处理
	go processImportJob(job, keys)

	return jobID
}

// GetImportJob 获取导入任务状态
func GetImportJob(id string) (*model.BatchImportStatusResponse, bool) {
	jobsMu.RLock()
	job, exists := jobs[id]
	jobsMu.RUnlock()

	if !exists {
		return nil, false
	}

	job.mu.Lock()
	defer job.mu.Unlock()

	return &model.BatchImportStatusResponse{
		ID:           job.ID,
		Status:       string(job.Status),
		Total:        job.Total,
		Processed:    job.Processed,
		SuccessCount: job.SuccessCount,
		FailCount:    job.FailCount,
		Errors:       job.Errors,
		Duplicates:   job.Duplicates,
	}, true
}

// CancelImportJob 取消导入任务
func CancelImportJob(id string) bool {
	jobsMu.RLock()
	job, exists := jobs[id]
	jobsMu.RUnlock()

	if !exists {
		return false
	}

	job.mu.Lock()
	defer job.mu.Unlock()

	// 只有运行中或等待中的任务可以取消
	if job.Status == ImportStatusRunning || job.Status == ImportStatusPending {
		if job.cancel != nil {
			job.cancel()
		}
		job.Status = ImportStatusFailed
		job.Errors = append(job.Errors, "任务已被用户取消")
		return true
	}
	return false
}

// processImportJob 处理导入任务
func processImportJob(job *ImportJob, keys []string) {
	ctx, cancel := context.WithCancel(context.Background())
	job.mu.Lock()
	// 如果任务已经失败（例如被取消），则直接返回
	if job.Status == ImportStatusFailed {
		job.mu.Unlock()
		cancel()
		return
	}
	job.Status = ImportStatusRunning
	job.cancel = cancel
	job.mu.Unlock()

	defer func() {
		job.mu.Lock()
		if job.Status == ImportStatusRunning {
			job.Status = ImportStatusCompleted
			// 导入完成后刷新渠道缓存
			if err := op.ChannelRefreshCacheByID(job.ChannelID, ctx); err != nil {
				log.Errorf("刷新渠道缓存失败: %v", err)
			}
		}
		job.UpdatedAt = time.Now()
		job.cancel = nil
		job.mu.Unlock()
	}()

	batchSize := 500
	for i := 0; i < len(keys); i += batchSize {
		select {
		case <-ctx.Done():
			return
		default:
		}

		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batchKeys := keys[i:end]

		processBatch(ctx, job, batchKeys)
	}
}

// processBatch 处理一批key
func processBatch(ctx context.Context, job *ImportJob, keys []string) {
	// 检查数据库中是否存在这些key
	existingKeys := make(map[string]bool)

	// 优化：一次性查询这批key中哪些已经存在
	var existing []string
	if err := db.GetDB().WithContext(ctx).Model(&model.ChannelKey{}).
		Where("channel_key IN ?", keys).
		Pluck("channel_key", &existing).Error; err != nil {
		log.Errorf("查询已存在key失败: %v", err)
		// 如果检查失败，将这一批全部标记为失败
		job.mu.Lock()
		job.FailCount += len(keys)
		job.Processed += len(keys)
		job.Errors = append(job.Errors, fmt.Sprintf("批量处理失败: %v", err))
		job.mu.Unlock()
		return
	}

	for _, k := range existing {
		existingKeys[k] = true
	}

	newKeys := make([]model.ChannelKey, 0, len(keys))
	duplicates := []string{}

	for _, k := range keys {
		if existingKeys[k] {
			duplicates = append(duplicates, k)
		} else {
			newKeys = append(newKeys, model.ChannelKey{
				ChannelID:  job.ChannelID,
				Enabled:    true,
				ChannelKey: k,
			})
		}
	}

	// 插入新key
	if len(newKeys) > 0 {
		tx := db.GetDB().WithContext(ctx).Begin()
		if err := tx.CreateInBatches(&newKeys, 50).Error; err != nil {
			tx.Rollback()
			log.Errorf("插入key失败: %v", err)

			job.mu.Lock()
			job.FailCount += len(newKeys)
			job.Errors = append(job.Errors, fmt.Sprintf("批量插入失败: %v", err))
			job.mu.Unlock()
		} else {
			tx.Commit()
			job.mu.Lock()
			job.SuccessCount += len(newKeys)
			job.mu.Unlock()
		}
	}

	job.mu.Lock()
	job.Duplicates = append(job.Duplicates, duplicates...)
	job.Processed += len(keys)
	job.FailCount += len(duplicates) // 重复的也算作失败
	job.mu.Unlock()
}

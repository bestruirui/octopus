package op

import (
	"context"
	"errors"
	"fmt"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"gorm.io/gorm"
)

func AuditLogCreate(ctx context.Context, entry *model.AuditLog) error {
	if entry == nil {
		return fmt.Errorf("audit log entry is nil")
	}
	return db.GetDB().WithContext(ctx).Create(entry).Error
}

func AuditLogList(ctx context.Context, page, pageSize int) ([]model.AuditLog, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs := make([]model.AuditLog, 0, pageSize)
	err := db.GetDB().WithContext(ctx).
		Order("created_at DESC, id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func AuditLogGetByID(ctx context.Context, id int64) (*model.AuditLog, error) {
	var auditLog model.AuditLog
	err := db.GetDB().WithContext(ctx).First(&auditLog, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &auditLog, nil
}

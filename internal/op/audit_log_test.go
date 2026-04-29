package op

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
)

func TestAuditLogListOrdersAndPaginates(t *testing.T) {
	setupAuditLogTestDB(t)

	first := insertAuditLogFixture(t, model.AuditLog{
		UserID:     1,
		Username:   "alice",
		Action:     "channel.create",
		Method:     "POST",
		Path:       "/api/v1/channel/create",
		StatusCode: 200,
		Target:     "primary",
	}, 1_700_000_000)
	second := insertAuditLogFixture(t, model.AuditLog{
		UserID:     2,
		Username:   "bob",
		Action:     "group.update",
		Method:     "POST",
		Path:       "/api/v1/group/update",
		StatusCode: 400,
		Target:     "routing-a",
	}, 1_700_000_100)
	third := insertAuditLogFixture(t, model.AuditLog{
		UserID:     3,
		Username:   "charlie",
		Action:     "user.delete",
		Method:     "DELETE",
		Path:       "/api/v1/user/delete/9",
		StatusCode: 200,
		Target:     "id=9",
	}, 1_700_000_200)

	pageOne, err := AuditLogList(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("AuditLogList(page 1) error = %v", err)
	}
	if len(pageOne) != 2 {
		t.Fatalf("AuditLogList(page 1) len = %d, want 2", len(pageOne))
	}
	if pageOne[0].ID != third.ID || pageOne[1].ID != second.ID {
		t.Fatalf("AuditLogList(page 1) order = %+v, want [%d %d]", pageOne, third.ID, second.ID)
	}

	pageTwo, err := AuditLogList(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("AuditLogList(page 2) error = %v", err)
	}
	if len(pageTwo) != 1 {
		t.Fatalf("AuditLogList(page 2) len = %d, want 1", len(pageTwo))
	}
	if pageTwo[0].ID != first.ID {
		t.Fatalf("AuditLogList(page 2) id = %d, want %d", pageTwo[0].ID, first.ID)
	}
}

func TestAuditLogGetByIDHandlesMissingRows(t *testing.T) {
	setupAuditLogTestDB(t)

	entry := insertAuditLogFixture(t, model.AuditLog{
		UserID:     7,
		Username:   "ops",
		Action:     "setting.set",
		Method:     "POST",
		Path:       "/api/v1/setting/set",
		StatusCode: 200,
		Target:     "proxy_url",
	}, 1_700_000_123)

	got, err := AuditLogGetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("AuditLogGetByID(existing) error = %v", err)
	}
	if got == nil || got.Action != entry.Action || got.Target != entry.Target {
		t.Fatalf("AuditLogGetByID(existing) = %+v, want action=%q target=%q", got, entry.Action, entry.Target)
	}

	missing, err := AuditLogGetByID(context.Background(), entry.ID+999)
	if err != nil {
		t.Fatalf("AuditLogGetByID(missing) error = %v", err)
	}
	if missing != nil {
		t.Fatalf("AuditLogGetByID(missing) = %+v, want nil", missing)
	}
}

func setupAuditLogTestDB(t *testing.T) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name()))
	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
}

func insertAuditLogFixture(t *testing.T, entry model.AuditLog, createdAt int64) model.AuditLog {
	t.Helper()

	entry.CreatedAt = 0
	if err := AuditLogCreate(context.Background(), &entry); err != nil {
		t.Fatalf("AuditLogCreate() error = %v", err)
	}
	if err := db.GetDB().Model(&model.AuditLog{}).Where("id = ?", entry.ID).Update("created_at", createdAt).Error; err != nil {
		t.Fatalf("set created_at: %v", err)
	}
	entry.CreatedAt = createdAt
	return entry
}

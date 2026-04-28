package op

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
)

func TestUserDeleteRejectsActiveUserEvenWhenIDIsNotOne(t *testing.T) {
	oldUserCache := userCache
	userCache = model.User{}
	t.Cleanup(func() {
		userCache = oldUserCache
	})

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name()))
	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	legacy := model.User{
		Username: "admin",
		Password: "legacy-secret-123",
	}
	if err := legacy.HashPassword(); err != nil {
		t.Fatalf("hash legacy password: %v", err)
	}
	if err := db.GetDB().Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy user: %v", err)
	}

	if err := UserInit(); err != nil {
		t.Fatalf("user init: %v", err)
	}
	if err := deleteLegacyAdminUser("alice"); err != nil {
		t.Fatalf("delete legacy admin: %v", err)
	}
	if err := UserBootstrapCreate("alice", "super-secret-123"); err != nil {
		t.Fatalf("bootstrap replacement user: %v", err)
	}

	activeUser := UserGet()
	if activeUser.ID == 1 {
		t.Fatalf("active user id = %d, want non-1 id to cover the regression scenario", activeUser.ID)
	}

	err := UserDelete(activeUser.ID, activeUser.ID, context.Background())
	if err == nil {
		t.Fatal("UserDelete() error = nil, want active-user protection")
	}
	if !strings.Contains(err.Error(), "cannot delete the active user") {
		t.Fatalf("UserDelete() error = %q, want active-user protection", err.Error())
	}
}

func TestUserVerifySupportsNonCachedManagedUsers(t *testing.T) {
	oldUserCache := userCache
	userCache = model.User{}
	t.Cleanup(func() {
		userCache = oldUserCache
	})

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name()))
	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := UserBootstrapCreate("admin", "super-secret-123"); err != nil {
		t.Fatalf("bootstrap user: %v", err)
	}
	if err := UserCreate(model.UserCreateRequest{
		Username: "viewer",
		Password: "viewer-secret-123",
		Role:     model.UserRoleViewer,
	}, context.Background()); err != nil {
		t.Fatalf("create managed user: %v", err)
	}

	user, err := UserVerify("viewer", "viewer-secret-123")
	if err != nil {
		t.Fatalf("UserVerify() error = %v", err)
	}
	if user.Username != "viewer" {
		t.Fatalf("UserVerify() username = %q, want viewer", user.Username)
	}
	if user.Role != model.UserRoleViewer {
		t.Fatalf("UserVerify() role = %q, want %q", user.Role, model.UserRoleViewer)
	}
}

package migrate

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/lingyuins/octopus/internal/model"
	"gorm.io/gorm"
)

func TestEnsureGroupsEndpointTypeColumnAddsMissingColumnAndBackfills(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Exec(`
		CREATE TABLE groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			mode INTEGER NOT NULL,
			match_regex TEXT,
			first_token_time_out INTEGER,
			session_keep_time INTEGER
		)
	`).Error; err != nil {
		t.Fatalf("create legacy groups table: %v", err)
	}

	if err := db.Exec(`
		INSERT INTO groups (name, mode, match_regex, first_token_time_out, session_keep_time)
		VALUES ('legacy-group', 1, '', 0, 0)
	`).Error; err != nil {
		t.Fatalf("insert legacy group: %v", err)
	}

	if err := ensureGroupsEndpointTypeColumn(db); err != nil {
		t.Fatalf("ensureGroupsEndpointTypeColumn: %v", err)
	}

	if !db.Migrator().HasColumn("groups", "endpoint_type") {
		t.Fatal("expected groups.endpoint_type column to exist")
	}

	var endpointType string
	if err := db.Raw("SELECT endpoint_type FROM groups WHERE name = ?", "legacy-group").
		Scan(&endpointType).Error; err != nil {
		t.Fatalf("read endpoint_type: %v", err)
	}
	if endpointType != model.EndpointTypeAll {
		t.Fatalf("expected endpoint_type %q, got %q", model.EndpointTypeAll, endpointType)
	}

	var indexName string
	if err := db.Raw(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'index' AND tbl_name = 'groups' AND name = 'idx_groups_endpoint_type'
		LIMIT 1
	`).Scan(&indexName).Error; err != nil {
		t.Fatalf("read endpoint_type index: %v", err)
	}
	if indexName != "idx_groups_endpoint_type" {
		t.Fatalf("expected endpoint_type index to exist, got %q", indexName)
	}
}

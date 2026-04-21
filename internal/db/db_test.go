package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestInitSQLiteCreatesParentDir(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "octopus.db")

	gdb, err := initSQLite(dbPath, &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("initSQLite() error = %v", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("gdb.DB() error = %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("sqlDB.Ping() error = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(dbPath)); err != nil {
		t.Fatalf("os.Stat(parent dir) error = %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("os.Stat(db file) error = %v", err)
	}
}

func TestSQLiteDSNAppendsParamsWithExistingQuery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "octopus.db") + "?_txlock=immediate"

	dsn, err := sqliteDSN(dbPath)
	if err != nil {
		t.Fatalf("sqliteDSN() error = %v", err)
	}
	got := dsn + sqliteDSNSeparator(dsn) + "_journal_mode=WAL"
	if !strings.Contains(got, "?_txlock=immediate&_journal_mode=WAL") {
		t.Fatalf("combined DSN = %q, want query parameters appended with '&'", got)
	}
}

func TestInitSQLiteCreatesParentDirForFileURI(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "octopus.db")
	dsn := "file:" + filepath.ToSlash(dbPath) + "?_txlock=immediate"

	gdb, err := initSQLite(dsn, &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("initSQLite() error = %v", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("gdb.DB() error = %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("sqlDB.Ping() error = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(dbPath)); err != nil {
		t.Fatalf("os.Stat(parent dir) error = %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("os.Stat(db file) error = %v", err)
	}
}

func TestSQLiteDSNSkipsDirCreationForMemoryFileURI(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "octopus.db")
	dsn := "file:" + filepath.ToSlash(dbPath) + "?mode=memory&cache=shared"

	got, err := sqliteDSN(dsn)
	if err != nil {
		t.Fatalf("sqliteDSN() error = %v", err)
	}
	if got != dsn {
		t.Fatalf("sqliteDSN() = %q, want %q", got, dsn)
	}
	if _, err := os.Stat(filepath.Dir(dbPath)); !os.IsNotExist(err) {
		t.Fatalf("expected memory sqlite DSN not to create parent dir, stat error = %v", err)
	}
}

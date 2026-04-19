package conf

import (
	"path/filepath"
	"testing"
)

func TestDefaultPathsUseDataDirEnv(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "octopus-data")
	t.Setenv("OCTOPUS_DATA_DIR", dataDir)

	if got, want := defaultDataDir(), filepath.Clean(dataDir); got != want {
		t.Fatalf("defaultDataDir() = %q, want %q", got, want)
	}
	if got, want := defaultConfigPath(), filepath.Join(filepath.Clean(dataDir), "config.json"); got != want {
		t.Fatalf("defaultConfigPath() = %q, want %q", got, want)
	}
	if got, want := defaultDatabasePath(), filepath.Join(filepath.Clean(dataDir), "data.db"); got != want {
		t.Fatalf("defaultDatabasePath() = %q, want %q", got, want)
	}
}

func TestDefaultPathsFallbackToDataDir(t *testing.T) {
	t.Setenv("OCTOPUS_DATA_DIR", "")

	if got, want := defaultDataDir(), "data"; got != want {
		t.Fatalf("defaultDataDir() = %q, want %q", got, want)
	}
	if got, want := defaultConfigPath(), filepath.Join("data", "config.json"); got != want {
		t.Fatalf("defaultConfigPath() = %q, want %q", got, want)
	}
	if got, want := defaultDatabasePath(), filepath.Join("data", "data.db"); got != want {
		t.Fatalf("defaultDatabasePath() = %q, want %q", got, want)
	}
}

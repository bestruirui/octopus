package cmd

import (
	"strings"
	"testing"

	"github.com/lingyuins/octopus/internal/conf"
)

func TestRunStartReturnsStartupError(t *testing.T) {
	originalConfig := conf.AppConfig
	t.Cleanup(func() {
		conf.AppConfig = originalConfig
	})

	conf.AppConfig = conf.Config{
		Server: conf.Server{
			Host: "127.0.0.1",
			Port: 0,
		},
		Database: conf.Database{
			Type: "invalid",
			Path: "ignored",
		},
	}

	err := runStart()
	if err == nil {
		t.Fatal("runStart() error = nil, want startup error")
	}
	if !strings.Contains(err.Error(), "unsupported database type: invalid") {
		t.Fatalf("runStart() error = %q, want unsupported database type", err)
	}
}

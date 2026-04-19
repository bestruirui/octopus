package handlers

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestClassifyGroupMutationError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
		wantOK     bool
	}{
		{
			name:       "legacy schema missing endpoint_type",
			err:        errors.New("SQL logic error: table groups has no column named endpoint_type (1)"),
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "database schema is outdated",
			wantOK:     true,
		},
		{
			name:       "duplicate group name",
			err:        errors.New("UNIQUE constraint failed: groups.name"),
			wantStatus: http.StatusConflict,
			wantMsg:    "group name already exists",
			wantOK:     true,
		},
		{
			name:       "duplicate group item",
			err:        errors.New("UNIQUE constraint failed: group_items.group_id, group_items.channel_id, group_items.model_name"),
			wantStatus: http.StatusConflict,
			wantMsg:    "group contains duplicate channel/model items",
			wantOK:     true,
		},
		{
			name:    "unexpected error",
			err:     errors.New("database is locked"),
			wantOK:  false,
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, msg, ok := classifyGroupMutationError(tt.err)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%t, got %t", tt.wantOK, ok)
			}
			if status != tt.wantStatus {
				t.Fatalf("expected status=%d, got %d", tt.wantStatus, status)
			}
			if tt.wantMsg == "" {
				if msg != "" {
					t.Fatalf("expected empty message, got %q", msg)
				}
				return
			}
			if !strings.Contains(msg, tt.wantMsg) {
				t.Fatalf("expected message containing %q, got %q", tt.wantMsg, msg)
			}
		})
	}
}

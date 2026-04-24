package handlers

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestClassifyChannelMutationError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
		wantOK     bool
	}{
		{
			name:       "channel not found",
			err:        errors.New("channel not found"),
			wantStatus: http.StatusNotFound,
			wantMsg:    "channel not found",
			wantOK:     true,
		},
		{
			name:       "invalid request rewrite profile",
			err:        errors.New("unsupported request rewrite profile: broken"),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "unsupported request rewrite profile: broken",
			wantOK:     true,
		},
		{
			name:       "unsupported request rewrite channel type",
			err:        errors.New("request rewrite profile openai_chat_compat is not supported for channel type 1"),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "request rewrite profile openai_chat_compat is not supported for channel type 1",
			wantOK:     true,
		},
		{
			name:       "duplicate channel name",
			err:        errors.New("UNIQUE constraint failed: channels.name"),
			wantStatus: http.StatusConflict,
			wantMsg:    "channel name already exists",
			wantOK:     true,
		},
		{
			name:       "legacy schema missing request_rewrite",
			err:        errors.New("SQL logic error: table channels has no column named request_rewrite (1)"),
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "database schema is outdated",
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
			status, msg, ok := classifyChannelMutationError(tt.err)
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

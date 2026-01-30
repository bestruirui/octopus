package op

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
)

func initTestDB(t *testing.T) context.Context {
	t.Helper()

	// op package uses global caches; clear them to avoid test cross-talk.
	channelCache.Clear()
	channelKeyCache.Clear()

	dbPath := filepath.Join(t.TempDir(), "octopus_test.db")
	if err := db.InitDB("sqlite", dbPath, false); err != nil {
		t.Fatalf("init db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		channelCache.Clear()
		channelKeyCache.Clear()
	})

	return context.Background()
}

func TestChannelUpdate_ParamOverrideEmptyClearsToNull(t *testing.T) {
	ctx := initTestDB(t)

	initial := "a=1"
	ch := model.Channel{
		Name:          "ch1",
		Type:          outbound.OutboundTypeOpenAIChat,
		Enabled:       true,
		Model:         "m",
		ParamOverride: &initial,
	}
	if err := ChannelCreate(&ch, ctx); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	empty := ""
	if _, err := ChannelUpdate(&model.ChannelUpdateRequest{
		ID:           ch.ID,
		ParamOverride: &empty,
	}, ctx); err != nil {
		t.Fatalf("update channel: %v", err)
	}

	var stored model.Channel
	if err := db.GetDB().WithContext(ctx).First(&stored, ch.ID).Error; err != nil {
		t.Fatalf("load channel: %v", err)
	}
	if stored.ParamOverride != nil {
		t.Fatalf("expected param_override NULL, got %q", *stored.ParamOverride)
	}
}

func TestChannelUpdate_ChannelProxyEmptyClearsToNull(t *testing.T) {
	ctx := initTestDB(t)

	initial := "http://proxy"
	ch := model.Channel{
		Name:         "ch2",
		Type:         outbound.OutboundTypeOpenAIChat,
		Enabled:      true,
		Model:        "m",
		ChannelProxy: &initial,
	}
	if err := ChannelCreate(&ch, ctx); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	empty := ""
	if _, err := ChannelUpdate(&model.ChannelUpdateRequest{
		ID:           ch.ID,
		ChannelProxy: &empty,
	}, ctx); err != nil {
		t.Fatalf("update channel: %v", err)
	}

	var stored model.Channel
	if err := db.GetDB().WithContext(ctx).First(&stored, ch.ID).Error; err != nil {
		t.Fatalf("load channel: %v", err)
	}
	if stored.ChannelProxy != nil {
		t.Fatalf("expected channel_proxy NULL, got %q", *stored.ChannelProxy)
	}
}

func TestChannelUpdate_MatchRegexEmptyClearsToNull(t *testing.T) {
	ctx := initTestDB(t)

	initial := ".*"
	ch := model.Channel{
		Name:      "ch3",
		Type:      outbound.OutboundTypeOpenAIChat,
		Enabled:   true,
		Model:     "m",
		MatchRegex: &initial,
	}
	if err := ChannelCreate(&ch, ctx); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	empty := ""
	if _, err := ChannelUpdate(&model.ChannelUpdateRequest{
		ID:        ch.ID,
		MatchRegex: &empty,
	}, ctx); err != nil {
		t.Fatalf("update channel: %v", err)
	}

	var stored model.Channel
	if err := db.GetDB().WithContext(ctx).First(&stored, ch.ID).Error; err != nil {
		t.Fatalf("load channel: %v", err)
	}
	if stored.MatchRegex != nil {
		t.Fatalf("expected match_regex NULL, got %q", *stored.MatchRegex)
	}
}

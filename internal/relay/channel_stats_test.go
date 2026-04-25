package relay

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
)

func TestUpdateChannelSuccessStatsIncludesTokenAndCost(t *testing.T) {
	testName := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", testName)

	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := op.InitCache(); err != nil {
		t.Fatalf("init cache: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	updateChannelSuccessStats(42, 321, model.StatsMetrics{
		InputToken:  70_000_000,
		OutputToken: 12_345,
		InputCost:   1.25,
		OutputCost:  2.5,
	})

	stats := op.StatsChannelGet(42)
	if stats.ChannelID != 42 {
		t.Fatalf("channel id = %d, want 42", stats.ChannelID)
	}
	if stats.RequestSuccess != 1 {
		t.Fatalf("request_success = %d, want 1", stats.RequestSuccess)
	}
	if stats.WaitTime != 321 {
		t.Fatalf("wait_time = %d, want 321", stats.WaitTime)
	}
	if stats.InputToken != 70_000_000 {
		t.Fatalf("input_token = %d, want 70000000", stats.InputToken)
	}
	if stats.OutputToken != 12_345 {
		t.Fatalf("output_token = %d, want 12345", stats.OutputToken)
	}
	if math.Abs(stats.InputCost-1.25) > 1e-9 {
		t.Fatalf("input_cost = %f, want 1.25", stats.InputCost)
	}
	if math.Abs(stats.OutputCost-2.5) > 1e-9 {
		t.Fatalf("output_cost = %f, want 2.5", stats.OutputCost)
	}
}

package relay

import (
	"testing"

	"github.com/lingyuins/octopus/internal/model"
)

func TestFinalChannelFallsBackToSkippedAttempt(t *testing.T) {
	attempts := []model.ChannelAttempt{
		{
			ChannelID:   56,
			ChannelName: "test-channel",
			Status:      model.AttemptCircuitBreak,
			Msg:         "circuit breaker tripped",
		},
	}

	channelID, channelName := finalChannel(attempts)
	if channelID != 56 || channelName != "test-channel" {
		t.Fatalf("finalChannel() = (%d, %q), want (56, %q)", channelID, channelName, "test-channel")
	}
}

func TestFinalChannelPrefersLastForwardedFailure(t *testing.T) {
	attempts := []model.ChannelAttempt{
		{
			ChannelID:   11,
			ChannelName: "failed-channel",
			Status:      model.AttemptFailed,
		},
		{
			ChannelID:   56,
			ChannelName: "skipped-channel",
			Status:      model.AttemptCircuitBreak,
		},
	}

	channelID, channelName := finalChannel(attempts)
	if channelID != 11 || channelName != "failed-channel" {
		t.Fatalf("finalChannel() = (%d, %q), want (11, %q)", channelID, channelName, "failed-channel")
	}
}

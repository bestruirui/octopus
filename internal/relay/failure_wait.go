package relay

import (
	"context"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
)

const defaultAllChannelsFailedWait = 10 * time.Second

func getAllChannelsFailedWait() time.Duration {
	seconds, err := op.SettingGetInt(model.SettingKeyAllChannelsFailedWait)
	if err != nil || seconds < 0 {
		return defaultAllChannelsFailedWait
	}
	return time.Duration(seconds) * time.Second
}

func waitBeforeAllChannelsFailed(ctx context.Context) error {
	wait := getAllChannelsFailedWait()
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

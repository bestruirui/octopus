package relay

import (
	dbmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
)

func updateChannelSuccessStats(channelID int, waitTimeMs int64, metrics dbmodel.StatsMetrics) {
	op.StatsChannelUpdate(channelID, dbmodel.StatsMetrics{
		WaitTime:       waitTimeMs,
		InputToken:     metrics.InputToken,
		OutputToken:    metrics.OutputToken,
		InputCost:      metrics.InputCost,
		OutputCost:     metrics.OutputCost,
		RequestSuccess: 1,
	})
}

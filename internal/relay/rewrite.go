package relay

import (
	"fmt"

	appmodel "github.com/lingyuins/octopus/internal/model"
	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
	"github.com/lingyuins/octopus/internal/transformer/rewrite"
)

func prepareInternalRequestForOutbound(channel *appmodel.Channel, request *transmodel.InternalLLMRequest) (*transmodel.InternalLLMRequest, error) {
	if channel == nil {
		return nil, fmt.Errorf("channel is nil")
	}
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}

	effectiveRewrite, enabled, err := rewrite.Resolve(channel.Type, channel.RequestRewrite)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return request, nil
	}

	return rewrite.Apply(request, effectiveRewrite)
}

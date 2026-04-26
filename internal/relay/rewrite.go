package relay

import (
	"fmt"

	appmodel "github.com/lingyuins/octopus/internal/model"
	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
	"github.com/lingyuins/octopus/internal/transformer/rewrite"
)

func prepareInternalRequestForOutbound(channel *appmodel.Channel, request *transmodel.InternalLLMRequest, groupEndpointType string) (*transmodel.InternalLLMRequest, error) {
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
		attachRelayGroupEndpointMetadata(request, groupEndpointType)
		return request, nil
	}

	rewritten, err := rewrite.Apply(request, effectiveRewrite)
	if err != nil {
		return nil, err
	}

	attachRelayGroupEndpointMetadata(rewritten, groupEndpointType)
	return rewritten, nil
}

func attachRelayGroupEndpointMetadata(request *transmodel.InternalLLMRequest, groupEndpointType string) {
	if request == nil {
		return
	}

	normalizedEndpointType := appmodel.NormalizeEndpointType(groupEndpointType)
	if normalizedEndpointType == "" {
		return
	}

	if request.TransformerMetadata == nil {
		request.TransformerMetadata = make(map[string]string)
	}
	request.TransformerMetadata[transmodel.TransformerMetadataGroupEndpointType] = normalizedEndpointType
}

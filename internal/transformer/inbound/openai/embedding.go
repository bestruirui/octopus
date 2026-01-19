package openai

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bestruirui/octopus/internal/transformer/model"
)

type EmbeddingInbound struct {
	// storedResponse stores the non-stream response
	storedResponse *model.InternalLLMResponse
}

// OpenAIEmbeddingRequest 是 OpenAI 标准的 embedding 请求格式
type OpenAIEmbeddingRequest struct {
	Model          string               `json:"model"`
	Input          model.EmbeddingInput `json:"input"` // 客户端使用 "input"
	Dimensions     *int64               `json:"dimensions,omitempty"`
	EncodingFormat *string              `json:"encoding_format,omitempty"`
	User           *string              `json:"user,omitempty"`
}

// OpenAIEmbeddingResponse 是 OpenAI 标准的 embedding 响应格式
type OpenAIEmbeddingResponse struct {
	ID      string                  `json:"id"`
	Object  string                  `json:"object"`
	Created int64                   `json:"created"`
	Model   string                  `json:"model"`
	Data    []model.EmbeddingObject `json:"data"` // 客户端期望 "data"
	Usage   *model.Usage            `json:"usage,omitempty"`
}

func (i *EmbeddingInbound) TransformRequest(ctx context.Context, body []byte) (*model.InternalLLMRequest, error) {
	var openAIReq OpenAIEmbeddingRequest
	if err := json.Unmarshal(body, &openAIReq); err != nil {
		return nil, err
	}

	// 转换为内部格式
	var request model.InternalLLMRequest
	request.Model = openAIReq.Model
	request.EmbeddingInput = &openAIReq.Input
	request.EmbeddingDimensions = openAIReq.Dimensions
	request.EmbeddingEncodingFormat = openAIReq.EncodingFormat
	request.User = openAIReq.User
	request.RawAPIFormat = model.APIFormatOpenAIEmbedding

	return &request, nil
}

func (i *EmbeddingInbound) TransformResponse(ctx context.Context, response *model.InternalLLMResponse) ([]byte, error) {
	// Store the response for later retrieval
	i.storedResponse = response

	// 转换为 OpenAI 标准格式
	openAIResp := OpenAIEmbeddingResponse{
		ID:      response.ID,
		Object:  response.Object,
		Created: response.Created,
		Model:   response.Model,
		Data:    response.EmbeddingData, // 使用 "data" 返回给客户端
		Usage:   response.Usage,
	}

	body, err := json.Marshal(openAIResp)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (i *EmbeddingInbound) TransformStream(ctx context.Context, stream *model.InternalLLMResponse) ([]byte, error) {
	// Embedding API does not support streaming
	return nil, errors.New("streaming is not supported for embedding API")
}

// GetInternalResponse returns the complete internal response for logging, statistics, etc.
func (i *EmbeddingInbound) GetInternalResponse(ctx context.Context) (*model.InternalLLMResponse, error) {
	return i.storedResponse, nil
}

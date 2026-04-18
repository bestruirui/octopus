package anthropic

import (
	"context"
	"testing"

	"github.com/lingyuins/octopus/internal/transformer/model"
)

func TestMessagesInboundGetInternalResponsePreservesSparseChoiceIndexes(t *testing.T) {
	first := "first"
	second := "second"
	inbound := &MessagesInbound{
		streamChunks: []*model.InternalLLMResponse{
			{
				ID:      "resp-id",
				Object:  "chat.completion.chunk",
				Created: 1,
				Model:   "claude-test",
				Choices: []model.Choice{
					{
						Index: 2,
						Delta: &model.Message{
							Role: "assistant",
							Content: model.MessageContent{
								Content: &second,
							},
						},
					},
				},
			},
			{
				ID:      "resp-id",
				Object:  "chat.completion.chunk",
				Created: 1,
				Model:   "claude-test",
				Choices: []model.Choice{
					{
						Index: 1,
						Delta: &model.Message{
							Role: "assistant",
							Content: model.MessageContent{
								Content: &first,
							},
						},
					},
				},
			},
		},
	}

	resp, err := inbound.GetInternalResponse(context.Background())
	if err != nil {
		t.Fatalf("GetInternalResponse() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetInternalResponse() response = nil")
	}
	if len(resp.Choices) != 2 {
		t.Fatalf("GetInternalResponse() choices len = %d, want 2", len(resp.Choices))
	}
	if resp.Choices[0].Index != 1 || resp.Choices[1].Index != 2 {
		t.Fatalf("GetInternalResponse() indexes = [%d %d], want [1 2]", resp.Choices[0].Index, resp.Choices[1].Index)
	}
	if resp.Choices[0].Message == nil || resp.Choices[0].Message.Content.Content == nil || *resp.Choices[0].Message.Content.Content != first {
		t.Fatalf("GetInternalResponse() first content = %+v, want %q", resp.Choices[0].Message, first)
	}
	if resp.Choices[1].Message == nil || resp.Choices[1].Message.Content.Content == nil || *resp.Choices[1].Message.Content.Content != second {
		t.Fatalf("GetInternalResponse() second content = %+v, want %q", resp.Choices[1].Message, second)
	}
}

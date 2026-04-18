package openai

import (
	"context"
	"testing"

	"github.com/lingyuins/octopus/internal/transformer/model"
)

func TestChatInboundGetInternalResponsePreservesSparseChoiceIndexes(t *testing.T) {
	inbound := &ChatInbound{
		streamChunks: []*model.InternalLLMResponse{
			makeChunkWithChoice(2, "second"),
			makeChunkWithChoice(1, "first"),
		},
	}

	resp, err := inbound.GetInternalResponse(context.Background())
	if err != nil {
		t.Fatalf("GetInternalResponse() error = %v", err)
	}

	assertChoiceOrderAndContent(t, resp, []int{1, 2}, []string{"first", "second"})
}

func TestResponseInboundGetInternalResponsePreservesSparseChoiceIndexes(t *testing.T) {
	inbound := &ResponseInbound{
		streamChunks: []*model.InternalLLMResponse{
			makeChunkWithChoice(3, "third"),
			makeChunkWithChoice(1, "first"),
		},
	}

	resp, err := inbound.GetInternalResponse(context.Background())
	if err != nil {
		t.Fatalf("GetInternalResponse() error = %v", err)
	}

	assertChoiceOrderAndContent(t, resp, []int{1, 3}, []string{"first", "third"})
}

func makeChunkWithChoice(index int, content string) *model.InternalLLMResponse {
	return &model.InternalLLMResponse{
		ID:      "resp-id",
		Object:  "chat.completion.chunk",
		Created: 1,
		Model:   "gpt-test",
		Choices: []model.Choice{
			{
				Index: index,
				Delta: &model.Message{
					Role: "assistant",
					Content: model.MessageContent{
						Content: &content,
					},
				},
			},
		},
	}
}

func assertChoiceOrderAndContent(t *testing.T, resp *model.InternalLLMResponse, wantIndexes []int, wantContents []string) {
	t.Helper()

	if resp == nil {
		t.Fatal("GetInternalResponse() response = nil")
	}
	if len(resp.Choices) != len(wantIndexes) {
		t.Fatalf("GetInternalResponse() choices len = %d, want %d", len(resp.Choices), len(wantIndexes))
	}

	for i, wantIndex := range wantIndexes {
		if resp.Choices[i].Index != wantIndex {
			t.Fatalf("GetInternalResponse() choices[%d].Index = %d, want %d", i, resp.Choices[i].Index, wantIndex)
		}
		if resp.Choices[i].Message == nil || resp.Choices[i].Message.Content.Content == nil {
			t.Fatalf("GetInternalResponse() choices[%d].Message.Content = nil", i)
		}
		if got := *resp.Choices[i].Message.Content.Content; got != wantContents[i] {
			t.Fatalf("GetInternalResponse() choices[%d].Message.Content = %q, want %q", i, got, wantContents[i])
		}
	}
}

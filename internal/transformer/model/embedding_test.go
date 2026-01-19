package model

import (
	"encoding/json"
	"testing"
)

func TestEmbeddingInput_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    EmbeddingInput
		expected string
	}{
		{
			name:     "single string",
			input:    EmbeddingInput{Single: strPtr("hello world")},
			expected: `"hello world"`,
		},
		{
			name:     "multiple strings",
			input:    EmbeddingInput{Multiple: []string{"hello", "world"}},
			expected: `["hello","world"]`,
		},
		{
			name:     "empty",
			input:    EmbeddingInput{},
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestEmbeddingInput_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected EmbeddingInput
	}{
		{
			name:     "single string",
			input:    `"hello world"`,
			expected: EmbeddingInput{Single: strPtr("hello world")},
		},
		{
			name:     "multiple strings",
			input:    `["hello","world"]`,
			expected: EmbeddingInput{Multiple: []string{"hello", "world"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input EmbeddingInput
			err := json.Unmarshal([]byte(tt.input), &input)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if input.Single != nil && tt.expected.Single != nil {
				if *input.Single != *tt.expected.Single {
					t.Errorf("expected %s, got %s", *tt.expected.Single, *input.Single)
				}
			}

			if len(input.Multiple) != len(tt.expected.Multiple) {
				t.Errorf("expected %d items, got %d", len(tt.expected.Multiple), len(input.Multiple))
			}
		})
	}
}

func TestEmbedding_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		emb      Embedding
		expected string
	}{
		{
			name:     "float array",
			emb:      Embedding{FloatArray: []float64{0.1, 0.2, 0.3}},
			expected: `[0.1,0.2,0.3]`,
		},
		{
			name:     "base64 string",
			emb:      Embedding{Base64String: strPtr("YWJjZGVm")},
			expected: `"YWJjZGVm"`,
		},
		{
			name:     "empty",
			emb:      Embedding{},
			expected: `[]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.emb)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestInternalLLMRequest_IsEmbeddingRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  InternalLLMRequest
		expected bool
	}{
		{
			name: "embedding request",
			request: InternalLLMRequest{
				Model:          "text-embedding-ada-002",
				EmbeddingInput: &EmbeddingInput{Single: strPtr("hello")},
			},
			expected: true,
		},
		{
			name: "chat request",
			request: InternalLLMRequest{
				Model:    "gpt-4",
				Messages: []Message{{Role: "user"}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.IsEmbeddingRequest()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestInternalLLMRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   InternalLLMRequest
		expectErr bool
	}{
		{
			name: "valid embedding request",
			request: InternalLLMRequest{
				Model:          "text-embedding-ada-002",
				EmbeddingInput: &EmbeddingInput{Single: strPtr("hello")},
			},
			expectErr: false,
		},
		{
			name: "valid chat request",
			request: InternalLLMRequest{
				Model:    "gpt-4",
				Messages: []Message{{Role: "user"}},
			},
			expectErr: false,
		},
		{
			name: "both messages and input",
			request: InternalLLMRequest{
				Model:          "gpt-4",
				Messages:       []Message{{Role: "user"}},
				EmbeddingInput: &EmbeddingInput{Single: strPtr("hello")},
			},
			expectErr: true,
		},
		{
			name: "neither messages nor input",
			request: InternalLLMRequest{
				Model: "gpt-4",
			},
			expectErr: true,
		},
		{
			name: "empty input",
			request: InternalLLMRequest{
				Model:          "text-embedding-ada-002",
				EmbeddingInput: &EmbeddingInput{},
			},
			expectErr: true,
		},
		{
			name:      "missing model",
			request:   InternalLLMRequest{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.expectErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestInternalLLMResponse_IsEmbeddingResponse(t *testing.T) {
	tests := []struct {
		name     string
		response InternalLLMResponse
		expected bool
	}{
		{
			name: "embedding response",
			response: InternalLLMResponse{
				Object:        "list",
				EmbeddingData: []EmbeddingObject{{Object: "embedding", Index: 0}},
			},
			expected: true,
		},
		{
			name: "chat response",
			response: InternalLLMResponse{
				Object:  "chat.completion",
				Choices: []Choice{{Index: 0}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.IsEmbeddingResponse()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

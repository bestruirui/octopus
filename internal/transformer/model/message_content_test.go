package model

import (
	"encoding/json"
	"testing"
)

func TestMessageContent_UnmarshalJSON_AllowsNull(t *testing.T) {
	var content MessageContent

	if err := json.Unmarshal([]byte("null"), &content); err != nil {
		t.Fatalf("expected null content to be accepted, got error: %v", err)
	}

	if content.Content != nil {
		t.Fatalf("expected Content to stay nil, got %q", *content.Content)
	}

	if len(content.MultipleContent) != 0 {
		t.Fatalf("expected MultipleContent to be empty, got %d parts", len(content.MultipleContent))
	}
}

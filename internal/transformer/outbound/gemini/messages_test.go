package gemini

import "testing"

func TestCleanGeminiSchemaRemovesNewlyUnsupportedKeywords(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":          "string",
				"pattern":       "^[a-z]+$",
				"minLength":     1,
				"maxLength":     32,
				"propertyNames": map[string]any{"type": "string"},
			},
			"count": map[string]any{
				"type":       "number",
				"minimum":    1,
				"maximum":    10,
				"multipleOf": 1,
			},
			"items": map[string]any{
				"type":        "array",
				"minItems":    1,
				"maxItems":    3,
				"uniqueItems": true,
				"items": map[string]any{
					"type": "string",
				},
			},
		},
	}

	cleanGeminiSchema(schema)

	nameSchema := schema["properties"].(map[string]any)["name"].(map[string]any)
	for _, key := range []string{"pattern", "minLength", "maxLength", "propertyNames"} {
		if _, exists := nameSchema[key]; exists {
			t.Fatalf("nameSchema still contains unsupported key %q", key)
		}
	}

	countSchema := schema["properties"].(map[string]any)["count"].(map[string]any)
	for _, key := range []string{"minimum", "maximum", "multipleOf"} {
		if _, exists := countSchema[key]; exists {
			t.Fatalf("countSchema still contains unsupported key %q", key)
		}
	}

	itemsSchema := schema["properties"].(map[string]any)["items"].(map[string]any)
	for _, key := range []string{"minItems", "maxItems", "uniqueItems"} {
		if _, exists := itemsSchema[key]; exists {
			t.Fatalf("itemsSchema still contains unsupported key %q", key)
		}
	}
}

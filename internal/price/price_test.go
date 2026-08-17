package price

import "testing"

func TestSelectLLMPriceURL(t *testing.T) {
	if got := selectLLMPriceURL(""); got != defaultLLMPriceURL {
		t.Fatalf("empty URL = %q, want %q", got, defaultLLMPriceURL)
	}
	if got := selectLLMPriceURL("   "); got != defaultLLMPriceURL {
		t.Fatalf("blank URL = %q, want %q", got, defaultLLMPriceURL)
	}

	const mirror = "https://mirror.example.com/models.dev/api.json"
	if got := selectLLMPriceURL("  " + mirror + "  "); got != mirror {
		t.Fatalf("configured URL = %q, want %q", got, mirror)
	}
}

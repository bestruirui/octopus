package relay

import (
	"strings"
	"testing"
)

func TestShouldMarkCompactUnsupported(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "model_not_found_503",
			err:  newTestErr("upstream error: 503: {\"error\":{\"code\":\"model_not_found\",\"message\":\"No available channel for model gpt-5.4-openai-compact\"}}"),
			want: true,
		},
		{
			name: "unsupported_path_404",
			err:  newTestErr("upstream error: 404: endpoint /v1/responses/compact not found"),
			want: true,
		},
		{
			name: "disabled_key_401_not_marked",
			err:  newTestErr("upstream error: 401: {\"code\":\"API_KEY_DISABLED\"}"),
			want: false,
		},
		{
			name: "context_canceled_not_marked",
			err:  newTestErr("failed to send request: Post \"https://a/v1/responses/compact\": context canceled"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldMarkCompactUnsupported(tt.err); got != tt.want {
				t.Fatalf("shouldMarkCompactUnsupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatUpstreamErrorMessage_CompactContext(t *testing.T) {
	body := []byte(`{"error":{"message":"This token has no access to model gpt-5.4-openai-compact"}}`)
	got := formatUpstreamErrorMessage(true, "gpt-5.4", "gpt-5.4", "/v1/responses/compact", 403, body)
	if got == "" {
		t.Fatal("empty message")
	}
	if !containsAll(got,
		"upstream error: 403:",
		`request_model="gpt-5.4"`,
		`routed_model="gpt-5.4"`,
		`endpoint="/v1/responses/compact"`,
		`note="upstream provider may internally map compact endpoint models to *-openai-compact aliases"`,
	) {
		t.Fatalf("unexpected message: %s", got)
	}
}

func TestFormatUpstreamErrorMessage_NonCompactUnchanged(t *testing.T) {
	body := []byte(`{"error":{"message":"bad"}}`)
	got := formatUpstreamErrorMessage(false, "gpt-5.4", "gpt-5.4", "/v1/responses", 400, body)
	want := `upstream error: 400: {"error":{"message":"bad"}}`
	if got != want {
		t.Fatalf("message = %s, want %s", got, want)
	}
}

type testErr string

func (e testErr) Error() string { return string(e) }

func newTestErr(v string) error { return testErr(v) }

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

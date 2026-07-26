package tailscale

import (
	"testing"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/model"
)

func TestInspectFunnel(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		wantActive   bool
		wantRunning  bool
		wantConflict bool
		wantSafe     bool
		wantURL      string
	}{
		{
			name: "matching root target",
			status: `{
				"Web":{"octopus.example.ts.net:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:8080"}}}},
				"AllowFunnel":{"octopus.example.ts.net:443":true}
			}`,
			wantActive: true, wantRunning: true, wantSafe: true,
			wantURL: "https://octopus.example.ts.net",
		},
		{
			name: "different target",
			status: `{
				"Web":{"other.example.ts.net:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:3000"}}}},
				"AllowFunnel":{"other.example.ts.net:443":true}
			}`,
			wantActive: true, wantConflict: true,
		},
		{
			name: "additional handler",
			status: `{
				"Web":{"octopus.example.ts.net:443":{"Handlers":{
					"/":{"Proxy":"8080"},"/docs":{"Proxy":"http://127.0.0.1:3000"}
				}}},
				"AllowFunnel":{"octopus.example.ts.net:443":true}
			}`,
			wantActive: true, wantConflict: true,
		},
		{
			name: "non-root target is not owned",
			status: `{
				"Web":{"octopus.example.ts.net:443":{"Handlers":{"/foo":{"Proxy":"8080"}}}},
				"AllowFunnel":{"octopus.example.ts.net:443":true}
			}`,
			wantActive: true, wantConflict: true,
		},
		{
			name: "tailnet-only Serve config blocks start",
			status: `{
				"Web":{"octopus.example.ts.net:443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:3000"}}}}
			}`,
			wantConflict: true,
		},
		{
			name: "multiple Funnel entries are not owned",
			status: `{
				"Web":{
					"octopus.example.ts.net:443":{"Handlers":{"/":{"Proxy":"8080"}}},
					"other.example.ts.net:443":{"Handlers":{"/":{"Proxy":"3000"}}}
				},
				"AllowFunnel":{"octopus.example.ts.net:443":true,"other.example.ts.net:443":true}
			}`,
			wantActive: true, wantConflict: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inspection, err := inspectFunnel([]byte(test.status), "http://localhost:8080/")
			if err != nil {
				t.Fatal(err)
			}
			if inspection.active != test.wantActive ||
				inspection.running != test.wantRunning ||
				inspection.configConflict != test.wantConflict ||
				inspection.safeToStop != test.wantSafe ||
				inspection.publicURL != test.wantURL {
				t.Fatalf("inspection = %+v", inspection)
			}
		})
	}
}

func TestLocalTargetURLUsesConfiguredBindAddress(t *testing.T) {
	previous := conf.AppConfig.Server
	defer func() { conf.AppConfig.Server = previous }()

	conf.AppConfig.Server.Host = "192.0.2.10"
	conf.AppConfig.Server.Port = 8080
	if got := localTargetURL(); got != "http://192.0.2.10:8080" {
		t.Fatalf("target URL = %q", got)
	}

	conf.AppConfig.Server.Host = "0.0.0.0"
	if got := localTargetURL(); got != "http://127.0.0.1:8080" {
		t.Fatalf("wildcard target URL = %q", got)
	}
}
func TestHasDefaultCredentials(t *testing.T) {
	user := model.User{Username: "admin", Password: "admin"}
	if err := user.HashPassword(); err != nil {
		t.Fatal(err)
	}
	if !hasDefaultCredentials(user) {
		t.Fatal("default credentials were not detected")
	}

	user.Username = "owner"
	if hasDefaultCredentials(user) {
		t.Fatal("renamed user was treated as default credentials")
	}
}

func TestDecodeCLIJSONIgnoresCommandNoise(t *testing.T) {
	var value map[string]bool
	if err := decodeCLIJSON([]byte("warning\n{\"ok\":true}\n"), &value); err != nil {
		t.Fatal(err)
	}
	if !value["ok"] {
		t.Fatalf("value = %#v", value)
	}
}

func TestExtractApprovalURL(t *testing.T) {
	message := "Enable Funnel at https://login.tailscale.com/admin/settings/features)."
	if got := extractApprovalURL(message); got != "https://login.tailscale.com/admin/settings/features" {
		t.Fatalf("approval URL = %q", got)
	}
}

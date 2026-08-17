package model

import "testing"

func TestModelPriceURLSettingValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty uses default", value: ""},
		{name: "https mirror", value: "https://mirror.example.com/models.dev/api.json"},
		{name: "http private mirror", value: "http://192.168.1.10:8080/api.json"},
		{name: "reject ftp", value: "ftp://mirror.example.com/api.json", wantErr: true},
		{name: "reject missing scheme", value: "mirror.example.com/api.json", wantErr: true},
		{name: "reject missing host", value: "https:///api.json", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setting := Setting{Key: SettingKeyModelPriceURL, Value: tc.value}
			err := setting.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

package model

import "testing"

func TestDefaultSettings_ContainsAllChannelsFailedWait(t *testing.T) {
	settings := DefaultSettings()
	for _, setting := range settings {
		if setting.Key == SettingKeyAllChannelsFailedWait {
			if setting.Value != "10" {
				t.Fatalf("expected default wait to be 10, got %q", setting.Value)
			}
			return
		}
	}

	t.Fatal("expected all_channels_failed_wait setting to exist")
}

func TestSettingValidate_AllChannelsFailedWait(t *testing.T) {
	valid := &Setting{Key: SettingKeyAllChannelsFailedWait, Value: "15"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid integer setting, got error: %v", err)
	}

	invalid := &Setting{Key: SettingKeyAllChannelsFailedWait, Value: "abc"}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected invalid integer setting to fail validation")
	}
}

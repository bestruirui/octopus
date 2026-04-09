package op

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
)

func TestResolveGroupRuntimeOptions(t *testing.T) {
	settingCache.Clear()
	settingCache.Set(model.SettingKeyGroupDefaultFirstTokenTimeOut, "12")
	settingCache.Set(model.SettingKeyGroupDefaultSessionKeepTime, "300")

	t.Run("inherit defaults when group values missing", func(t *testing.T) {
		firstTokenTimeOut, sessionKeepTime := ResolveGroupRuntimeOptions(model.Group{})
		if firstTokenTimeOut != 12 || sessionKeepTime != 300 {
			t.Fatalf("expected defaults 12/300, got %d/%d", firstTokenTimeOut, sessionKeepTime)
		}
	})

	t.Run("custom values override defaults independently", func(t *testing.T) {
		firstTokenTimeOut, sessionKeepTime := ResolveGroupRuntimeOptions(model.Group{
			FirstTokenTimeOut: 18,
			SessionKeepTime:   0,
		})
		if firstTokenTimeOut != 18 || sessionKeepTime != 300 {
			t.Fatalf("expected 18/300, got %d/%d", firstTokenTimeOut, sessionKeepTime)
		}

		firstTokenTimeOut, sessionKeepTime = ResolveGroupRuntimeOptions(model.Group{
			FirstTokenTimeOut: 0,
			SessionKeepTime:   90,
		})
		if firstTokenTimeOut != 12 || sessionKeepTime != 90 {
			t.Fatalf("expected 12/90, got %d/%d", firstTokenTimeOut, sessionKeepTime)
		}
	})
}

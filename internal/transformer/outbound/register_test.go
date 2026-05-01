package outbound

import "testing"

func TestGet_MimoUsesOpenAIChatOutbound(t *testing.T) {
	got := Get(OutboundTypeMimo)
	if got == nil {
		t.Fatal("Get(OutboundTypeMimo) = nil, want outbound instance")
	}
}

func TestIsChatChannelType_Mimo(t *testing.T) {
	if !IsChatChannelType(OutboundTypeMimo) {
		t.Fatal("IsChatChannelType(OutboundTypeMimo) = false, want true")
	}
}

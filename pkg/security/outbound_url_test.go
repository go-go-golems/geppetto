package security

import "testing"

func TestValidateOutboundURLRejectsZonedIPv6ByDefault(t *testing.T) {
	err := ValidateOutboundURL("https://[fe80::1%25eth0]/", OutboundURLOptions{})
	if err == nil {
		t.Fatal("expected zone-literal IPv6 host to be rejected")
	}
}

func TestValidateOutboundURLAllowsZonedIPv6WhenLocalNetworksAllowed(t *testing.T) {
	err := ValidateOutboundURL("https://[fe80::1%25eth0]/", OutboundURLOptions{
		AllowLocalNetworks: true,
	})
	if err != nil {
		t.Fatalf("expected zone-literal IPv6 host to be allowed when local networks are enabled: %v", err)
	}
}

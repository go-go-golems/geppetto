package security

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

// OutboundURLOptions configures outbound request URL validation.
type OutboundURLOptions struct {
	// AllowHTTP permits plain HTTP URLs. HTTPS is always allowed.
	AllowHTTP bool
	// AllowLocalNetworks permits loopback/private/link-local IP targets and localhost hostnames.
	AllowLocalNetworks bool
}

// ValidateOutboundURL validates that a URL is safe for outbound provider requests.
// It rejects unsafe schemes and local-network targets unless explicitly allowed.
func ValidateOutboundURL(rawURL string, opts OutboundURLOptions) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	switch parsed.Scheme {
	case "https":
	case "http":
		if !opts.AllowHTTP {
			return fmt.Errorf("http scheme is not allowed")
		}
	default:
		return fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("URL host is required")
	}

	if !opts.AllowLocalNetworks {
		if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
			return fmt.Errorf("local hostname %q is not allowed", host)
		}
	}

	// When host is an IP literal, enforce network restrictions without DNS lookups.
	if addr, err := netip.ParseAddr(host); err == nil {
		if addr.Zone() != "" && !opts.AllowLocalNetworks {
			return fmt.Errorf("zoned IP address %q is not allowed", host)
		}
		addr = addr.Unmap()

		if addr.IsUnspecified() || addr.IsMulticast() {
			return fmt.Errorf("disallowed IP address %q", host)
		}

		if !opts.AllowLocalNetworks {
			if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
				return fmt.Errorf("local network IP %q is not allowed", host)
			}
		}
	}

	return nil
}

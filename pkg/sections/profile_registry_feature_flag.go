package sections

import (
	"os"
	"strings"
)

const profileRegistryMiddlewareEnv = "PINOCCHIO_ENABLE_PROFILE_REGISTRY_MIDDLEWARE"

func isProfileRegistryMiddlewareEnabled() bool {
	raw := strings.TrimSpace(os.Getenv(profileRegistryMiddlewareEnv))
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

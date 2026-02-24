package profiles

// RegistrySlugFromString adapts string-based call sites (CLI/HTTP) into typed slugs.
func RegistrySlugFromString(raw string) (RegistrySlug, error) {
	return ParseRegistrySlug(raw)
}

// ProfileSlugFromString adapts string-based call sites (CLI/HTTP) into typed slugs.
func ProfileSlugFromString(raw string) (ProfileSlug, error) {
	return ParseProfileSlug(raw)
}

// RuntimeKeyFromString adapts string-based call sites (CLI/HTTP) into typed runtime keys.
func RuntimeKeyFromString(raw string) (RuntimeKey, error) {
	return ParseRuntimeKey(raw)
}

// RegistrySlugToString adapts typed slugs back to plain strings for external APIs.
func RegistrySlugToString(slug RegistrySlug) string {
	return slug.String()
}

// ProfileSlugToString adapts typed slugs back to plain strings for external APIs.
func ProfileSlugToString(slug ProfileSlug) string {
	return slug.String()
}

// RuntimeKeyToString adapts typed runtime keys back to plain strings for external APIs.
func RuntimeKeyToString(key RuntimeKey) string {
	return key.String()
}

package profiles

import "testing"

func TestAdapterHelpersRoundTrip(t *testing.T) {
	r, err := RegistrySlugFromString("Default")
	if err != nil {
		t.Fatalf("RegistrySlugFromString failed: %v", err)
	}
	if RegistrySlugToString(r) != "default" {
		t.Fatalf("registry round-trip mismatch: %q", r)
	}

	p, err := ProfileSlugFromString("Agent")
	if err != nil {
		t.Fatalf("ProfileSlugFromString failed: %v", err)
	}
	if ProfileSlugToString(p) != "agent" {
		t.Fatalf("profile round-trip mismatch: %q", p)
	}

	k, err := RuntimeKeyFromString("Agent")
	if err != nil {
		t.Fatalf("RuntimeKeyFromString failed: %v", err)
	}
	if RuntimeKeyToString(k) != "agent" {
		t.Fatalf("runtime key round-trip mismatch: %q", k)
	}
}

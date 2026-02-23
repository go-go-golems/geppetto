package sections

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func TestGatherFlagsFromProfileRegistry_LegacyProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profiles.yaml")
	content := `default:
  ai-chat:
    ai-engine: default-engine
agent:
  ai-chat:
    ai-engine: profile-engine
`
	if err := os.WriteFile(profilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	schema_ := mustGeppettoSchema(t)
	parsed := values.New()
	err := sources.Execute(
		schema_,
		parsed,
		GatherFlagsFromProfileRegistry(profilePath, profilePath, "agent", "default"),
		sources.FromDefaults(),
	)
	if err != nil {
		t.Fatalf("sources.Execute returned error: %v", err)
	}

	ss, err := settings.NewStepSettingsFromParsedValues(parsed)
	if err != nil {
		t.Fatalf("NewStepSettingsFromParsedValues returned error: %v", err)
	}
	if ss.Chat == nil || ss.Chat.Engine == nil || *ss.Chat.Engine != "profile-engine" {
		t.Fatalf("expected profile engine override, got %#v", ss.Chat)
	}
}

func TestGatherFlagsFromProfileRegistry_DefaultProfileMissingReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profiles.yaml")
	content := `agent:
  ai-chat:
    ai-engine: profile-engine
`
	if err := os.WriteFile(profilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	schema_ := mustGeppettoSchema(t)
	err := sources.Execute(
		schema_,
		values.New(),
		GatherFlagsFromProfileRegistry(profilePath, profilePath, "default", "default"),
		sources.FromDefaults(),
	)
	if err != nil {
		t.Fatalf("expected nil error for missing default profile, got %v", err)
	}
}

func TestGatherFlagsFromProfileRegistry_MissingNonDefaultProfileErrors(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profiles.yaml")
	content := `default:
  ai-chat:
    ai-engine: default-engine
`
	if err := os.WriteFile(profilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	schema_ := mustGeppettoSchema(t)
	err := sources.Execute(
		schema_,
		values.New(),
		GatherFlagsFromProfileRegistry(profilePath, profilePath, "agent", "default"),
		sources.FromDefaults(),
	)
	if err == nil {
		t.Fatalf("expected error for missing non-default profile")
	}
}

func TestIsProfileRegistryMiddlewareEnabled(t *testing.T) {
	t.Setenv(profileRegistryMiddlewareEnv, "")
	if isProfileRegistryMiddlewareEnabled() {
		t.Fatalf("expected feature flag disabled for empty value")
	}

	for _, raw := range []string{"1", "true", "yes", "on", "TRUE"} {
		t.Setenv(profileRegistryMiddlewareEnv, raw)
		if !isProfileRegistryMiddlewareEnabled() {
			t.Fatalf("expected feature flag enabled for %q", raw)
		}
	}
}

func mustGeppettoSchema(t *testing.T) *schema.Schema {
	t.Helper()
	sections, err := CreateGeppettoSections()
	if err != nil {
		t.Fatalf("CreateGeppettoSections returned error: %v", err)
	}
	return schema.NewSchema(schema.WithSections(sections...))
}

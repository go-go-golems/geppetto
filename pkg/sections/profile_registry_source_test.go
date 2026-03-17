package sections

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/spf13/cobra"
)

func TestGatherFlagsFromProfileRegistry_IsNoOp(t *testing.T) {
	schema_ := mustGeppettoSchema(t)
	parsed := values.New()
	err := sources.Execute(
		schema_,
		parsed,
		GatherFlagsFromProfileRegistry([]string{"unused.yaml"}, "agent"),
		sources.FromDefaults(),
	)
	if err != nil {
		t.Fatalf("sources.Execute returned error: %v", err)
	}

	ss, err := settings.NewStepSettingsFromParsedValues(parsed)
	if err != nil {
		t.Fatalf("NewStepSettingsFromParsedValues returned error: %v", err)
	}
	if ss.Chat == nil || ss.Chat.Engine == nil || *ss.Chat.Engine == "" {
		t.Fatalf("expected default step settings to survive, got %#v", ss.Chat)
	}
}

func TestGetCobraCommandGeppettoMiddlewares_CobraWinsWithoutProfileOverlay(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profiles.yaml")

	profilesYAML := `slug: private
profiles:
  default:
    slug: default
    runtime:
      system_prompt: hello
`
	if err := os.WriteFile(profilePath, []byte(profilesYAML), 0o644); err != nil {
		t.Fatalf("WriteFile profiles returned error: %v", err)
	}

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("config", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flags().String("profile-registries", "", "")
	cmd.Flags().String("ai-engine", "", "")
	if err := cmd.Flags().Set("profile-registries", profilePath); err != nil {
		t.Fatalf("Set profile-registries flag failed: %v", err)
	}
	if err := cmd.Flags().Set("profile", "default"); err != nil {
		t.Fatalf("Set profile flag failed: %v", err)
	}
	if err := cmd.Flags().Set("ai-engine", "cobra-engine"); err != nil {
		t.Fatalf("Set ai-engine flag failed: %v", err)
	}
	schema_ := mustGeppettoSchema(t)

	middlewares, err := GetCobraCommandGeppettoMiddlewares(nil, cmd, []string{})
	if err != nil {
		t.Fatalf("GetCobraCommandGeppettoMiddlewares returned error: %v", err)
	}
	finalParsed := values.New()
	if err := sources.Execute(schema_, finalParsed, middlewares...); err != nil {
		t.Fatalf("sources.Execute returned error: %v", err)
	}

	ss, err := settings.NewStepSettingsFromParsedValues(finalParsed)
	if err != nil {
		t.Fatalf("NewStepSettingsFromParsedValues returned error: %v", err)
	}
	if ss.Chat == nil || ss.Chat.Engine == nil || *ss.Chat.Engine != "cobra-engine" {
		t.Fatalf("expected cobra engine to survive, got %#v", ss.Chat)
	}
}

func mustGeppettoSchema(t *testing.T) *schema.Schema {
	t.Helper()
	sections_, err := CreateGeppettoSections()
	if err != nil {
		t.Fatalf("CreateGeppettoSections returned error: %v", err)
	}
	return schema.NewSchema(schema.WithSections(sections_...))
}

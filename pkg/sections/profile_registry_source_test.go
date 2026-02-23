package sections

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/spf13/cobra"
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

func TestGetCobraCommandGeppettoMiddlewares_ProfileOrderingWithRegistryAdapter(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profiles.yaml")
	configPath := filepath.Join(tmpDir, "config.yaml")

	profilesYAML := `agent:
  ai-chat:
    ai-engine: profile-engine
`
	if err := os.WriteFile(profilePath, []byte(profilesYAML), 0o644); err != nil {
		t.Fatalf("WriteFile profiles returned error: %v", err)
	}

	configYAML := "profile-settings:\n" +
		"  profile-file: " + profilePath + "\n" +
		"  profile: agent\n" +
		"ai-chat:\n" +
		"  ai-engine: config-engine\n"
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("WriteFile config returned error: %v", err)
	}
	oldEnv, hadEnv := os.LookupEnv("PINOCCHIO_AI_ENGINE")
	defer func() {
		if hadEnv {
			_ = os.Setenv("PINOCCHIO_AI_ENGINE", oldEnv)
		} else {
			_ = os.Unsetenv("PINOCCHIO_AI_ENGINE")
		}
	}()

	parseEngine := func(args []string, envEngine string) string {
		t.Helper()
		_ = os.Unsetenv("PINOCCHIO_AI_ENGINE")
		if envEngine != "" {
			_ = os.Setenv("PINOCCHIO_AI_ENGINE", envEngine)
		}

		cmd := &cobra.Command{Use: "test"}
		schema_ := mustGeppettoSchemaWithCommandAndProfile(t)
		addSchemaFlagsToCommand(t, schema_, cmd)
		if err := cmd.ParseFlags(args); err != nil {
			t.Fatalf("ParseFlags returned error: %v", err)
		}

		parsedCommandSections, err := cli.ParseCommandSettingsSection(cmd)
		if err != nil {
			t.Fatalf("ParseCommandSettingsSection returned error: %v", err)
		}
		middlewares_, err := GetCobraCommandGeppettoMiddlewares(parsedCommandSections, cmd, nil)
		if err != nil {
			t.Fatalf("GetCobraCommandGeppettoMiddlewares returned error: %v", err)
		}

		parsedValues := values.New()
		if err := sources.Execute(schema_, parsedValues, middlewares_...); err != nil {
			t.Fatalf("sources.Execute returned error: %v", err)
		}

		ss, err := settings.NewStepSettingsFromParsedValues(parsedValues)
		if err != nil {
			t.Fatalf("NewStepSettingsFromParsedValues returned error: %v", err)
		}
		if ss.Chat == nil || ss.Chat.Engine == nil {
			t.Fatalf("expected chat engine to be set")
		}
		return *ss.Chat.Engine
	}

	baseArgs := []string{"--config-file", configPath}
	if got := parseEngine(baseArgs, ""); got != "profile-engine" {
		t.Fatalf("expected profile to override config, got %q", got)
	}
	if got := parseEngine(baseArgs, "env-engine"); got != "env-engine" {
		t.Fatalf("expected env to override profile, got %q", got)
	}
	if got := parseEngine(append(baseArgs, "--ai-engine", "flag-engine"), "env-engine"); got != "flag-engine" {
		t.Fatalf("expected flags to override env/profile/config, got %q", got)
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

func mustGeppettoSchemaWithCommandAndProfile(t *testing.T) *schema.Schema {
	t.Helper()
	sections, err := CreateGeppettoSections()
	if err != nil {
		t.Fatalf("CreateGeppettoSections returned error: %v", err)
	}
	commandSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		t.Fatalf("NewCommandSettingsSection returned error: %v", err)
	}
	profileSection, err := cli.NewProfileSettingsSection()
	if err != nil {
		t.Fatalf("NewProfileSettingsSection returned error: %v", err)
	}
	allSections := append([]schema.Section{}, sections...)
	allSections = append(allSections, commandSection, profileSection)
	return schema.NewSchema(schema.WithSections(allSections...))
}

func addSchemaFlagsToCommand(t *testing.T, schema_ *schema.Schema, cmd *cobra.Command) {
	t.Helper()
	err := schema_.ForEachE(func(_ string, section schema.Section) error {
		cobraSection, ok := section.(schema.CobraSection)
		if !ok {
			return nil
		}
		return cobraSection.AddSectionToCobraCommand(cmd)
	})
	if err != nil {
		t.Fatalf("failed to add schema flags to command: %v", err)
	}
}

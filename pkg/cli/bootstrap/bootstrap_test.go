package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func testAppBootstrapConfig() AppBootstrapConfig {
	return AppBootstrapConfig{
		AppName:          "gp53app",
		EnvPrefix:        "GP53APP",
		ConfigFileMapper: testConfigFileMapper,
		NewProfileSection: func() (schema.Section, error) {
			return geppettosections.NewProfileSettingsSection()
		},
		BuildBaseSections: func() ([]schema.Section, error) {
			return geppettosections.CreateGeppettoSections()
		},
	}
}

func testConfigFileMapper(rawConfig interface{}) (map[string]map[string]interface{}, error) {
	configMap, ok := rawConfig.(map[string]interface{})
	if !ok {
		return nil, nil
	}
	result := make(map[string]map[string]interface{})
	for key, value := range configMap {
		layerParams, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		result[key] = layerParams
	}
	return result, nil
}

func TestNewCLISelectionValuesBuildsCommandAndProfileSections(t *testing.T) {
	cfg := testAppBootstrapConfig()
	parsed, err := NewCLISelectionValues(cfg, CLISelectionInput{
		ConfigFile:        "custom.yaml",
		Profile:           " analyst ",
		ProfileRegistries: []string{" one.yaml ", "", "two.yaml"},
	})
	if err != nil {
		t.Fatalf("NewCLISelectionValues failed: %v", err)
	}

	commandSettings := &cli.CommandSettings{}
	if err := parsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err != nil {
		t.Fatalf("decode command settings: %v", err)
	}
	if got := commandSettings.ConfigFile; got != "custom.yaml" {
		t.Fatalf("expected config file custom.yaml, got %q", got)
	}

	profileSettings := ResolveProfileSettings(parsed)
	if got := profileSettings.Profile; got != "analyst" {
		t.Fatalf("expected trimmed profile analyst, got %q", got)
	}
	if len(profileSettings.ProfileRegistries) != 2 || profileSettings.ProfileRegistries[0] != "one.yaml" || profileSettings.ProfileRegistries[1] != "two.yaml" {
		t.Fatalf("expected normalized registries, got %#v", profileSettings.ProfileRegistries)
	}
}

func TestResolveCLIConfigFiles_UsesAppNameForDefaultDiscovery(t *testing.T) {
	cfg := testAppBootstrapConfig()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "xdg"))
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, "xdg", cfg.AppName, "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("profile-settings: {}\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	files, err := ResolveCLIConfigFiles(cfg, values.New())
	if err != nil {
		t.Fatalf("ResolveCLIConfigFiles failed: %v", err)
	}
	if len(files) != 1 || files[0] != configPath {
		t.Fatalf("expected discovered config path, got %#v", files)
	}
}

func TestResolveCLIProfileSelection_UsesConfiguredEnvPrefix(t *testing.T) {
	cfg := testAppBootstrapConfig()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "xdg"))
	t.Setenv("HOME", tmpDir)
	t.Setenv("GP53APP_PROFILE", "env-profile")
	t.Setenv("GP53APP_PROFILE_REGISTRIES", filepath.Join(tmpDir, "env-registry.yaml"))

	resolved, err := ResolveCLIProfileSelection(cfg, values.New())
	if err != nil {
		t.Fatalf("ResolveCLIProfileSelection failed: %v", err)
	}
	if got := resolved.Profile; got != "env-profile" {
		t.Fatalf("expected env profile, got %q", got)
	}
	if len(resolved.ProfileRegistries) != 1 || resolved.ProfileRegistries[0] != filepath.Join(tmpDir, "env-registry.yaml") {
		t.Fatalf("expected env registries, got %#v", resolved.ProfileRegistries)
	}
}

func TestResolveCLIProfileSelection_DoesNotUseImplicitProfilesFallback(t *testing.T) {
	cfg := testAppBootstrapConfig()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "xdg"))
	t.Setenv("HOME", tmpDir)

	registryPath := filepath.Join(tmpDir, "xdg", cfg.AppName, "profiles.yaml")
	if err := os.MkdirAll(filepath.Dir(registryPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(registryPath, []byte("slug: workspace\nprofiles: {}\n"), 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	parsed := values.New()
	commandSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		t.Fatalf("command section: %v", err)
	}
	commandValues, err := values.NewSectionValues(commandSection)
	if err != nil {
		t.Fatalf("command values: %v", err)
	}
	parsed.Set(cli.CommandSettingsSlug, commandValues)

	resolved, err := ResolveCLIProfileSelection(cfg, parsed)
	if err != nil {
		t.Fatalf("ResolveCLIProfileSelection failed: %v", err)
	}
	if len(resolved.ProfileRegistries) != 0 {
		t.Fatalf("expected no implicit registry fallback, got %#v", resolved.ProfileRegistries)
	}
}

func TestResolveCLIEngineSettings_RejectsProfileWithoutRegistries(t *testing.T) {
	cfg := testAppBootstrapConfig()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "xdg"))
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, "base.yaml")
	if err := os.WriteFile(configPath, []byte("ai-chat:\n  ai-api-type: openai\n  ai-engine: base-model\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	parsed, err := NewCLISelectionValues(cfg, CLISelectionInput{
		ConfigFile: configPath,
		Profile:    "analyst",
	})
	if err != nil {
		t.Fatalf("NewCLISelectionValues failed: %v", err)
	}

	_, err = ResolveCLIEngineSettings(context.Background(), cfg, parsed)
	if err == nil {
		t.Fatal("expected profile without registries to fail")
	}
}

func TestResolveCLIEngineSettings_UsesBaseOnlyModeWhenNoRegistriesConfigured(t *testing.T) {
	cfg := testAppBootstrapConfig()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "xdg"))
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, "base.yaml")
	if err := os.WriteFile(configPath, []byte("ai-chat:\n  ai-api-type: openai\n  ai-engine: base-model\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	parsed, err := NewCLISelectionValues(cfg, CLISelectionInput{
		ConfigFile: configPath,
	})
	if err != nil {
		t.Fatalf("NewCLISelectionValues failed: %v", err)
	}

	resolved, err := ResolveCLIEngineSettings(context.Background(), cfg, parsed)
	if err != nil {
		t.Fatalf("ResolveCLIEngineSettings failed: %v", err)
	}
	if resolved.FinalInferenceSettings == nil || resolved.FinalInferenceSettings.Chat == nil || resolved.FinalInferenceSettings.Chat.Engine == nil {
		t.Fatal("expected final inference settings with chat engine")
	}
	if got := *resolved.FinalInferenceSettings.Chat.Engine; got != "base-model" {
		t.Fatalf("expected base model, got %q", got)
	}
	if resolved.ResolvedEngineProfile != nil {
		t.Fatalf("expected no resolved engine profile, got %#v", resolved.ResolvedEngineProfile)
	}
}

func TestResolveCLIEngineSettingsFromBase_MatchesResolvedPath(t *testing.T) {
	cfg := testAppBootstrapConfig()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "xdg"))
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, "base.yaml")
	if err := os.WriteFile(configPath, []byte("ai-chat:\n  ai-api-type: openai\n  ai-engine: base-model\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	registryPath := filepath.Join(tmpDir, "profiles.yaml")
	registryYAML := `
slug: workspace
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: openai-responses
        engine: gpt-5-mini
`
	if err := os.WriteFile(registryPath, []byte(registryYAML), 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	parsed, err := NewCLISelectionValues(cfg, CLISelectionInput{
		ConfigFile:        configPath,
		Profile:           "default",
		ProfileRegistries: []string{registryPath},
	})
	if err != nil {
		t.Fatalf("NewCLISelectionValues failed: %v", err)
	}

	base, configFiles, err := ResolveBaseInferenceSettings(cfg, parsed)
	if err != nil {
		t.Fatalf("ResolveBaseInferenceSettings failed: %v", err)
	}

	resolvedDirect, err := ResolveCLIEngineSettings(context.Background(), cfg, parsed)
	if err != nil {
		t.Fatalf("ResolveCLIEngineSettings failed: %v", err)
	}
	if resolvedDirect.Close != nil {
		defer resolvedDirect.Close()
	}

	resolvedFromBase, err := ResolveCLIEngineSettingsFromBase(context.Background(), cfg, base, parsed, configFiles)
	if err != nil {
		t.Fatalf("ResolveCLIEngineSettingsFromBase failed: %v", err)
	}
	if resolvedFromBase.Close != nil {
		defer resolvedFromBase.Close()
	}

	if got, want := *resolvedFromBase.BaseInferenceSettings.Chat.Engine, *resolvedDirect.BaseInferenceSettings.Chat.Engine; got != want {
		t.Fatalf("base engine mismatch: got %q want %q", got, want)
	}
	if got, want := *resolvedFromBase.FinalInferenceSettings.Chat.Engine, *resolvedDirect.FinalInferenceSettings.Chat.Engine; got != want {
		t.Fatalf("final engine mismatch: got %q want %q", got, want)
	}
	if got, want := string(*resolvedFromBase.FinalInferenceSettings.Chat.ApiType), string(*resolvedDirect.FinalInferenceSettings.Chat.ApiType); got != want {
		t.Fatalf("final api type mismatch: got %q want %q", got, want)
	}
}

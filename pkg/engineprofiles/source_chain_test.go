package engineprofiles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEngineProfileRegistrySourceEntries(t *testing.T) {
	entries, err := ParseEngineProfileRegistrySourceEntries(" a.yaml, b.db ,sqlite-dsn:file:test.db ")
	if err != nil {
		t.Fatalf("ParseEngineProfileRegistrySourceEntries failed: %v", err)
	}
	if got, want := len(entries), 3; got != want {
		t.Fatalf("entry count mismatch: got=%d want=%d", got, want)
	}
	if entries[0] != "a.yaml" || entries[1] != "b.db" || entries[2] != "sqlite-dsn:file:test.db" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestParseRegistrySourceSpecs_AutodetectAndPrefixes(t *testing.T) {
	tmpDir := t.TempDir()
	sqlitePath := filepath.Join(tmpDir, "profiles.db")
	store, err := NewSQLiteEngineProfileStore("file:"+sqlitePath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore failed: %v", err)
	}
	_ = store.Close()

	specs, err := ParseRegistrySourceSpecs([]string{
		"yaml://" + filepath.Join(tmpDir, "profiles-urlish.yaml"),
		"yaml:" + filepath.Join(tmpDir, "profiles.yaml"),
		sqlitePath,
		"sqlite-dsn:file:test.db?_journal_mode=WAL",
	})
	if err != nil {
		t.Fatalf("ParseRegistrySourceSpecs failed: %v", err)
	}
	if got, want := len(specs), 4; got != want {
		t.Fatalf("spec count mismatch: got=%d want=%d", got, want)
	}
	if specs[0].Kind != RegistrySourceKindYAML {
		t.Fatalf("spec[0] kind mismatch: %q", specs[0].Kind)
	}
	if specs[1].Kind != RegistrySourceKindYAML {
		t.Fatalf("spec[1] kind mismatch: %q", specs[1].Kind)
	}
	if specs[2].Kind != RegistrySourceKindSQLite {
		t.Fatalf("spec[2] kind mismatch: %q", specs[2].Kind)
	}
	if specs[3].Kind != RegistrySourceKindSQLiteDSN {
		t.Fatalf("spec[3] kind mismatch: %q", specs[3].Kind)
	}
}

func TestParseRegistrySourceSpecs_YAMLURLishRelativePathDoesNotProduceDoubleSlash(t *testing.T) {
	specs, err := ParseRegistrySourceSpecs([]string{"yaml://./profile-registry.yaml"})
	if err != nil {
		t.Fatalf("ParseRegistrySourceSpecs failed: %v", err)
	}
	if got, want := len(specs), 1; got != want {
		t.Fatalf("spec count mismatch: got=%d want=%d", got, want)
	}
	if specs[0].Kind != RegistrySourceKindYAML {
		t.Fatalf("spec kind mismatch: %q", specs[0].Kind)
	}
	if specs[0].Path != "./profile-registry.yaml" {
		t.Fatalf("unexpected yaml path: %q", specs[0].Path)
	}
	if strings.HasPrefix(specs[0].Path, "//") {
		t.Fatalf("unexpected double-slash path: %q", specs[0].Path)
	}
}

func TestDecodeRuntimeYAMLSingleRegistry_StrictFormat(t *testing.T) {
	_, err := DecodeRuntimeYAMLSingleRegistry([]byte(`registries:
  default:
    slug: default
`))
	if err == nil {
		t.Fatalf("expected error for registries bundle")
	}

	_, err = DecodeRuntimeYAMLSingleRegistry([]byte(`slug: default
default_profile_slug: default
profiles:
  default:
    slug: default
`))
	if err == nil {
		t.Fatalf("expected error for default_profile_slug")
	}

	_, err = DecodeRuntimeYAMLSingleRegistry([]byte(`default:
  ai-chat:
    ai-engine: x
`))
	if err == nil {
		t.Fatalf("expected error for legacy profile map")
	}

	reg, err := DecodeRuntimeYAMLSingleRegistry([]byte(`slug: private
profiles:
  default:
    slug: default
    runtime:
      system_prompt: test
`))
	if err != nil {
		t.Fatalf("DecodeRuntimeYAMLSingleRegistry failed: %v", err)
	}
	if reg == nil {
		t.Fatalf("expected registry")
	}
	if reg.Slug != MustRegistrySlug("private") {
		t.Fatalf("registry slug mismatch: %q", reg.Slug)
	}
	if reg.DefaultEngineProfileSlug != MustEngineProfileSlug("default") {
		t.Fatalf("default profile slug mismatch: %q", reg.DefaultEngineProfileSlug)
	}
}

func TestChainedRegistry_ResolveTopOfStackAndWriteRouting(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	yamlPath := filepath.Join(tmpDir, "private.yaml")
	if err := os.WriteFile(yamlPath, []byte(`slug: private
profiles:
  default:
    slug: default
    runtime:
      system_prompt: private-default
  analyst:
    slug: analyst
    runtime:
      system_prompt: private-analyst
`), 0o644); err != nil {
		t.Fatalf("WriteFile yaml failed: %v", err)
	}

	sqlitePath := filepath.Join(tmpDir, "shared.db")
	dsn, err := SQLiteProfileDSNForFile(sqlitePath)
	if err != nil {
		t.Fatalf("SQLiteProfileDSNForFile failed: %v", err)
	}
	sqliteStore, err := NewSQLiteEngineProfileStore(dsn, MustRegistrySlug("default"))
	if err != nil {
		t.Fatalf("NewSQLiteEngineProfileStore failed: %v", err)
	}
	defer func() {
		_ = sqliteStore.Close()
	}()

	shared := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("shared"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug: MustEngineProfileSlug("default"),
				Runtime: RuntimeSpec{
					SystemPrompt: "shared-default",
				},
			},
			MustEngineProfileSlug("analyst"): {
				Slug: MustEngineProfileSlug("analyst"),
				Runtime: RuntimeSpec{
					SystemPrompt: "shared-analyst",
				},
			},
		},
	}
	if err := sqliteStore.UpsertRegistry(ctx, shared, SaveOptions{Actor: "tests", Source: "tests"}); err != nil {
		t.Fatalf("UpsertRegistry shared failed: %v", err)
	}

	specs, err := ParseRegistrySourceSpecs([]string{sqlitePath, yamlPath})
	if err != nil {
		t.Fatalf("ParseRegistrySourceSpecs failed: %v", err)
	}
	chain, err := NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		t.Fatalf("NewChainedRegistryFromSourceSpecs failed: %v", err)
	}
	defer func() {
		_ = chain.Close()
	}()

	resolvedAnalyst, err := chain.ResolveEngineProfile(ctx, ResolveInput{EngineProfileSlug: MustEngineProfileSlug("analyst")})
	if err != nil {
		t.Fatalf("ResolveEngineProfile analyst failed: %v", err)
	}
	if got, want := resolvedAnalyst.RegistrySlug, MustRegistrySlug("private"); got != want {
		t.Fatalf("analyst registry mismatch: got=%q want=%q", got, want)
	}
	if got, want := resolvedAnalyst.EffectiveRuntime.SystemPrompt, "private-analyst"; got != want {
		t.Fatalf("analyst prompt mismatch: got=%q want=%q", got, want)
	}

	resolvedDefault, err := chain.ResolveEngineProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEngineProfile default failed: %v", err)
	}
	if got, want := resolvedDefault.RegistrySlug, MustRegistrySlug("private"); got != want {
		t.Fatalf("default registry mismatch: got=%q want=%q", got, want)
	}
	if got, want := resolvedDefault.EffectiveRuntime.SystemPrompt, "private-default"; got != want {
		t.Fatalf("default prompt mismatch: got=%q want=%q", got, want)
	}

}

func TestChainedRegistry_RejectsDuplicateRegistrySlugsAcrossSources(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	pathA := filepath.Join(tmpDir, "a.yaml")
	pathB := filepath.Join(tmpDir, "b.yaml")

	content := `slug: duplicate
profiles:
  default:
    slug: default
`
	if err := os.WriteFile(pathA, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile a.yaml failed: %v", err)
	}
	if err := os.WriteFile(pathB, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile b.yaml failed: %v", err)
	}

	specs, err := ParseRegistrySourceSpecs([]string{pathA, pathB})
	if err != nil {
		t.Fatalf("ParseRegistrySourceSpecs failed: %v", err)
	}
	_, err = NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err == nil {
		t.Fatalf("expected duplicate registry slug error")
	}
	if !strings.Contains(err.Error(), "duplicate registry slug") {
		t.Fatalf("expected duplicate registry slug error, got: %v", err)
	}
}

func TestChainedRegistry_ResolveDefaultUsesTopRegistryDefaultProfile(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	sharedPath := filepath.Join(tmpDir, "shared.yaml")
	privatePath := filepath.Join(tmpDir, "private.yaml")

	if err := os.WriteFile(sharedPath, []byte(`slug: shared
profiles:
  default:
    slug: default
    runtime:
      system_prompt: shared-default
`), 0o644); err != nil {
		t.Fatalf("WriteFile shared.yaml failed: %v", err)
	}

	// No profile named "default" here; decoder infers default_profile_slug from available profiles.
	if err := os.WriteFile(privatePath, []byte(`slug: private
profiles:
  assistant:
    slug: assistant
    runtime:
      system_prompt: private-assistant
`), 0o644); err != nil {
		t.Fatalf("WriteFile private.yaml failed: %v", err)
	}

	specs, err := ParseRegistrySourceSpecs([]string{sharedPath, privatePath})
	if err != nil {
		t.Fatalf("ParseRegistrySourceSpecs failed: %v", err)
	}
	chain, err := NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		t.Fatalf("NewChainedRegistryFromSourceSpecs failed: %v", err)
	}
	defer func() {
		_ = chain.Close()
	}()

	resolved, err := chain.ResolveEngineProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEngineProfile default failed: %v", err)
	}
	if got, want := resolved.RegistrySlug, MustRegistrySlug("private"); got != want {
		t.Fatalf("default registry mismatch: got=%q want=%q", got, want)
	}
	if got, want := resolved.EngineProfileSlug, MustEngineProfileSlug("assistant"); got != want {
		t.Fatalf("default profile mismatch: got=%q want=%q", got, want)
	}
	if got, want := resolved.EffectiveRuntime.SystemPrompt, "private-assistant"; got != want {
		t.Fatalf("default prompt mismatch: got=%q want=%q", got, want)
	}
}

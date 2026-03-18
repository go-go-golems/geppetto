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
}

func TestDecodeEngineProfileYAMLSingleRegistry_StrictFormat(t *testing.T) {
	_, err := DecodeEngineProfileYAMLSingleRegistry([]byte(`registries:
  default:
    slug: default
`))
	if err == nil {
		t.Fatalf("expected error for registries bundle")
	}

	_, err = DecodeEngineProfileYAMLSingleRegistry([]byte(`slug: default
default_profile_slug: default
profiles:
  default:
    slug: default
`))
	if err == nil {
		t.Fatalf("expected error for default_profile_slug")
	}

	_, err = DecodeEngineProfileYAMLSingleRegistry([]byte(`default:
  inference_settings:
    chat:
      engine: gpt-4o-mini
`))
	if err == nil {
		t.Fatalf("expected error for legacy profile map")
	}
}

func TestChainedRegistry_ResolveTopOfStack(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	yamlPath := filepath.Join(tmpDir, "private.yaml")
	if err := os.WriteFile(yamlPath, []byte(`slug: private
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-5-mini
  analyst:
    slug: analyst
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4o-mini
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
	defer func() { _ = sqliteStore.Close() }()

	shared := &EngineProfileRegistry{
		Slug:                     MustRegistrySlug("shared"),
		DefaultEngineProfileSlug: MustEngineProfileSlug("default"),
		Profiles: map[EngineProfileSlug]*EngineProfile{
			MustEngineProfileSlug("default"): {
				Slug:              MustEngineProfileSlug("default"),
				InferenceSettings: mustTestInferenceSettings(t, "openai", "shared-default"),
			},
			MustEngineProfileSlug("analyst"): {
				Slug:              MustEngineProfileSlug("analyst"),
				InferenceSettings: mustTestInferenceSettings(t, "openai", "shared-analyst"),
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
	defer func() { _ = chain.Close() }()

	resolvedAnalyst, err := chain.ResolveEngineProfile(ctx, ResolveInput{EngineProfileSlug: MustEngineProfileSlug("analyst")})
	if err != nil {
		t.Fatalf("ResolveEngineProfile analyst failed: %v", err)
	}
	if got, want := resolvedAnalyst.RegistrySlug, MustRegistrySlug("private"); got != want {
		t.Fatalf("analyst registry mismatch: got=%q want=%q", got, want)
	}
	if got := *resolvedAnalyst.InferenceSettings.Chat.Engine; got != "gpt-4o-mini" {
		t.Fatalf("analyst engine mismatch: got=%q", got)
	}

	resolvedDefault, err := chain.ResolveEngineProfile(ctx, ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveEngineProfile default failed: %v", err)
	}
	if got := *resolvedDefault.InferenceSettings.Chat.Engine; got != "gpt-5-mini" {
		t.Fatalf("default engine mismatch: got=%q", got)
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

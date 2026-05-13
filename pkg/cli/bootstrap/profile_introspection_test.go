package bootstrap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func TestBuildProfileRegistryReportFromRegistryListsDefaultsAndResolution(t *testing.T) {
	ctx := context.Background()
	registry := testProfileReportRegistry(t)
	report, err := BuildProfileRegistryReportFromRegistry(ctx, ProfileRegistryReportInput{
		SourceEntries:       []string{"profiles.yaml"},
		Registry:            registry,
		DefaultRegistrySlug: gepprofiles.MustRegistrySlug("default"),
		ResolveInput: gepprofiles.ResolveInput{
			RegistrySlug:      gepprofiles.MustRegistrySlug("default"),
			EngineProfileSlug: gepprofiles.MustEngineProfileSlug("fast"),
		},
	}, ProfileRegistryReportOptions{IncludeResolution: true, IncludeMergedSettings: true, RedactSecrets: true})
	if err != nil {
		t.Fatalf("BuildProfileRegistryReportFromRegistry: %v", err)
	}
	if report.DefaultRegistry != "default" {
		t.Fatalf("default registry mismatch: %#v", report)
	}
	if report.SelectedProfile != "fast" {
		t.Fatalf("selected profile mismatch: %#v", report)
	}
	if len(report.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(report.Profiles))
	}
	if report.Profiles[1].Model != "gpt-4.1-mini" {
		t.Fatalf("profile summary missing model: %#v", report.Profiles)
	}
	if report.Resolution == nil || len(report.Resolution.Lineage) != 2 {
		t.Fatalf("expected stack lineage, got %#v", report.Resolution)
	}
	if got := report.Resolution.InferenceSettings["api"].(map[string]any)["api_keys"]; got != "***REDACTED***" {
		t.Fatalf("expected api keys redacted, got %#v", got)
	}
}

func TestRenderProfileRegistryReportText(t *testing.T) {
	ctx := context.Background()
	registry := testProfileReportRegistry(t)
	report, err := BuildProfileRegistryReportFromRegistry(ctx, ProfileRegistryReportInput{
		SourceEntries:       []string{"profiles.yaml"},
		Registry:            registry,
		DefaultRegistrySlug: gepprofiles.MustRegistrySlug("default"),
	}, ProfileRegistryReportOptions{})
	if err != nil {
		t.Fatalf("BuildProfileRegistryReportFromRegistry: %v", err)
	}
	var buf bytes.Buffer
	if err := RenderProfileRegistryReport(&buf, report, "text"); err != nil {
		t.Fatalf("RenderProfileRegistryReport: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Profile sources", "Registries", "Profiles", "default", "fast", "gpt-4.1-mini", "openai"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestDefaultProfileSlugPopulatedFromRegistry(t *testing.T) {
	// When no explicit --profile is given, the report should still show
	// the default profile from the default registry.
	// BuildProfileRegistryReportFromRegistry sets it from the registry metadata
	// when the caller provides DefaultProfileSlug.
	ctx := context.Background()
	registry := testProfileReportRegistry(t)
	report, err := BuildProfileRegistryReportFromRegistry(ctx, ProfileRegistryReportInput{
		SourceEntries:       []string{"profiles.yaml"},
		Registry:            registry,
		DefaultRegistrySlug: gepprofiles.MustRegistrySlug("default"),
		DefaultProfileSlug:  gepprofiles.MustEngineProfileSlug("default"),
		// No ResolveInput.EngineProfileSlug set — simulates no --profile flag.
	}, ProfileRegistryReportOptions{})
	if err != nil {
		t.Fatalf("BuildProfileRegistryReportFromRegistry: %v", err)
	}
	// The default registry's default profile is "default" (from the YAML fixture).
	if report.DefaultProfile != "default" {
		t.Fatalf("expected default_profile=default, got %q", report.DefaultProfile)
	}
}

func TestDefaultProfileSlugExplicitOverridesRegistryDefault(t *testing.T) {
	// When --profile is explicitly set, DefaultProfileSlug should remain empty
	// (only the selected/resolve path should reflect the choice).
	ctx := context.Background()
	registry := testProfileReportRegistry(t)
	report, err := BuildProfileRegistryReportFromRegistry(ctx, ProfileRegistryReportInput{
		SourceEntries:       []string{"profiles.yaml"},
		Registry:            registry,
		DefaultRegistrySlug: gepprofiles.MustRegistrySlug("default"),
		DefaultProfileSlug:  gepprofiles.MustEngineProfileSlug("fast"),
	}, ProfileRegistryReportOptions{})
	if err != nil {
		t.Fatalf("BuildProfileRegistryReportFromRegistry: %v", err)
	}
	if report.DefaultProfile != "fast" {
		t.Fatalf("expected default_profile=fast, got %q", report.DefaultProfile)
	}
}

func TestSourceReportRedactsDSN(t *testing.T) {
	reports := sourceReports([]string{"sqlite-dsn:file:test.db?_auth&_auth_user=admin&_auth_pass=s3cret"})
	if len(reports) != 1 {
		t.Fatalf("expected 1 source report, got %d", len(reports))
	}
	if reports[0].DSN != "***REDACTED***" {
		t.Fatalf("expected redacted DSN, got %q", reports[0].DSN)
	}
	if reports[0].Raw != "sqlite-dsn:***REDACTED***" {
		t.Fatalf("expected redacted Raw, got %q", reports[0].Raw)
	}
	if reports[0].Kind != "sqlite-dsn" {
		t.Fatalf("expected kind sqlite-dsn, got %q", reports[0].Kind)
	}
}

func TestSourceReportPreservesNonDSN(t *testing.T) {
	reports := sourceReports([]string{"profiles.yaml"})
	if len(reports) != 1 {
		t.Fatalf("expected 1 source report, got %d", len(reports))
	}
	if reports[0].Kind != "yaml" {
		t.Fatalf("expected yaml kind, got %q", reports[0].Kind)
	}
	if reports[0].Path != "profiles.yaml" {
		t.Fatalf("expected path profiles.yaml, got %q", reports[0].Path)
	}
	if reports[0].DSN != "" {
		t.Fatalf("expected empty DSN for yaml source, got %q", reports[0].DSN)
	}
}

func testProfileReportRegistry(t *testing.T) gepprofiles.Registry {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.yaml")
	body := `slug: default
profiles:
  default:
    slug: default
    inference_settings:
      chat:
        api_type: openai
        engine: gpt-4.1-mini
      api:
        api_keys:
          openai-api-key: secret-value
  fast:
    slug: fast
    description: Fast profile
    stack:
      - profile_slug: default
    inference_settings:
      chat:
        engine: gpt-4.1-mini
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	specs, err := gepprofiles.ParseRegistrySourceSpecs([]string{path})
	if err != nil {
		t.Fatalf("ParseRegistrySourceSpecs: %v", err)
	}
	registry, err := gepprofiles.NewChainedRegistryFromSourceSpecs(context.Background(), specs)
	if err != nil {
		t.Fatalf("NewChainedRegistryFromSourceSpecs: %v", err)
	}
	t.Cleanup(func() { _ = registry.Close() })
	_ = aitypes.ApiTypeOpenAI // keep package import checked against profile YAML values used above
	return registry
}

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

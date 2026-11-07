package sections

import (
	"testing"

	decisionconfig "github.com/go-go-golems/geppetto/pkg/decision/config"
	embeddingsconfig "github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

// TestCreateGeppettoSectionsExposesCohereEmbeddingAPIKey verifies the
// advertised --embeddings-type cohere CLI choice has a credential flag in a
// registered section. Fields ending in -api-key are decoded into
// APISettings.APIKeys by the existing wildcard tag and embeddings slug loop.
func TestCreateGeppettoSectionsExposesCohereEmbeddingAPIKey(t *testing.T) {
	sections, err := CreateGeppettoSections()
	if err != nil {
		t.Fatalf("CreateGeppettoSections: %v", err)
	}

	for _, section := range sections {
		if section.GetSlug() != embeddingsconfig.EmbeddingsSlug {
			continue
		}
		if _, ok := section.GetDefinitions().Get("cohere-api-key"); !ok {
			t.Fatal("embeddings section does not expose cohere-api-key")
		}
		return
	}

	t.Fatal("CreateGeppettoSections did not register the embeddings section")
}

// The decision section is registered with its type, engine and System One
// credential flags.
func TestCreateGeppettoSectionsRegistersDecision(t *testing.T) {
	sections, err := CreateGeppettoSections()
	if err != nil {
		t.Fatalf("CreateGeppettoSections: %v", err)
	}
	for _, section := range sections {
		if section.GetSlug() != decisionconfig.DecisionSlug {
			continue
		}
		for _, name := range []string{"decision-type", "decision-engine", "systemone-api-key", "systemone-base-url"} {
			if _, ok := section.GetDefinitions().Get(name); !ok {
				t.Fatalf("decision section does not expose %s", name)
			}
		}
		return
	}
	t.Fatal("CreateGeppettoSections did not register the decision section")
}

func parseSettings(t *testing.T, m map[string]map[string]interface{}, without string) *settings.InferenceSettings {
	t.Helper()
	all, err := CreateGeppettoSections()
	if err != nil {
		t.Fatal(err)
	}
	kept := all[:0]
	for _, s := range all {
		if s.GetSlug() != without {
			kept = append(kept, s)
		}
	}
	parsed := values.New()
	if err := sources.Execute(schema.NewSchema(schema.WithSections(kept...)), parsed,
		sources.FromMap(m), sources.FromDefaults(fields.WithSource(fields.SourceDefaults))); err != nil {
		t.Fatal(err)
	}
	s, err := settings.NewInferenceSettingsFromParsedValues(parsed)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Flags fill DecisionConfig and the System One key and base URL; with no
// type or engine, Decision stays nil; a host without the section still parses.
func TestDecisionFlagsDecodeIntoInferenceSettings(t *testing.T) {
	s := parseSettings(t, map[string]map[string]interface{}{
		decisionconfig.DecisionSlug: {"decision-type": "systemone", "decision-engine": "djev", "systemone-api-key": "k", "systemone-base-url": "https://gw.example/v1"},
	}, "")
	if s.Decision == nil || s.Decision.Type != "systemone" || s.Decision.Engine != "djev" || s.Decision.MaxRequestBytes != 8<<20 {
		t.Fatalf("decision: %#v", s.Decision)
	}
	if s.API.APIKeys["systemone-api-key"] != "k" || s.API.BaseUrls["systemone-base-url"] != "https://gw.example/v1" {
		t.Fatalf("api: %#v", s.API)
	}
	if s := parseSettings(t, nil, ""); s.Decision != nil {
		t.Fatalf("unconfigured decision should be nil: %#v", s.Decision)
	}
	if s := parseSettings(t, nil, decisionconfig.DecisionSlug); s.Decision != nil {
		t.Fatalf("host without the section: %#v", s.Decision)
	}
}

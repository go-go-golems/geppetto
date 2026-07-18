// Package main implements rerank-profile-smoke: a profile-backed reranker
// smoke test that resolves a rerank profile, constructs a llama.cpp rerank
// provider, and runs one rerank call against a real or local server.
//
// This is the runnable counterpart to the opt-in live test in
// pkg/rerank/llamacpp/live_test.go. Unlike a _test, it is a real CLI you can
// point at any running llama.cpp reranker endpoint and inspect the output with
// Glazed (e.g. --output json).
//
// Usage:
//
//	# Resolve a rerank profile from ~/.config/pinocchio/profiles.yaml:
//	rerank-profile-smoke run --profile bge-reranker-local \
//	  --query "How does TTC calculate a payroll adjustment?"
//
//	# Or overlay rerank flags onto a base profile:
//	rerank-profile-smoke run --base-profile llamacpp-base \
//	  --rerank-type llamacpp --rerank-engine qllama/bge-reranker-v2-m3:q4_k_m \
//	  --rerank-base-url http://127.0.0.1:18012
//
// Example output (JSON):
//
//	{
//	  "profile": "bge-reranker-local",
//	  "provider": "llama.cpp",
//	  "model": "qllama/bge-reranker-v2-m3:q4_k_m",
//	  "query": "How does TTC calculate a payroll adjustment?",
//	  "results": [
//	    {"rank":1,"document_id":"chunk-001","index":0,"score":3.70},
//	    {"rank":2,"document_id":"chunk-002","index":1,"score":-10.99}
//	  ],
//	  "usage": {"input_tokens":34,"total_tokens":34},
//	  "duration_ms": 42
//	}
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/rerank"
	rerankconfig "github.com/go-go-golems/geppetto/pkg/rerank/config"
	rerankfactory "github.com/go-go-golems/geppetto/pkg/rerank/factory"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	aistepssettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type rerankCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*rerankCommand)(nil)

type rerankSettings struct {
	BaseProfile    string   `glazed:"base-profile"`
	RerankType     string   `glazed:"rerank-type"`
	RerankEngine   string   `glazed:"rerank-engine"`
	RerankBaseURL  string   `glazed:"rerank-base-url"`
	Query          string   `glazed:"query"`
	Document       []string `glazed:"document"`
	TimeoutSeconds int      `glazed:"timeout-seconds"`
}

func newRerankCommand() (*rerankCommand, error) {
	profileSettingsSection, err := geppettosections.NewProfileSettingsSection(
		geppettosections.WithProfileRegistriesDefault(defaultPinocchioProfilesPath()),
	)
	if err != nil {
		return nil, err
	}

	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run one profile-backed rerank call against a llama.cpp reranker"),
		cmds.WithLong(`Run one rerank call using profile registry settings.

If --profile is set, the selected profile must already contain rerank settings
(inference_settings.rerank.type, .engine, and api.base_urls.rerank-base-url).

If --profile is empty, the command resolves --base-profile and overlays the
rerank flags onto that base profile. --document may be repeated; each value is
"id|text" (the pipe separates the caller-controlled document ID from its text).

Use Glazed output flags for machine-readable output, for example:
  rerank-profile-smoke run --output json --query "..." --document "a|text a"`),
		cmds.WithFlags(
			fields.New("base-profile", fields.TypeString,
				fields.WithDefault("llamacpp-base"),
				fields.WithHelp("Base profile to stack when --profile is empty"),
			),
			fields.New("rerank-type", fields.TypeString,
				fields.WithDefault("llamacpp"),
				fields.WithHelp("Rerank provider type (only llamacpp supported)"),
			),
			fields.New("rerank-engine", fields.TypeString,
				fields.WithDefault("qllama/bge-reranker-v2-m3:q4_k_m"),
				fields.WithHelp("Rerank model/engine"),
			),
			fields.New("rerank-base-url", fields.TypeString,
				fields.WithDefault("http://127.0.0.1:18012"),
				fields.WithHelp("llama.cpp reranker base URL (set rerank-base-url in profile api.base_urls instead for profile use)"),
			),
			fields.New("query", fields.TypeString,
				fields.WithDefault("How does TTC calculate a payroll adjustment?"),
				fields.WithHelp("Query text to rerank documents against"),
			),
			fields.New("document", fields.TypeStringList,
				fields.WithHelp("Document as 'id|text'; repeat for multiple documents"),
			),
			fields.New("timeout-seconds", fields.TypeInteger,
				fields.WithDefault(30),
				fields.WithHelp("Rerank request timeout in seconds"),
			),
		),
		cmds.WithSections(profileSettingsSection),
	)

	return &rerankCommand{CommandDescription: description}, nil
}

func (c *rerankCommand) RunIntoGlazeProcessor(ctx context.Context, parsedValues *values.Values, gp middlewares.Processor) error {
	s := &rerankSettings{}
	if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return errors.Wrap(err, "decode rerank settings")
	}
	profileSettings := &geppettosections.ProfileSettings{}
	if err := parsedValues.DecodeSectionInto(geppettosections.ProfileSettingsSectionSlug, profileSettings); err != nil {
		return errors.Wrap(err, "decode profile settings")
	}

	timeout := time.Duration(s.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resolved, effectiveProfile, err := resolveSettings(ctx, profileSettings.ProfileRegistries, profileSettings.Profile, s)
	if err != nil {
		return err
	}

	if err := rerankfactory.ValidateInferenceSettingsForRerank(resolved); err != nil {
		return err
	}

	factory, err := rerankfactory.NewSettingsFactoryFromInferenceSettings(resolved)
	if err != nil {
		return err
	}
	provider, err := factory.NewProvider()
	if err != nil {
		return err
	}

	docs := defaultDocuments()
	if len(s.Document) > 0 {
		docs, err = parseDocuments(s.Document)
		if err != nil {
			return err
		}
	}

	resp, err := provider.Rerank(ctx, rerank.Request{
		Query:     s.Query,
		Documents: docs,
		TopN:      len(docs),
	})
	if err != nil {
		return err
	}

	model := provider.Model()
	row := types.NewRow(
		types.MRP("profile", effectiveProfile),
		types.MRP("provider", model.Provider),
		types.MRP("model", model.Name),
		types.MRP("query", s.Query),
		types.MRP("document_count", len(docs)),
		types.MRP("results", resultsToRows(resp.Results)),
	)
	if resp.Usage != nil {
		row.Set("usage_input_tokens", resp.Usage.InputTokens)
		row.Set("usage_total_tokens", resp.Usage.TotalTokens)
	}
	if resp.DurationMs != nil {
		row.Set("duration_ms", *resp.DurationMs)
	}
	return gp.AddRow(ctx, row)
}

func resolveSettings(ctx context.Context, registryEntries []string, profile string, s *rerankSettings) (*aistepssettings.InferenceSettings, string, error) {
	specs, err := profiles.ParseRegistrySourceSpecs(registryEntries)
	if err != nil {
		return nil, "", err
	}
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return nil, "", err
	}

	if strings.TrimSpace(profile) != "" {
		profileSlug, err := profiles.ParseEngineProfileSlug(profile)
		if err != nil {
			_ = chain.Close()
			return nil, "", err
		}
		resolved, err := chain.ResolveEngineProfile(ctx, profiles.ResolveInput{EngineProfileSlug: profileSlug})
		if err != nil {
			_ = chain.Close()
			return nil, "", err
		}
		return resolved.InferenceSettings, profile, nil
	}

	baseProfile := s.BaseProfile
	baseSlug, err := profiles.ParseEngineProfileSlug(baseProfile)
	if err != nil {
		_ = chain.Close()
		return nil, "", err
	}
	baseResolved, err := chain.ResolveEngineProfile(ctx, profiles.ResolveInput{EngineProfileSlug: baseSlug})
	if err != nil {
		_ = chain.Close()
		return nil, "", err
	}

	overlay := &aistepssettings.InferenceSettings{
		API: &aistepssettings.APISettings{
			BaseUrls:           map[string]string{"rerank-base-url": s.RerankBaseURL},
			AllowHTTP:          map[string]bool{"rerank": true},
			AllowLocalNetworks: map[string]bool{"rerank": true},
		},
		Rerank: &rerankconfig.RerankConfig{
			Type:   s.RerankType,
			Engine: s.RerankEngine,
		},
	}
	merged, err := profiles.MergeInferenceSettings(baseResolved.InferenceSettings, overlay)
	if err != nil {
		_ = chain.Close()
		return nil, "", err
	}
	return merged, fmt.Sprintf("%s + rerank(%s/%s)", baseProfile, s.RerankType, s.RerankEngine), nil
}

func defaultDocuments() []rerank.Document {
	return []rerank.Document{
		{ID: "chunk-001", Text: "A payroll adjustment corrects wages or deductions."},
		{ID: "chunk-002", Text: "Cypress trees tolerate dry conditions."},
		{ID: "chunk-003", Text: "Weather forecasts predict rain tomorrow."},
	}
}

func parseDocuments(raw []string) ([]rerank.Document, error) {
	docs := make([]rerank.Document, 0, len(raw))
	for i, d := range raw {
		idx := strings.Index(d, "|")
		if idx < 0 {
			return nil, fmt.Errorf("document %d %q must be 'id|text'", i, d)
		}
		id := strings.TrimSpace(d[:idx])
		text := strings.TrimSpace(d[idx+1:])
		if id == "" || text == "" {
			return nil, fmt.Errorf("document %d %q requires non-empty id and text", i, d)
		}
		docs = append(docs, rerank.Document{ID: id, Text: text})
	}
	return docs, nil
}

func resultsToRows(results []rerank.Result) []types.Row {
	rows := make([]types.Row, 0, len(results))
	for _, r := range results {
		rows = append(rows, types.NewRow(
			types.MRP("rank", r.Rank),
			types.MRP("document_id", r.DocumentID),
			types.MRP("index", r.Index),
			types.MRP("score", r.Score),
		))
	}
	return rows
}

func defaultPinocchioProfilesPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(".config", "pinocchio", "profiles.yaml")
	}
	return filepath.Join(home, ".config", "pinocchio", "profiles.yaml")
}

func main() {
	root := examplecmd.NewRoot("rerank-profile-smoke", "Profile-backed reranker smoke test")
	cmd, err := newRerankCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
}

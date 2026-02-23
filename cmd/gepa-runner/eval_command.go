package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	gepaopt "github.com/go-go-golems/geppetto/pkg/optimizer/gepa"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type EvalCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*EvalCommand)(nil)

type EvalSettings struct {
	ScriptPath  string `glazed:"script"`
	DatasetPath string `glazed:"dataset"`
	Prompt      string `glazed:"prompt"`
	PromptFile  string `glazed:"prompt-file"`
	OutReport   string `glazed:"out-report"`
	Debug       bool   `glazed:"debug"`
}

func NewEvalCommand() (*EvalCommand, error) {
	geppettoSections, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}

	description := cmds.NewCommandDescription(
		"eval",
		cmds.WithShort("Evaluate a prompt/candidate on a dataset using a JS evaluator plugin"),
		cmds.WithFlags(
			fields.New("script", fields.TypeString, fields.WithHelp("Path to JS optimizer plugin (descriptor)"), fields.WithRequired(true)),
			fields.New("dataset", fields.TypeString, fields.WithHelp("Path to dataset (.json or .jsonl). Optional if plugin provides dataset().")),
			fields.New("prompt", fields.TypeString, fields.WithHelp("Prompt text (overrides --prompt-file)")),
			fields.New("prompt-file", fields.TypeString, fields.WithHelp("Path to prompt file")),
			fields.New("out-report", fields.TypeString, fields.WithHelp("Write JSON eval report to this file (optional)")),
			fields.New("debug", fields.TypeBool, fields.WithHelp("Debug mode - show parsed layers"), fields.WithDefault(false)),
		),
		cmds.WithSections(geppettoSections...),
	)

	return &EvalCommand{CommandDescription: description}, nil
}

func (c *EvalCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	s := &EvalSettings{}
	if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}

	if s.Debug {
		b, err := yaml.Marshal(parsedValues)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, "=== Parsed Layers Debug ===")
		fmt.Fprintln(w, string(b))
		fmt.Fprintln(w, "==========================")
		return nil
	}

	if strings.TrimSpace(s.ScriptPath) == "" {
		return fmt.Errorf("--script is required")
	}

	promptText, err := resolveSeedText(s.Prompt, s.PromptFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(promptText) == "" {
		return fmt.Errorf("prompt is empty (use --prompt or --prompt-file)")
	}

	profile, err := resolvePinocchioProfile(parsedValues)
	if err != nil {
		return errors.Wrap(err, "failed to resolve pinocchio profile")
	}
	if err := applyProfileEnvironment(profile, parsedValues); err != nil {
		return errors.Wrap(err, "failed to apply profile environment")
	}
	engineOptions, err := resolveEngineOptions(parsedValues)
	if err != nil {
		return errors.Wrap(err, "failed to resolve engine options from parsed settings")
	}

	absScript, err := filepath.Abs(s.ScriptPath)
	if err != nil {
		return err
	}
	scriptRoot := filepath.Dir(absScript)

	jsrt, err := newJSRuntime(scriptRoot)
	if err != nil {
		return err
	}
	defer jsrt.Close()

	hostContext := map[string]any{
		"app":           "gepa-runner",
		"scriptPath":    filepath.ToSlash(absScript),
		"scriptRoot":    filepath.ToSlash(scriptRoot),
		"profile":       profile,
		"engineOptions": engineOptions,
	}
	plugin, meta, err := loadOptimizerPlugin(jsrt, absScript, hostContext)
	if err != nil {
		return err
	}
	log.Info().Str("plugin_id", meta.ID).Str("plugin_name", meta.Name).Msg("Loaded evaluator plugin")

	var examples []any
	if strings.TrimSpace(s.DatasetPath) != "" {
		examples, err = loadDataset(s.DatasetPath)
		if err != nil {
			return err
		}
	} else {
		examples, err = plugin.Dataset()
		if err != nil {
			return err
		}
	}
	if len(examples) == 0 {
		return fmt.Errorf("dataset is empty")
	}

	evals := make([]gepaopt.ExampleEval, 0, len(examples))
	for i, ex := range examples {
		r, err := plugin.Evaluate(gepaopt.Candidate{"prompt": promptText}, i, ex, pluginEvaluateOptions{
			Profile:       profile,
			EngineOptions: engineOptions,
		})
		if err != nil {
			return err
		}
		if len(r.Objectives) == 0 {
			r.Objectives = gepaopt.ObjectiveScores{"score": r.Score}
		}
		evals = append(evals, gepaopt.ExampleEval{ExampleIndex: i, Result: r})
	}

	stats := gepaopt.AggregateStats(evals)

	fmt.Fprintf(w, "Plugin: %s (%s)\n", meta.Name, meta.ID)
	fmt.Fprintf(w, "Dataset: %d examples\n", len(examples))
	fmt.Fprintf(w, "Mean score: %.6f\n", stats.MeanScore)
	if len(stats.MeanObjectives) > 0 {
		b, _ := json.MarshalIndent(stats.MeanObjectives, "", "  ")
		fmt.Fprintf(w, "Mean objectives:\n%s\n", string(b))
	}

	if strings.TrimSpace(s.OutReport) != "" {
		report := map[string]any{
			"plugin": map[string]any{
				"id":         meta.ID,
				"name":       meta.Name,
				"apiVersion": meta.APIVersion,
				"kind":       meta.Kind,
			},
			"stats": stats,
		}
		blob, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(s.OutReport, blob, 0o644); err != nil {
			return errors.Wrap(err, "failed to write out report")
		}
		fmt.Fprintf(w, "Wrote report to: %s\n", s.OutReport)
	}

	return nil
}

// Ensure the ProfileSettings section is linked.
var _ = cli.ProfileSettingsSlug

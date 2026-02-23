package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	gepaopt "github.com/go-go-golems/geppetto/pkg/optimizer/gepa"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var rootCmd = &cobra.Command{
	Use:   "gepa-runner",
	Short: "GEPA-style prompt optimization on top of Geppetto + JS evaluators",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

type OptimizeCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*OptimizeCommand)(nil)

type OptimizeSettings struct {
	ScriptPath       string `glazed:"script"`
	DatasetPath      string `glazed:"dataset"`
	Seed             string `glazed:"seed"`
	SeedFile         string `glazed:"seed-file"`
	Objective        string `glazed:"objective"`
	MaxEvalCalls     int    `glazed:"max-evals"`
	BatchSize        int    `glazed:"batch-size"`
	MaxSideInfoChars int    `glazed:"max-side-info-chars"`
	OutPrompt        string `glazed:"out-prompt"`
	OutReport        string `glazed:"out-report"`
	Debug            bool   `glazed:"debug"`
}

func NewOptimizeCommand() (*OptimizeCommand, error) {
	geppettoSections, err := geppettosections.CreateGeppettoSections()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create geppetto parameter layer")
	}

	description := cmds.NewCommandDescription(
		"optimize",
		cmds.WithShort("Run a GEPA-style reflective prompt evolution loop"),
		cmds.WithFlags(
			fields.New("script", fields.TypeString, fields.WithHelp("Path to JS optimizer plugin (descriptor)"), fields.WithRequired(true)),
			fields.New("dataset", fields.TypeString, fields.WithHelp("Path to dataset (.json or .jsonl). Optional if plugin provides dataset().")),
			fields.New("seed", fields.TypeString, fields.WithHelp("Seed prompt text (overrides --seed-file)")),
			fields.New("seed-file", fields.TypeString, fields.WithHelp("Path to seed prompt file")),
			fields.New("objective", fields.TypeString, fields.WithHelp("Natural-language optimization objective (used in reflection prompt)")),
			fields.New("max-evals", fields.TypeInteger, fields.WithHelp("Max evaluator calls (each example eval counts as 1)"), fields.WithDefault(200)),
			fields.New("batch-size", fields.TypeInteger, fields.WithHelp("Minibatch size per iteration"), fields.WithDefault(8)),
			fields.New("max-side-info-chars", fields.TypeInteger, fields.WithHelp("Cap formatted side-info chars passed to reflection LLM (0 = uncapped)"), fields.WithDefault(8000)),
			fields.New("out-prompt", fields.TypeString, fields.WithHelp("Write best prompt to this file (optional)")),
			fields.New("out-report", fields.TypeString, fields.WithHelp("Write JSON optimization report to this file (optional)")),
			fields.New("debug", fields.TypeBool, fields.WithHelp("Debug mode - show parsed layers"), fields.WithDefault(false)),
		),
		cmds.WithSections(geppettoSections...),
	)

	return &OptimizeCommand{CommandDescription: description}, nil
}

func (c *OptimizeCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	s := &OptimizeSettings{}
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

	seedText, err := resolveSeedText(s.Seed, s.SeedFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(seedText) == "" {
		return fmt.Errorf("seed prompt is empty (use --seed or --seed-file)")
	}

	// Ensure JS-side engine creation resolves the same profile by default.
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

	// Reflection LLM engine (Go side).
	engine, err := factory.NewEngineFromParsedValues(parsedValues)
	if err != nil {
		return errors.Wrap(err, "failed to create reflection engine from parsed values")
	}

	absScript, err := filepath.Abs(s.ScriptPath)
	if err != nil {
		return err
	}
	scriptRoot := filepath.Dir(absScript)

	// Load JS plugin.
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
	log.Info().Str("plugin_id", meta.ID).Str("plugin_name", meta.Name).Msg("Loaded optimizer plugin")

	// Load dataset.
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

	evalFn := func(ctx context.Context, cand gepaopt.Candidate, exampleIndex int, example any) (gepaopt.EvalResult, error) {
		return plugin.Evaluate(cand, exampleIndex, example, pluginEvaluateOptions{
			Profile:       profile,
			EngineOptions: engineOptions,
		})
	}

	cfg := gepaopt.Config{
		MaxEvalCalls:     s.MaxEvalCalls,
		BatchSize:        s.BatchSize,
		Objective:        s.Objective,
		MaxSideInfoChars: s.MaxSideInfoChars,
	}

	reflector := &gepaopt.Reflector{
		Engine:    engine,
		Objective: cfg.Objective,
	}

	opt := gepaopt.NewOptimizer(cfg, evalFn, reflector)

	res, err := opt.Optimize(ctx, gepaopt.Candidate{"prompt": seedText}, examples)
	if err != nil {
		return err
	}

	// Output summary.
	fmt.Fprintf(w, "Plugin: %s (%s)\n", meta.Name, meta.ID)
	fmt.Fprintf(w, "Dataset: %d examples\n", len(examples))
	fmt.Fprintf(w, "Calls used: %d / %d\n", res.CallsUsed, s.MaxEvalCalls)
	fmt.Fprintf(w, "Best mean score (over cached evals): %.6f (n=%d)\n", res.BestStats.MeanScore, res.BestStats.N)

	bestPrompt := res.BestCandidate["prompt"]
	if strings.TrimSpace(s.OutPrompt) != "" {
		if err := os.WriteFile(s.OutPrompt, []byte(bestPrompt), 0o644); err != nil {
			return errors.Wrap(err, "failed to write out prompt")
		}
		fmt.Fprintf(w, "Wrote best prompt to: %s\n", s.OutPrompt)
	} else {
		fmt.Fprintln(w, "\n=== Best Prompt ===")
		fmt.Fprintln(w, bestPrompt)
	}

	if strings.TrimSpace(s.OutReport) != "" {
		blob, _ := json.MarshalIndent(res, "", "  ")
		if err := os.WriteFile(s.OutReport, blob, 0o644); err != nil {
			return errors.Wrap(err, "failed to write out report")
		}
		fmt.Fprintf(w, "Wrote report to: %s\n", s.OutReport)
	}

	return nil
}

func main() {
	err := clay.InitGlazed("pinocchio", rootCmd)
	cobra.CheckErr(err)

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	cobra.CheckErr(err)

	optCmd, err := NewOptimizeCommand()
	cobra.CheckErr(err)

	command, err := cli.BuildCobraCommand(optCmd,
		cli.WithCobraMiddlewaresFunc(geppettosections.GetCobraCommandGeppettoMiddlewares),
		cli.WithProfileSettingsSection(),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command)

	evalCmd, err := NewEvalCommand()
	cobra.CheckErr(err)
	command2, err := cli.BuildCobraCommand(evalCmd,
		cli.WithCobraMiddlewaresFunc(geppettosections.GetCobraCommandGeppettoMiddlewares),
		cli.WithProfileSettingsSection(),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(command2)

	cobra.CheckErr(rootCmd.Execute())
}

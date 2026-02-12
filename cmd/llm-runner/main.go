package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	clay "github.com/go-go-golems/clay/pkg"
	fixtures "github.com/go-go-golems/geppetto/pkg/inference/fixtures"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openaisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/serde"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// fileSink moved into fixtures package

// Glazed BareCommand: run
type RunSettings struct {
	In          string `glazed:"in"`
	Out         string `glazed:"out"`
	Cassette    string `glazed:"cassette"`
	Record      bool   `glazed:"record"`
	Model       string `glazed:"model"`
	Stream      bool   `glazed:"stream"`
	EchoEvents  bool   `glazed:"echo-events"`
	Second      bool   `glazed:"second"`
	SecondUser  string `glazed:"second-user"`
	Raw         bool   `glazed:"raw"`
	CaptureLogs bool   `glazed:"capture-logs"`
}

type RunCommand struct{ *cmds.CommandDescription }

var _ cmds.BareCommand = (*RunCommand)(nil)

func NewRunCommand() (*RunCommand, error) {
	desc := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run LLM turns (OpenAI Responses) and record artifacts"),
		cmds.WithFlags(
			fields.New("in", fields.TypeString, fields.WithHelp("Input turn YAML path")),
			fields.New("out", fields.TypeString, fields.WithDefault("out"), fields.WithHelp("Output directory")),
			fields.New("cassette", fields.TypeString, fields.WithDefault(""), fields.WithHelp("VCR cassette base path (without .yaml)")),
			fields.New("record", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Record HTTP (otherwise replay)")),
			fields.New("model", fields.TypeString, fields.WithDefault("o4-mini"), fields.WithHelp("Model id for Responses")),
			fields.New("stream", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Use streaming")),
			fields.New("echo-events", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Echo NDJSON events to stdout while recording")),
			fields.New("second", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Run a second inference on the resulting turn")),
			fields.New("second-user", fields.TypeString, fields.WithDefault("Hello"), fields.WithHelp("User message to append before second run")),
			fields.New("raw", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Capture raw provider data under out/raw")),
			fields.New("capture-logs", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Capture logs to out/logs.jsonl")),
		),
	)
	return &RunCommand{CommandDescription: desc}, nil
}

func (c *RunCommand) Run(ctx context.Context, parsed *values.Values) error {
	// Setup logging from viper as in examples
	if err := logging.InitLoggerFromViper(); err != nil {
		return err
	}
	s := &RunSettings{}
	if err := parsed.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return err
	}
	if s.In == "" {
		return fmt.Errorf("--in is required")
	}
	if err := os.MkdirAll(s.Out, 0755); err != nil {
		return err
	}

	// Load fixture (wrapper) or raw turn YAML
	turn, followups, err := fixtures.LoadFixtureOrTurn(s.In)
	if err != nil {
		return err
	}
	if err := serde.SaveTurnYAML(filepath.Join(s.Out, "input_turn.yaml"), turn, serde.Options{}); err != nil {
		return err
	}

	// Engine settings
	apiKey := os.Getenv("OPENAI_API_KEY")
	st := &settings.StepSettings{
		API:    &settings.APISettings{APIKeys: map[string]string{"openai-api-key": apiKey}, BaseUrls: map[string]string{"openai-base-url": "https://api.openai.com/v1"}},
		Chat:   &settings.ChatSettings{Engine: &s.Model, Stream: s.Stream},
		OpenAI: &openaisettings.Settings{ReasoningEffort: strPtr("medium"), ReasoningSummary: strPtr("detailed")},
	}
	// Build follow-up steps: fixture-provided blocks, plus optional CLI-provided second user message
	steps := make([]turns.Block, 0, len(followups)+1)
	steps = append(steps, followups...)
	if s.Second {
		msg := s.SecondUser
		if strings.TrimSpace(msg) == "" {
			msg = "Hello"
		}
		steps = append(steps, turns.NewUserTextBlock(msg))
	}
	_, err = fixtures.ExecuteFixture(ctx, turn, steps, st, fixtures.ExecuteOptions{
		OutDir: s.Out, Cassette: s.Cassette, Record: s.Record, EchoEvents: s.EchoEvents, PrintTurns: true, RawCapture: s.Raw, CaptureLogs: s.CaptureLogs,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "RunInference failed:", err)
		return err
	}
	return nil
}

// fixture loading moved into fixtures package

// Glazed BareCommand: report
type ReportSettings struct {
	Out string `glazed:"out"`
}
type ReportCommand struct{ *cmds.CommandDescription }

var _ cmds.BareCommand = (*ReportCommand)(nil)

func NewReportCommand() (*ReportCommand, error) {
	desc := cmds.NewCommandDescription(
		"report",
		cmds.WithShort("Generate a Markdown report from artifacts in --out"),
		cmds.WithFlags(fields.New("out", fields.TypeString, fields.WithDefault("out"), fields.WithHelp("Artifacts directory"))),
	)
	return &ReportCommand{CommandDescription: desc}, nil
}

func (c *ReportCommand) Run(ctx context.Context, parsed *values.Values) error {
	if err := logging.InitLoggerFromViper(); err != nil {
		return err
	}
	s := &ReportSettings{}
	if err := parsed.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return err
	}
	outDir := s.Out
	reportPath, err := fixtures.BuildReport(outDir)
	if err != nil {
		return err
	}
	log.Info().Str("report", reportPath).Msg("report generated")
	fmt.Println("Report:", reportPath)
	return nil
}

func main() {
	rootCmd := &cobra.Command{Use: "llm-runner",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return logging.InitLoggerFromViper() },
	}
	// Init Viper and help
	if err := clay.InitGlazed("geppetto", rootCmd); err != nil {
		cobra.CheckErr(err)
	}
	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	runCmd, err := NewRunCommand()
	cobra.CheckErr(err)
	runCobra, err := cli.BuildCobraCommand(runCmd)
	cobra.CheckErr(err)
	reportCmd, err := NewReportCommand()
	cobra.CheckErr(err)
	reportCobra, err := cli.BuildCobraCommand(reportCmd)
	cobra.CheckErr(err)
	serveCmd, err := NewServeCommand()
	cobra.CheckErr(err)
	serveCobra, err := cli.BuildCobraCommand(serveCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(runCobra, reportCobra, serveCobra)

	cobra.CheckErr(rootCmd.Execute())
}

func strPtr(s string) *string { return &s }

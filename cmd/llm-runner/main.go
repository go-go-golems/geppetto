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
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// fileSink moved into fixtures package

// Glazed BareCommand: run
type RunSettings struct {
	In          string `glazed.parameter:"in"`
	Out         string `glazed.parameter:"out"`
	Cassette    string `glazed.parameter:"cassette"`
	Record      bool   `glazed.parameter:"record"`
	Model       string `glazed.parameter:"model"`
	Stream      bool   `glazed.parameter:"stream"`
	EchoEvents  bool   `glazed.parameter:"echo-events"`
	Second      bool   `glazed.parameter:"second"`
	SecondUser  string `glazed.parameter:"second-user"`
	Raw         bool   `glazed.parameter:"raw"`
	CaptureLogs bool   `glazed.parameter:"capture-logs"`
}

type RunCommand struct{ *cmds.CommandDescription }

var _ cmds.BareCommand = (*RunCommand)(nil)

func NewRunCommand() (*RunCommand, error) {
	desc := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run LLM turns (OpenAI Responses) and record artifacts"),
		cmds.WithFlags(
			parameters.NewParameterDefinition("in", parameters.ParameterTypeString, parameters.WithHelp("Input turn YAML path")),
			parameters.NewParameterDefinition("out", parameters.ParameterTypeString, parameters.WithDefault("out"), parameters.WithHelp("Output directory")),
			parameters.NewParameterDefinition("cassette", parameters.ParameterTypeString, parameters.WithDefault(""), parameters.WithHelp("VCR cassette base path (without .yaml)")),
			parameters.NewParameterDefinition("record", parameters.ParameterTypeBool, parameters.WithDefault(false), parameters.WithHelp("Record HTTP (otherwise replay)")),
			parameters.NewParameterDefinition("model", parameters.ParameterTypeString, parameters.WithDefault("o4-mini"), parameters.WithHelp("Model id for Responses")),
			parameters.NewParameterDefinition("stream", parameters.ParameterTypeBool, parameters.WithDefault(true), parameters.WithHelp("Use streaming")),
			parameters.NewParameterDefinition("echo-events", parameters.ParameterTypeBool, parameters.WithDefault(false), parameters.WithHelp("Echo NDJSON events to stdout while recording")),
			parameters.NewParameterDefinition("second", parameters.ParameterTypeBool, parameters.WithDefault(false), parameters.WithHelp("Run a second inference on the resulting turn")),
			parameters.NewParameterDefinition("second-user", parameters.ParameterTypeString, parameters.WithDefault("Hello"), parameters.WithHelp("User message to append before second run")),
			parameters.NewParameterDefinition("raw", parameters.ParameterTypeBool, parameters.WithDefault(false), parameters.WithHelp("Capture raw provider data under out/raw")),
			parameters.NewParameterDefinition("capture-logs", parameters.ParameterTypeBool, parameters.WithDefault(true), parameters.WithHelp("Capture logs to out/logs.jsonl")),
		),
	)
	return &RunCommand{CommandDescription: desc}, nil
}

func (c *RunCommand) Run(ctx context.Context, parsed *layers.ParsedLayers) error {
	// Setup logging from viper as in examples
	if err := logging.InitLoggerFromViper(); err != nil {
		return err
	}
	s := &RunSettings{}
	if err := parsed.InitializeStruct(layers.DefaultSlug, s); err != nil {
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

	// Engine settings (chat completions for now; Responses wiring follows in PR07)
	apiKey := os.Getenv("OPENAI_API_KEY")
	st := &settings.StepSettings{
		API:    &settings.APISettings{APIKeys: map[string]string{"openai-api-key": apiKey}, BaseUrls: map[string]string{"openai-base-url": "https://api.openai.com/v1"}},
		Chat:   &settings.ChatSettings{Engine: &s.Model, Stream: s.Stream},
		OpenAI: &openaisettings.Settings{},
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
	Out string `glazed.parameter:"out"`
}
type ReportCommand struct{ *cmds.CommandDescription }

var _ cmds.BareCommand = (*ReportCommand)(nil)

func NewReportCommand() (*ReportCommand, error) {
	desc := cmds.NewCommandDescription(
		"report",
		cmds.WithShort("Generate a Markdown report from artifacts in --out"),
		cmds.WithFlags(parameters.NewParameterDefinition("out", parameters.ParameterTypeString, parameters.WithDefault("out"), parameters.WithHelp("Artifacts directory"))),
	)
	return &ReportCommand{CommandDescription: desc}, nil
}

func (c *ReportCommand) Run(ctx context.Context, parsed *layers.ParsedLayers) error {
	if err := logging.InitLoggerFromViper(); err != nil {
		return err
	}
	s := &ReportSettings{}
	if err := parsed.InitializeStruct(layers.DefaultSlug, s); err != nil {
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
	if err := clay.InitViper("geppetto", rootCmd); err != nil {
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

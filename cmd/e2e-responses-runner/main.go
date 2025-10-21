package main

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/dnaeon/go-vcr/recorder"
    clay "github.com/go-go-golems/clay/pkg"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
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

// fileSink writes events as NDJSON into a file; optionally echoes to stdout
type fileSink struct{
    f *os.File
    echo bool
}

func (s *fileSink) PublishEvent(e events.Event) error {
    t := string(e.Type())
    b, err := json.Marshal(map[string]any{
        "type": t,
        "event": e,
        "ts": time.Now().UnixMilli(),
    })
    if err != nil { return err }
	line := append(b, '\n')
	if _, err := s.f.Write(line); err != nil { return err }
	if s.echo {
		_, _ = os.Stdout.Write(line)
	}
    return nil
}

// Glazed BareCommand: run
type RunSettings struct {
    In        string `glazed.parameter:"in"`
    Out       string `glazed.parameter:"out"`
    Cassette  string `glazed.parameter:"cassette"`
    Record    bool   `glazed.parameter:"record"`
    Model     string `glazed.parameter:"model"`
    Stream    bool   `glazed.parameter:"stream"`
    EchoEvents bool  `glazed.parameter:"echo-events"`
}

type RunCommand struct{ *cmds.CommandDescription }

var _ cmds.BareCommand = (*RunCommand)(nil)

func NewRunCommand() (*RunCommand, error) {
    desc := cmds.NewCommandDescription(
        "run",
        cmds.WithShort("Run a turn against OpenAI Responses and record artifacts"),
        cmds.WithFlags(
            parameters.NewParameterDefinition("in", parameters.ParameterTypeString, parameters.WithHelp("Input turn YAML path")),
            parameters.NewParameterDefinition("out", parameters.ParameterTypeString, parameters.WithDefault("out"), parameters.WithHelp("Output directory")),
            parameters.NewParameterDefinition("cassette", parameters.ParameterTypeString, parameters.WithDefault(""), parameters.WithHelp("VCR cassette base path (without .yaml)")),
            parameters.NewParameterDefinition("record", parameters.ParameterTypeBool, parameters.WithDefault(false), parameters.WithHelp("Record HTTP (otherwise replay)")),
            parameters.NewParameterDefinition("model", parameters.ParameterTypeString, parameters.WithDefault("o4-mini"), parameters.WithHelp("Model id for Responses")),
            parameters.NewParameterDefinition("stream", parameters.ParameterTypeBool, parameters.WithDefault(true), parameters.WithHelp("Use streaming")),
            parameters.NewParameterDefinition("echo-events", parameters.ParameterTypeBool, parameters.WithDefault(false), parameters.WithHelp("Echo NDJSON events to stdout while recording")),
        ),
    )
    return &RunCommand{CommandDescription: desc}, nil
}

func (c *RunCommand) Run(ctx context.Context, parsed *layers.ParsedLayers) error {
    // Setup logging from viper as in examples
    if err := logging.InitLoggerFromViper(); err != nil { return err }
    s := &RunSettings{}
    if err := parsed.InitializeStruct(layers.DefaultSlug, s); err != nil { return err }
    if s.In == "" { return fmt.Errorf("--in is required") }
    if err := os.MkdirAll(s.Out, 0755); err != nil { return err }

    // Load and persist input
    turn, err := serde.LoadTurnYAML(s.In)
    if err != nil { return err }
    if err := serde.SaveTurnYAML(filepath.Join(s.Out, "input_turn.yaml"), turn, serde.Options{}); err != nil { return err }

    // Optional VCR
    var rec *recorder.Recorder
    if s.Cassette != "" {
        mode := recorder.ModeReplaying
        if s.Record { mode = recorder.ModeRecording }
        rec, err = recorder.NewAsMode(s.Cassette, mode, nil)
        if err != nil { return err }
        defer rec.Stop()
        orig := http.DefaultTransport
        http.DefaultTransport = rec
        defer func(){ http.DefaultTransport = orig }()
    }

    // Event NDJSON
    ef, err := os.Create(filepath.Join(s.Out, "events.ndjson"))
    if err != nil { return err }
    defer ef.Close()
    sink := &fileSink{f: ef, echo: s.EchoEvents}

    // Engine settings
    apiKey := os.Getenv("OPENAI_API_KEY")
    st := &settings.StepSettings{
        API: &settings.APISettings{ APIKeys: map[string]string{"openai-api-key": apiKey}, BaseUrls: map[string]string{"openai-base-url": "https://api.openai.com/v1"} },
        Chat: &settings.ChatSettings{ Engine: &s.Model, Stream: s.Stream },
        OpenAI: &openaisettings.Settings{ ReasoningEffort: strPtr("medium"), ReasoningSummary: strPtr("detailed") },
    }
    eng, err := openai_responses.NewEngine(st, engine.WithSink(sink))
    if err != nil { return err }

    runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
    defer cancel()
    finalTurn, err := eng.RunInference(runCtx, turn)
    if err != nil {
        // Surface provider error bodies (e.g., 429) explicitly
        fmt.Fprintln(os.Stderr, "RunInference failed:", err)
        return err
    }

    // Print final turn
    turns.FprintfTurn(os.Stdout, finalTurn, turns.WithRoles(true), turns.WithToolDetail(true))

    // Save final turn
    if err := serde.SaveTurnYAML(filepath.Join(s.Out, "final_turn.yaml"), finalTurn, serde.Options{}); err != nil { return err }
    return nil
}

// Glazed BareCommand: report
type ReportSettings struct { Out string `glazed.parameter:"out"` }
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
    if err := logging.InitLoggerFromViper(); err != nil { return err }
    s := &ReportSettings{}
    if err := parsed.InitializeStruct(layers.DefaultSlug, s); err != nil { return err }
    outDir := s.Out
    reportPath := filepath.Join(outDir, "report.md")
    var inputYAML, finalYAML string
    if b, err := os.ReadFile(filepath.Join(outDir, "input_turn.yaml")); err == nil { inputYAML = string(b) }
    if b, err := os.ReadFile(filepath.Join(outDir, "final_turn.yaml")); err == nil { finalYAML = string(b) }

    // Read events
    eventsPath := filepath.Join(outDir, "events.ndjson")
    type recLine struct{ Type string `json:"type"`; Ts int64 `json:"ts"`; Event map[string]any `json:"event"` }
    var lines []recLine
    if f, err := os.Open(eventsPath); err == nil {
        defer f.Close()
        sc := bufio.NewScanner(f)
        for sc.Scan() {
            var rl recLine
            if err := json.Unmarshal([]byte(sc.Text()), &rl); err == nil { lines = append(lines, rl) }
        }
    }
    var model, finalText string
    for _, l := range lines {
        if em, ok := l.Event["meta"].(map[string]any); ok {
            if mid, ok2 := em["model"].(string); ok2 && model == "" { model = mid }
        }
        if l.Type == string(events.EventTypeFinal) {
            if txt, ok := l.Event["text"].(string); ok { finalText = txt }
        }
    }
    var b strings.Builder
    b.WriteString("# E2E Responses Report\n\n")
    if model != "" { b.WriteString(fmt.Sprintf("- Model: %s\n", model)) }
    b.WriteString(fmt.Sprintf("- Generated: %s\n\n", time.Now().Format(time.RFC3339)))
    b.WriteString("## Input Turn (YAML)\n\n")
    if inputYAML != "" { b.WriteString("```yaml\n" + inputYAML + "\n```\n\n") } else { b.WriteString("(missing)\n\n") }
    b.WriteString("## Final Turn (YAML)\n\n")
    if finalYAML != "" { b.WriteString("```yaml\n" + finalYAML + "\n```\n\n") } else { b.WriteString("(missing)\n\n") }
    b.WriteString("## Final Assistant Text\n\n")
    if finalText != "" { b.WriteString(finalText + "\n\n") } else { b.WriteString("(not found)\n\n") }
    b.WriteString("## Event Timeline\n\n")
    for _, l := range lines { b.WriteString(fmt.Sprintf("- %s @ %d\n", l.Type, l.Ts)) }
    if err := os.WriteFile(reportPath, []byte(b.String()), 0644); err != nil { return err }
    log.Info().Str("report", reportPath).Msg("report generated")
    fmt.Println("Report:", reportPath)
    return nil
}

func main() {
    rootCmd := &cobra.Command{ Use: "responses-runner",
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return logging.InitLoggerFromViper() },
    }
    // Init Viper and help
    if err := clay.InitViper("geppetto", rootCmd); err != nil { cobra.CheckErr(err) }
    helpSystem := help.NewHelpSystem(); help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

    runCmd, err := NewRunCommand(); cobra.CheckErr(err)
    runCobra, err := cli.BuildCobraCommand(runCmd); cobra.CheckErr(err)
    reportCmd, err := NewReportCommand(); cobra.CheckErr(err)
    reportCobra, err := cli.BuildCobraCommand(reportCmd); cobra.CheckErr(err)
    rootCmd.AddCommand(runCobra, reportCobra)

    cobra.CheckErr(rootCmd.Execute())
}

func strPtr(s string) *string { return &s }

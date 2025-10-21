package fixtures

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/dnaeon/go-vcr/recorder"
    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/go-go-golems/geppetto/pkg/inference/engine"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/go-go-golems/geppetto/pkg/turns/serde"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "gopkg.in/yaml.v3"
)

// Fixture format wrapper: top-level 'turn' and 'followups' with full blocks
type FixtureDoc struct {
    Version   int           `yaml:"version,omitempty"`
    Turn      *turns.Turn   `yaml:"turn,omitempty"`
    Followups []turns.Block `yaml:"followups,omitempty"`
}

// LoadFixtureOrTurn loads either a FixtureDoc (turn + followups) or a raw Turn from YAML.
func LoadFixtureOrTurn(path string) (*turns.Turn, []turns.Block, error) {
    b, err := os.ReadFile(path)
    if err != nil { return nil, nil, err }
    var fx FixtureDoc
    if err := yaml.Unmarshal(b, &fx); err == nil && (fx.Turn != nil || len(fx.Followups) > 0) {
        if fx.Turn == nil { t := &turns.Turn{}; return t, fx.Followups, nil }
        serde.NormalizeTurn(fx.Turn)
        return fx.Turn, fx.Followups, nil
    }
    // Fallback: raw Turn document
    t, err := serde.FromYAML(b)
    if err != nil { return nil, nil, err }
    return t, nil, nil
}

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

type ExecuteOptions struct {
    OutDir     string
    Cassette   string
    Record     bool
    EchoEvents bool
    // When true, print each resulting turn to stdout
    PrintTurns bool
    // When true, also capture raw provider data into out/raw
    RawCapture bool
    // When true, capture logs to out/logs.jsonl
    CaptureLogs bool
}

// ExecuteFixture runs the initial turn and all provided follow-up blocks, persisting artifacts.
// Artifacts:
// - input_turn.yaml
// - events.ndjson (first run)
// - final_turn.yaml (after first run)
// - For each follow-up i starting at 1:
//   - final_turn_i.yaml (turn after appending follow-up, before run)
//   - events-(i+1).ndjson (events for this run)
//   - final_turn_(i+1).yaml (turn after run)
func ExecuteFixture(ctx context.Context, turn *turns.Turn, followups []turns.Block, st *settings.StepSettings, opts ExecuteOptions) (*turns.Turn, error) {
    if opts.OutDir == "" { return nil, fmt.Errorf("out dir required") }
    if err := os.MkdirAll(opts.OutDir, 0755); err != nil { return nil, err }

    // Setup log capture if requested
    var logFile *os.File
    var origLogger zerolog.Logger
    if opts.CaptureLogs {
        lf, err := os.Create(filepath.Join(opts.OutDir, "logs.jsonl"))
        if err != nil { return nil, err }
        logFile = lf
        defer logFile.Close()
        // Setup multi-writer to both console and file
        origLogger = log.Logger
        multi := io.MultiWriter(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}, logFile)
        log.Logger = log.Logger.Output(multi)
        defer func() { log.Logger = origLogger }()
    }

    if err := serde.SaveTurnYAML(filepath.Join(opts.OutDir, "input_turn.yaml"), turn, serde.Options{}); err != nil { return nil, err }

    // Optional VCR
    var rec *recorder.Recorder
    if opts.Cassette != "" {
        mode := recorder.ModeReplaying
        if opts.Record { mode = recorder.ModeRecording }
        r, err := recorder.NewAsMode(opts.Cassette, mode, nil)
        if err != nil { return nil, err }
        rec = r
        defer rec.Stop()
        orig := http.DefaultTransport
        http.DefaultTransport = rec
        defer func(){ http.DefaultTransport = orig }()
    }

    // First run events
    ef, err := os.Create(filepath.Join(opts.OutDir, "events.ndjson"))
    if err != nil { return nil, err }
    defer ef.Close()
    sink := &fileSink{f: ef, echo: opts.EchoEvents}

    engOpts := []engine.Option{ engine.WithSink(sink) }
    eng, err := openai_responses.NewEngine(st, engOpts...)
    if err != nil { return nil, err }
    runCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
    defer cancel()
    if opts.RawCapture {
        turnID := ""
        if turn != nil { turnID = turn.ID }
        tap := NewDiskTap(opts.OutDir, 1, turnID)
        defer tap.Close()
        runCtx = engine.WithDebugTap(runCtx, tap)
    }
    finalTurn, err := eng.RunInference(runCtx, turn)
    if err != nil { return nil, err }
    if opts.PrintTurns {
        turns.FprintfTurn(os.Stdout, finalTurn, turns.WithRoles(true), turns.WithToolDetail(true))
    }
    if err := serde.SaveTurnYAML(filepath.Join(opts.OutDir, "final_turn.yaml"), finalTurn, serde.Options{}); err != nil { return nil, err }

    // Follow-ups
    for i, fb := range followups {
        stepIdx := i + 1
        turns.AppendBlock(finalTurn, fb)
        if err := serde.SaveTurnYAML(filepath.Join(opts.OutDir, fmt.Sprintf("final_turn_%d.yaml", stepIdx)), finalTurn, serde.Options{}); err != nil { return nil, err }

        ef2, err := os.Create(filepath.Join(opts.OutDir, fmt.Sprintf("events-%d.ndjson", stepIdx+1)))
        if err != nil { return nil, err }
        // Close after this iteration
        func() {
            defer ef2.Close()
            sink2 := &fileSink{f: ef2, echo: opts.EchoEvents}
            eng2, err := openai_responses.NewEngine(st, engine.WithSink(sink2))
            if err != nil { log.Error().Err(err).Msg("failed to create engine for follow-up"); return }
            runCtx2, cancel2 := context.WithTimeout(ctx, 60*time.Second)
            defer cancel2()
            if opts.RawCapture {
                turnID := ""
                if finalTurn != nil { turnID = finalTurn.ID }
                tap := NewDiskTap(opts.OutDir, stepIdx+1, turnID)
                defer tap.Close()
                runCtx2 = engine.WithDebugTap(runCtx2, tap)
            }
            finalTurn2, err := eng2.RunInference(runCtx2, finalTurn)
            if err != nil { log.Error().Err(err).Msg("follow-up RunInference failed"); return }
            if opts.PrintTurns {
                turns.FprintfTurn(os.Stdout, finalTurn2, turns.WithRoles(true), turns.WithToolDetail(true))
            }
            if err := serde.SaveTurnYAML(filepath.Join(opts.OutDir, fmt.Sprintf("final_turn_%d.yaml", stepIdx+1)), finalTurn2, serde.Options{}); err != nil {
                log.Error().Err(err).Msg("failed to save follow-up final turn")
                return
            }
            finalTurn = finalTurn2
        }()
    }
    return finalTurn, nil
}

// BuildReport generates a simple Markdown report from artifacts in opts.OutDir.
func BuildReport(outDir string) (string, error) {
    reportPath := filepath.Join(outDir, "report.md")
    var inputYAML, finalYAML string
    if b, err := os.ReadFile(filepath.Join(outDir, "input_turn.yaml")); err == nil { inputYAML = string(b) }
    if b, err := os.ReadFile(filepath.Join(outDir, "final_turn.yaml")); err == nil { finalYAML = string(b) }

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
    if err := os.WriteFile(reportPath, []byte(b.String()), 0644); err != nil { return "", err }
    return reportPath, nil
}



package agentmode

import (
    "context"
    "fmt"
    "strings"
    "time"

    rootmw "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/steps/parse"
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/rs/zerolog/log"
    "gopkg.in/yaml.v3"
)

// Data keys for agent mode; local to this middleware package
const (
    DataKeyAgentMode            = "agent_mode"
    DataKeyAgentModeAllowedTools = "agent_mode_allowed_tools"
)

// AgentMode describes a mode name with allowed tools and an optional system prompt snippet.
type AgentMode struct {
    Name         string
    AllowedTools []string
    Prompt       string
}

// Resolver resolves a mode name to its definition.
// Deprecated: Resolver and Store merged into Service in service.go
type Resolver interface{ GetMode(ctx context.Context, name string) (*AgentMode, error) }
type Store interface{ GetCurrentMode(ctx context.Context, runID string) (string, error); RecordModeChange(ctx context.Context, change ModeChange) error }

// ModeChange captures a mode transition with optional analysis text.
type ModeChange struct {
    RunID    string
    TurnID   string
    FromMode string
    ToMode   string
    Analysis string
    At       time.Time
}

// Config configures the behavior of the middleware.
type Config struct {
    DefaultMode               string
    InsertSystemPrompt        bool
    InsertSwitchInstructions  bool
}

func DefaultConfig() Config {
    return Config{
        DefaultMode:              "default",
        InsertSystemPrompt:       true,
        InsertSwitchInstructions: true,
    }
}

// NewMiddleware returns a middleware.Middleware compatible handler.
func NewMiddleware(svc Service, cfg Config) rootmw.Middleware {
    return func(next rootmw.HandlerFunc) rootmw.HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            if t == nil {
                return next(ctx, t)
            }
            if t.Data == nil {
                t.Data = map[string]any{}
            }

            // Determine current mode: from Turn.Data or Store fallback
            modeName, _ := t.Data[DataKeyAgentMode].(string)
            if modeName == "" && svc != nil && t.RunID != "" {
                if m, err := svc.GetCurrentMode(ctx, t.RunID); err == nil && m != "" {
                    modeName = m
                }
            }
            if modeName == "" {
                modeName = cfg.DefaultMode
                t.Data[DataKeyAgentMode] = modeName
            }

            mode, err := svc.GetMode(ctx, modeName)
            if err != nil {
                log.Warn().Str("requested_mode", modeName).Msg("agentmode: unknown mode; continuing without restrictions")
            } else {
                if cfg.InsertSystemPrompt && strings.TrimSpace(mode.Prompt) != "" {
                    turns.AppendBlock(t, turns.NewSystemTextBlock(mode.Prompt))
                }
                if cfg.InsertSwitchInstructions {
                    instr := BuildYamlModeSwitchInstructions(mode.Name)
                    turns.AppendBlock(t, turns.NewSystemTextBlock(instr))
                }
                // Pass allowed tools hint to downstream tool middleware
                if len(mode.AllowedTools) > 0 {
                    t.Data[DataKeyAgentModeAllowedTools] = append([]string(nil), mode.AllowedTools...)
                }
            }

            // Run next
            res, err := next(ctx, t)
            if err != nil {
                return res, err
            }

            // Parse assistant response to detect YAML mode switch
            newMode, analysis := DetectYamlModeSwitch(res)
            if newMode != "" && newMode != modeName {
                log.Debug().Str("from", modeName).Str("to", newMode).Msg("agentmode: detected mode switch via YAML")
                // Apply to turn for next call
                res.Data[DataKeyAgentMode] = newMode
                // Record change
                if svc != nil {
                    _ = svc.RecordModeChange(ctx, ModeChange{RunID: res.RunID, TurnID: res.ID, FromMode: modeName, ToMode: newMode, Analysis: analysis, At: time.Now()})
                }
                // Announce
                turns.AppendBlock(res, turns.NewSystemTextBlock(fmt.Sprintf("[agent-mode] switched to %s", newMode)))
            }
            return res, nil
        }
    }
}

// BuildYamlModeSwitchInstructions returns instructions for the model to propose a mode switch using YAML.
func BuildYamlModeSwitchInstructions(current string) string {
    var b strings.Builder
    b.WriteString("Please propose a mode switch by emitting the following YAML (no additional text around it):\n\n")
    b.WriteString("```yaml\n")
    b.WriteString("mode_switch:\n")
    b.WriteString("  analysis: |\n")
    b.WriteString("    Provide a long, detailed reasoning for why switching mode helps. Use multiple sentences.\n")
    b.WriteString("  new_mode: MODE_NAME\n")
    b.WriteString("```\n\n")
    b.WriteString("Current mode: ")
    b.WriteString(current)
    return b.String()
}

// DetectYamlModeSwitch scans assistant LLM text blocks for a YAML code fence containing mode_switch.
func DetectYamlModeSwitch(t *turns.Turn) (newMode string, analysis string) {
    if t == nil {
        return "", ""
    }
    for _, b := range t.Blocks {
        if b.Kind != turns.BlockKindLLMText {
            continue
        }
        txt, _ := b.Payload[turns.PayloadKeyText].(string)
        if txt == "" {
            continue
        }
        blocks, err := parse.ExtractYAMLBlocks(txt)
        if err != nil {
            continue
        }
        for _, body := range blocks {
            body = strings.TrimSpace(body)
            var data struct{
                ModeSwitch struct {
                    Analysis string `yaml:"analysis"`
                    NewMode  string `yaml:"new_mode"`
                } `yaml:"mode_switch"`
            }
            if err := yaml.Unmarshal([]byte(body), &data); err != nil {
                continue
            }
            if nm := strings.TrimSpace(data.ModeSwitch.NewMode); nm != "" {
                return nm, strings.TrimSpace(data.ModeSwitch.Analysis)
            }
        }
    }
    return "", ""
}



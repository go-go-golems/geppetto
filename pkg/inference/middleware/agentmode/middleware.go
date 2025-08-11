package agentmode

import (
    "context"
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/go-go-golems/geppetto/pkg/events"
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
                // Remove previously inserted AgentMode-related blocks
                _ = turns.RemoveBlocksByMetadata(t, "agentmode_tag",
                    "agentmode_system_prompt",
                    "agentmode_switch_instructions",
                    "agentmode_user_prompt",
                )

                // Build a single user block with mode prompt and (optionally) switch instructions
                var bldr strings.Builder
                if cfg.InsertSystemPrompt && strings.TrimSpace(mode.Prompt) != "" {
                    bldr.WriteString("<currentMode>")
                    bldr.WriteString(strings.TrimSpace(mode.Prompt))
                    bldr.WriteString("</currentMode>")
                }
                if cfg.InsertSwitchInstructions {
                    if bldr.Len() > 0 { bldr.WriteString("\n\n") }
                    bldr.WriteString(BuildYamlModeSwitchInstructions(mode.Name, listModeNames(svc)))
                }
                if bldr.Len() > 0 {
                    usr := turns.WithBlockMetadata(
                        turns.NewUserTextBlock(bldr.String()),
                        map[string]any{"agentmode_tag": "agentmode_user_prompt", "agentmode": mode.Name},
                    )
                    // Insert as second-to-last
                    turns.InsertBlockBeforeLast(t, usr)
                    // Log insertion
                    events.PublishEventToContext(ctx, events.NewLogEvent(
                        events.EventMetadata{}, nil, "info",
                        "agentmode: user prompt inserted",
                        map[string]any{"mode": mode.Name},
                    ))
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
                // Announce (append system message and emit Info event)
                turns.AppendBlock(res, turns.NewSystemTextBlock(fmt.Sprintf("[agent-mode] switched to %s", newMode)))
                events.PublishEventToContext(ctx, events.NewInfoEvent(
                    events.EventMetadata{}, nil,
                    "agentmode: mode switched",
                    map[string]any{
                        "from":     modeName,
                        "to":       newMode,
                        "analysis": analysis,
                    },
                ))
                // Also add a user-visible line to REPL via an Info event that UIs can append
                events.PublishEventToContext(ctx, events.NewInfoEvent(
                    events.EventMetadata{}, nil,
                    "Mode changed",
                    map[string]any{
                        "from": modeName,
                        "to":   newMode,
                    },
                ))
            }
            return res, nil
        }
    }
}

// BuildYamlModeSwitchInstructions returns instructions for the model to propose a mode switch using YAML.
func BuildYamlModeSwitchInstructions(current string, available []string) string {
    var b strings.Builder
    b.WriteString("<modeSwitchGuidelines>")
    b.WriteString("Analyze the current conversation and determine if a mode switch would be beneficial. ")
    b.WriteString("Consider the user's request, the context, and the available capabilities in different modes. ")
    b.WriteString("If a mode switch would improve your ability to help the user, propose it using the following YAML format. ")
    b.WriteString("If the current mode is appropriate, do not include the new_mode field.")
    b.WriteString("</modeSwitchGuidelines>\n\n")
    b.WriteString("```yaml\n")
    b.WriteString("mode_switch:\n")
    b.WriteString("  analysis: |\n")
    b.WriteString("    Provide a detailed analysis of the current situation. Explain what the user is trying to accomplish,\n")
    b.WriteString("    what capabilities are needed, and why the current mode may or may not be optimal.\n")
    b.WriteString("    If proposing a switch, explain the specific benefits the new mode would provide.\n")
    b.WriteString("    Use multiple sentences to thoroughly justify your reasoning.\n")
    b.WriteString("  new_mode: MODE_NAME  # Only include this if you recommend switching modes\n")
    b.WriteString("```\n\n")
    b.WriteString("Current mode: ")
    b.WriteString(current)
    if len(available) > 0 {
        b.WriteString("\nAvailable modes: ")
        b.WriteString(strings.Join(available, ", "))
    }
    b.WriteString("\n\nRemember: Only propose a mode switch if it would genuinely improve your ability to assist the user. ")
    b.WriteString("Staying in the current mode is often the right choice.")
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

// listModeNames extracts available mode names from the provided Service, if it is a known implementation.
func listModeNames(svc Service) []string {
    if svc == nil {
        return nil
    }
    // Support StaticService and SQLiteService which both embed a modes map keyed by lower-case name
    switch s := svc.(type) {
    case *StaticService:
        names := make([]string, 0, len(s.modes))
        for _, m := range s.modes {
            if m != nil && m.Name != "" {
                names = append(names, m.Name)
            }
        }
        sort.Strings(names)
        return names
    case *SQLiteService:
        names := make([]string, 0, len(s.modes))
        for _, m := range s.modes {
            if m != nil && m.Name != "" {
                names = append(names, m.Name)
            }
        }
        sort.Strings(names)
        return names
    default:
        return nil
    }
}



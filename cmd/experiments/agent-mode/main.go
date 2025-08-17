package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	clay "github.com/go-go-golems/clay/pkg"
	engpkg "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/pinocchio/pkg/middlewares/agentmode"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "agent-mode"}

type Cmd struct{}

// DummyEngine is a minimal Engine that emits a tool_call depending on the user text.
type DummyEngine struct{}

func (d *DummyEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	// Find last user text
	last := ""
	for i := len(t.Blocks) - 1; i >= 0; i-- {
		b := t.Blocks[i]
		if b.Kind == turns.BlockKindUser {
			if txt, ok := b.Payload[turns.PayloadKeyText].(string); ok {
				last = txt
			}
			break
		}
	}
	lt := strings.ToLower(last)
	if strings.Contains(lt, "time") {
		turns.AppendBlock(t, turns.NewAssistantTextBlock("I'll check the current time."))
		turns.AppendBlock(t, turns.NewToolCallBlock("call-1", "time_now", map[string]any{}))
	} else {
		turns.AppendBlock(t, turns.NewAssistantTextBlock("I'll echo your text."))
		turns.AppendBlock(t, turns.NewToolCallBlock("call-1", "echo", map[string]any{"text": last}))
	}
	return t, nil
}

var _ engpkg.Engine = (*DummyEngine)(nil)

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run agent-mode middleware experiment",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := logging.InitLoggerFromViper(); err != nil {
				return err
			}

			// Engine: dummy local engine
			var engine engpkg.Engine = &DummyEngine{}

			// Build toolbox and registry with two demo tools
			tb := middleware.NewMockToolbox()
			tb.RegisterTool("echo", "Echo text", map[string]any{"text": map[string]any{"type": "string"}}, func(ctx context.Context, args map[string]any) (any, error) {
				return args["text"], nil
			})
			tb.RegisterTool("time_now", "Return current time", map[string]any{}, func(ctx context.Context, _ map[string]any) (any, error) {
				return time.Now().Format(time.RFC3339), nil
			})

			// Registry for engine tool exposure
			reg := tools.NewInMemoryToolRegistry()
			_ = reg.RegisterTool("echo", tools.ToolDefinition{Name: "echo", Description: "Echo text"})
			_ = reg.RegisterTool("time_now", tools.ToolDefinition{Name: "time_now", Description: "Return current time"})

			// Middlewares: agent mode service, then tools
			svc := agentmode.NewStaticService([]*agentmode.AgentMode{
				{Name: "chat", AllowedTools: []string{"echo"}, Prompt: "You are in chat mode; prefer concise helpful answers."},
				{Name: "clock", AllowedTools: []string{"time_now"}, Prompt: "You are in clock mode; you may use time_now when necessary."},
			})
			amw := agentmode.NewMiddleware(svc, agentmode.DefaultConfig())
			toolMw := middleware.NewToolMiddleware(tb, middleware.ToolConfig{MaxIterations: 3, Timeout: 15 * time.Second})
			engine = middleware.NewEngineWithMiddleware(engine, amw, toolMw)

			// Build the Turn
			turn := &turns.Turn{Data: map[string]any{}}
			// seed directly
			turns.AppendBlock(turn, turns.NewSystemTextBlock("You are a helpful assistant."))
			turns.AppendBlock(turn, turns.NewUserTextBlock("What time is it?"))

			// Attach registry and generic tool config so engine advertises tools
			turn.Data[turns.DataKeyToolRegistry] = reg
			turn.Data[turns.DataKeyToolConfig] = engpkg.ToolConfig{Enabled: true, ToolChoice: engpkg.ToolChoiceAuto, MaxIterations: 3, ExecutionTimeout: 15 * time.Second}

			// Mode flag
			mode, _ := cmd.Flags().GetString("mode")
			if mode == "" {
				mode = "clock"
			}
			turn.Data[agentmode.DataKeyAgentMode] = mode

			// Run
			updated, err := engine.RunInference(ctx, turn)
			if err != nil {
				return errors.Wrap(err, "inference failed")
			}

			// Print blocks
			fmt.Println("=== Blocks ===")
			for i, b := range updated.Blocks {
				role := b.Role
				txt, _ := b.Payload[turns.PayloadKeyText].(string)
				name, _ := b.Payload[turns.PayloadKeyName].(string)
				fmt.Printf("%02d kind=%v role=%s name=%s text=%s\n", i, b.Kind, role, name, txt)
			}
			return nil
		},
	}
	// flags
	cmd.Flags().String("mode", "clock", "agent mode to set (chat|clock)")
	return cmd
}

func main() {
	if err := clay.InitViper("pinocchio", rootCmd); err != nil {
		panic(err)
	}
	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	rootCmd.AddCommand(newCmd())
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("agent-mode experiment failed")
	}
}

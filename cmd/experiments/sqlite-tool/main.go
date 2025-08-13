package main

import (
	"context"
	"fmt"

	clay "github.com/go-go-golems/clay/pkg"
	engpkg "github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware/sqlitetool"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Dummy engine that always asks the tool to run a query.
type DummyEngine struct{}

func (d *DummyEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	turns.AppendBlock(t, turns.NewAssistantTextBlock("I'll run a SQL query."))
	turns.AppendBlock(t, turns.NewToolCallBlock("q1", "sql_query", map[string]any{"sql": "SELECT name FROM sqlite_master WHERE type='table'"}))
	return t, nil
}

var _ engpkg.Engine = (*DummyEngine)(nil)

var rootCmd = &cobra.Command{Use: "sqlite-tool"}

func main() {
	if err := clay.InitViper("pinocchio", rootCmd); err != nil {
		panic(err)
	}
	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	cmd := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := logging.InitLoggerFromViper(); err != nil {
				return err
			}
			ctx := context.Background()
			var engine engpkg.Engine = &DummyEngine{}

			// Registry and tool middleware are still needed for provider advertisement patterns.
			reg := tools.NewInMemoryToolRegistry()
			t := &turns.Turn{Data: map[string]any{turns.DataKeyToolRegistry: reg}}

			// Attach sqlite DSN
			t.Data[sqlitetool.DataKeySQLiteDSN] = "anonymized-data.db"

			// Compose sqlite middleware
			smw := sqlitetool.NewMiddleware(sqlitetool.DefaultConfig())
			engine = middleware.NewEngineWithMiddleware(engine, smw)

			// Run
			updated, err := engine.RunInference(ctx, t)
			if err != nil {
				return errors.Wrap(err, "inference failed")
			}

			// Print resulting blocks
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
	rootCmd.AddCommand(cmd)
	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("sqlite-tool failed")
	}
}

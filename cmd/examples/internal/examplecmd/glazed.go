package examplecmd

import (
	"context"

	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"
)

// NewRoot creates the common Glazed/Cobra root used by examples.
func NewRoot(use string, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}
}

// ExecuteSingleCommand initializes Glazed root plumbing, adds one Glazed command,
// and executes the example with a background context.
func ExecuteSingleCommand(root *cobra.Command, appName string, command cmds.Command, opts ...cli.CobraOption) error {
	if err := clay.InitGlazed(appName, root); err != nil {
		return err
	}

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	cobraCommand, err := cli.BuildCobraCommand(command, opts...)
	if err != nil {
		return err
	}
	root.AddCommand(cobraCommand)

	return root.ExecuteContext(context.Background())
}

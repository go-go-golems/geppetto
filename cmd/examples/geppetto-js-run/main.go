package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	gpruntime "github.com/go-go-golems/geppetto/pkg/js/runtime"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type runCommand struct {
	*cmds.CommandDescription
}

var _ cmds.WriterCommand = (*runCommand)(nil)

type runSettings struct {
	ScriptPath           string `glazed:"script"`
	ProfileRegistriesRaw string `glazed:"profile-registries"`
	Profile              string `glazed:"profile"`
	TimeoutMs            int    `glazed:"timeout-ms"`
}

func newRunCommand() (*runCommand, error) {
	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run a Geppetto JavaScript example with profile-backed inference settings"),
		cmds.WithFlags(
			fields.New("script", fields.TypeString,
				fields.WithHelp("JavaScript file to execute"),
			),
			fields.New("profile-registries", fields.TypeString,
				fields.WithDefault(defaultPinocchioProfilesPath()),
				fields.WithHelp("Comma-separated Geppetto profile registry sources (YAML/SQLite/DSN)"),
			),
			fields.New("profile", fields.TypeString,
				fields.WithDefault("default"),
				fields.WithHelp("Default inference profile slug"),
			),
			fields.New("timeout-ms", fields.TypeInteger,
				fields.WithDefault(120000),
				fields.WithHelp("Timeout in milliseconds exposed to the JS example"),
			),
		),
	)
	return &runCommand{CommandDescription: description}, nil
}

func (c *runCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, _ io.Writer) error {
	s := &runSettings{}
	if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
		return errors.Wrap(err, "decode run settings")
	}
	return runScript(ctx, s)
}

func runScript(ctx context.Context, s *runSettings) error {
	if s == nil {
		return fmt.Errorf("settings are required")
	}
	if strings.TrimSpace(s.ScriptPath) == "" {
		return fmt.Errorf("--script is required")
	}

	entries, err := profiles.ParseEngineProfileRegistrySourceEntries(s.ProfileRegistriesRaw)
	if err != nil {
		return fmt.Errorf("parse --profile-registries: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("--profile-registries must not be empty")
	}
	specs, err := profiles.ParseRegistrySourceSpecs(entries)
	if err != nil {
		return fmt.Errorf("parse registry source specs: %w", err)
	}
	chain, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return fmt.Errorf("load registry sources: %w", err)
	}
	defer func() { _ = chain.Close() }()

	profileSlug, err := profiles.ParseEngineProfileSlug(s.Profile)
	if err != nil {
		return fmt.Errorf("parse --profile: %w", err)
	}

	rt, err := gpruntime.NewRuntime(ctx, gpruntime.Options{
		ModuleOptions: gp.Options{
			EngineProfileRegistry:     chain,
			EngineProfileRegistrySpec: entries,
			UseDefaultProfileResolve:  true,
			DefaultProfileResolve: profiles.ResolveInput{
				EngineProfileSlug: profileSlug,
			},
		},
		IncludeDefaultModules: true,
	})
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close(ctx) }()

	absScript, err := filepath.Abs(s.ScriptPath)
	if err != nil {
		return err
	}
	if err := rt.VM.Set("GEPPETTO_EXAMPLE", map[string]any{
		"profileRegistries": entries,
		"profile":           s.Profile,
		"timeoutMs":         s.TimeoutMs,
		"script":            absScript,
	}); err != nil {
		return fmt.Errorf("set GEPPETTO_EXAMPLE: %w", err)
	}

	b, err := os.ReadFile(absScript)
	if err != nil {
		return err
	}
	_, err = rt.VM.RunScript(absScript, string(b))
	return err
}

func defaultPinocchioProfilesPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".config", "pinocchio", "profiles.yaml")
	}
	return filepath.Join(home, ".config", "pinocchio", "profiles.yaml")
}

func main() {
	root := examplecmd.NewRoot("geppetto-js-run", "Run Geppetto JavaScript examples")
	cmd, err := newRunCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(examplecmd.ExecuteSingleCommand(root, "geppetto", cmd))
}

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/geppetto/cmd/examples/internal/examplecmd"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	gp "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	gpruntime "github.com/go-go-golems/geppetto/pkg/js/runtime"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
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
	b, err := os.ReadFile(absScript)
	if err != nil {
		return err
	}
	ret, err := rt.Owner.Call(ctx, "geppetto-js-run.runScript", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if setErr := vm.Set("GEPPETTO_EXAMPLE", map[string]any{
			"profileRegistries": entries,
			"profile":           s.Profile,
			"timeoutMs":         s.TimeoutMs,
			"script":            absScript,
		}); setErr != nil {
			return nil, fmt.Errorf("set GEPPETTO_EXAMPLE: %w", setErr)
		}
		value, runErr := vm.RunScript(absScript, string(b))
		if runErr != nil {
			return nil, runErr
		}
		if value != nil {
			if promise, ok := value.Export().(*goja.Promise); ok {
				return promise, nil
			}
		}
		return value, nil
	})
	if err != nil {
		return err
	}
	if promise, ok := ret.(*goja.Promise); ok {
		return waitForScriptPromise(ctx, rt, promise, s.TimeoutMs)
	}
	return nil
}

func waitForScriptPromise(ctx context.Context, rt *gojengine.Runtime, promise *goja.Promise, timeoutMs int) error {
	if promise == nil {
		return nil
	}
	waitCtx := ctx
	var cancel context.CancelFunc
	if timeoutMs > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs+5000)*time.Millisecond)
		defer cancel()
	}
	for {
		ret, err := rt.Owner.Call(waitCtx, "geppetto-js-run.promiseState", func(_ context.Context, _ *goja.Runtime) (any, error) {
			return promise.State(), nil
		})
		if err != nil {
			return err
		}
		switch state := ret.(type) {
		case goja.PromiseState:
			switch state {
			case goja.PromiseStateFulfilled:
				return nil
			case goja.PromiseStateRejected:
				result, resultErr := rt.Owner.Call(waitCtx, "geppetto-js-run.promiseResult", func(_ context.Context, _ *goja.Runtime) (any, error) {
					if promise.Result() == nil {
						return "", nil
					}
					return promise.Result().String(), nil
				})
				if resultErr != nil {
					return resultErr
				}
				return fmt.Errorf("script promise rejected: %s", result)
			case goja.PromiseStatePending:
				// Keep polling while owner-thread posts and promise callbacks settle.
			}
		}
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("wait for script promise: %w", waitCtx.Err())
		case <-time.After(10 * time.Millisecond):
		}
	}
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

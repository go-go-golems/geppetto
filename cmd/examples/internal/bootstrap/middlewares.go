package bootstrap

import (
	"context"
	"strings"

	geppettobootstrap "github.com/go-go-golems/geppetto/pkg/cli/bootstrap"
	geppettosections "github.com/go-go-golems/geppetto/pkg/sections"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedconfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func AppBootstrapConfig() geppettobootstrap.AppBootstrapConfig {
	cfg := geppettobootstrap.AppBootstrapConfig{
		AppName:          "pinocchio",
		EnvPrefix:        "PINOCCHIO",
		ConfigFileMapper: configFileMapper,
		NewProfileSection: func() (schema.Section, error) {
			return geppettosections.NewProfileSettingsSection()
		},
		BuildBaseSections: func() ([]schema.Section, error) {
			return geppettosections.CreateGeppettoSections()
		},
	}
	cfg.ConfigPlanBuilder = func(parsed *values.Values) (*glazedconfig.Plan, error) {
		explicit := ""
		if parsed != nil {
			commandSettings := &cli.CommandSettings{}
			if err := parsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err == nil {
				explicit = strings.TrimSpace(commandSettings.ConfigFile)
			}
		}

		return glazedconfig.NewPlan(
			glazedconfig.WithLayerOrder(
				glazedconfig.LayerSystem,
				glazedconfig.LayerUser,
				glazedconfig.LayerExplicit,
			),
			glazedconfig.WithDedupePaths(),
		).Add(
			glazedconfig.SystemAppConfig(cfg.AppName).Named("system-app-config").Kind("app-config"),
			glazedconfig.HomeAppConfig(cfg.AppName).Named("home-app-config").Kind("app-config"),
			glazedconfig.XDGAppConfig(cfg.AppName).Named("xdg-app-config").Kind("app-config"),
			glazedconfig.ExplicitFile(explicit).Named("explicit-config-file").Kind("explicit-file"),
		), nil
	}
	return cfg
}

func GetCobraCommandMiddlewares(parsed *values.Values, cmd *cobra.Command, args []string) ([]sources.Middleware, error) {
	cfg := AppBootstrapConfig()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return []sources.Middleware{
		sources.FromCobra(cmd, fields.WithSource("cobra")),
		sources.FromArgs(args, fields.WithSource("arguments")),
		sources.FromEnv(cfg.EnvPrefix, fields.WithSource("env")),
		sources.FromConfigPlanBuilder(func(_ctx context.Context, parsedValues *values.Values) (*glazedconfig.Plan, error) {
			return cfg.ConfigPlanBuilder(parsedValues)
		},
			sources.WithConfigFileMapper(cfg.ConfigFileMapper),
			sources.WithParseOptions(fields.WithSource("config")),
		),
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	}, nil
}

func configFileMapper(rawConfig interface{}) (map[string]map[string]interface{}, error) {
	configMap, ok := rawConfig.(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("expected map[string]interface{}, got %T", rawConfig)
	}

	result := make(map[string]map[string]interface{})
	for key, value := range configMap {
		sectionValues, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		result[key] = sectionValues
	}
	return result, nil
}

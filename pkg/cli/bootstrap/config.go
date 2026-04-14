package bootstrap

import (
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedconfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/pkg/errors"
)

const ProfileSettingsSectionSlug = "profile-settings"

type ConfigPlanBuilder func(parsed *values.Values) (*glazedconfig.Plan, error)

type AppBootstrapConfig struct {
	AppName           string
	EnvPrefix         string
	ConfigFileMapper  sources.ConfigFileMapper
	NewProfileSection func() (schema.Section, error)
	BuildBaseSections func() ([]schema.Section, error)
	ConfigPlanBuilder ConfigPlanBuilder
}

func (c AppBootstrapConfig) Validate() error {
	if strings.TrimSpace(c.AppName) == "" {
		return errors.New("app bootstrap config: app name is required")
	}
	if strings.TrimSpace(c.EnvPrefix) == "" {
		return errors.New("app bootstrap config: env prefix is required")
	}
	if c.ConfigFileMapper == nil {
		return errors.New("app bootstrap config: config file mapper is required")
	}
	if c.NewProfileSection == nil {
		return errors.New("app bootstrap config: profile section builder is required")
	}
	if c.BuildBaseSections == nil {
		return errors.New("app bootstrap config: base sections builder is required")
	}
	return nil
}

func (c AppBootstrapConfig) normalizedAppName() string {
	return strings.TrimSpace(c.AppName)
}

func (c AppBootstrapConfig) normalizedEnvPrefix() string {
	return strings.TrimSpace(c.EnvPrefix)
}

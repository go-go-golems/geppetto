package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	geppettomodule "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "geppetto"

type Config struct {
	Profile      string `json:"profile,omitempty"`
	Registry     string `json:"registry,omitempty"`
	AllowNetwork bool   `json:"allowNetwork,omitempty"`
	AllowTools   bool   `json:"allowTools,omitempty"`
}

type HostServices interface {
	GeppettoOptions(ctx context.Context, cfg Config) (geppettomodule.Options, error)
}

var configSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "profile": {"type": "string", "description": "Optional default engine profile selector interpreted by host services."},
    "registry": {"type": "string", "description": "Optional profile registry selector interpreted by host services."},
    "allowNetwork": {"type": "boolean", "description": "Explicitly allow host services to configure network-backed inference engines."},
    "allowTools": {"type": "boolean", "description": "Explicitly allow host services to expose Go tool registries to JavaScript."}
  },
  "additionalProperties": false
}`)

func Register(registry *providerapi.Registry) error {
	return registry.Package(PackageID, providerapi.Module{
		Name:         geppettomodule.ModuleName,
		DefaultAs:    geppettomodule.ModuleName,
		Description:  "Geppetto inference, turns, engine, profile, and runner helpers exposed as require(\"geppetto\").",
		ConfigSchema: configSchema,
		New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
			cfg, err := decodeConfig(ctx.Config)
			if err != nil {
				return nil, fmt.Errorf("geppetto provider config: %w", err)
			}
			host, ok := ctx.Host.(HostServices)
			if !ok || host == nil {
				return nil, fmt.Errorf("geppetto provider requires geppetto provider HostServices")
			}
			opts, err := host.GeppettoOptions(ctx.Context, cfg)
			if err != nil {
				return nil, fmt.Errorf("geppetto provider host options: %w", err)
			}
			return geppettomodule.NewLoader(opts), nil
		},
	})
}

func decodeConfig(data json.RawMessage) (Config, error) {
	cfg := Config{}
	if len(data) > 0 && string(data) != "null" {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja_nodejs/require"
	profiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	geppettomodule "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "geppetto"

type Config struct {
	Profile           string       `json:"profile,omitempty"`
	Registry          string       `json:"registry,omitempty"`
	ProfileRegistries []string     `json:"profileRegistries,omitempty"`
	DefaultProfile    string       `json:"defaultProfile,omitempty"`
	AllowRegistryLoad bool         `json:"allowRegistryLoad,omitempty"`
	AllowNetwork      bool         `json:"allowNetwork,omitempty"`
	AllowTools        bool         `json:"allowTools,omitempty"`
	EnableStorage     bool         `json:"enableStorage,omitempty"`
	Turns             *TurnsConfig `json:"turns,omitempty"`
}

type TurnsConfig struct {
	DSN      string `json:"dsn,omitempty"`
	DB       string `json:"db,omitempty"`
	Default  bool   `json:"default,omitempty"`
	Phase    string `json:"phase,omitempty"`
	Readonly bool   `json:"readonly,omitempty"`
}

type HostServices interface {
	GeppettoOptions(ctx context.Context, cfg Config) (geppettomodule.Options, error)
}

type StorageHostServices interface {
	GeppettoTurnStores(ctx context.Context, cfg Config) (geppettomodule.StorageOptions, error)
}

var configSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "profile": {"type": "string", "description": "Legacy optional default engine profile selector interpreted by host services."},
    "registry": {"type": "string", "description": "Legacy optional profile registry selector/source interpreted by host services."},
    "profileRegistries": {
      "description": "Geppetto engine profile registry sources to load (YAML path, yaml:PATH, yaml://PATH, SQLite path, sqlite:PATH, sqlite-dsn:DSN).",
      "oneOf": [
        {"type": "string"},
        {"type": "array", "items": {"type": "string"}}
      ]
    },
    "defaultProfile": {"type": "string", "description": "Default engine profile slug for gp.inferenceProfiles.resolve() when no profile is supplied."},
    "allowRegistryLoad": {"type": "boolean", "description": "Allow this provider instance to load profileRegistries itself."},
    "allowNetwork": {"type": "boolean", "description": "Explicitly allow host services to configure network-backed inference engines."},
    "allowTools": {"type": "boolean", "description": "Explicitly allow host services to expose Go tool registries to JavaScript."},
    "enableStorage": {"type": "boolean", "description": "Allow this provider instance to request host-backed turn storage."},
    "turns": {
      "type": "object",
      "description": "Host-mediated turn-store configuration, such as Pinocchio-style turns DSNs.",
      "properties": {
        "dsn": {"type": "string", "description": "Turn-store DSN interpreted by host services."},
        "db": {"type": "string", "description": "Turn-store database path interpreted by host services."},
        "default": {"type": "boolean", "description": "Install the resolved turn store as the module default persister."},
        "phase": {"type": "string", "description": "Preferred persisted phase, usually final."},
        "readonly": {"type": "boolean", "description": "Open the store read-only when supported by the host."}
      },
      "additionalProperties": false
    }
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
			if err := applyConfigRegistryOptions(ctx.Context, cfg, &opts); err != nil {
				return nil, err
			}
			if err := applyConfigStorageOptions(ctx.Context, cfg, ctx.Host, &opts); err != nil {
				return nil, err
			}
			return geppettomodule.NewLoader(opts), nil
		},
	})
}

func decodeConfig(data json.RawMessage) (Config, error) {
	cfg := Config{}
	if len(data) > 0 && string(data) != "null" {
		type configAlias Config
		var raw struct {
			*configAlias
			ProfileRegistries any `json:"profileRegistries"`
		}
		raw.configAlias = (*configAlias)(&cfg)
		if err := json.Unmarshal(data, &raw); err != nil {
			return Config{}, err
		}
		if raw.ProfileRegistries != nil {
			entries, err := decodeSourceEntries(raw.ProfileRegistries)
			if err != nil {
				return Config{}, err
			}
			cfg.ProfileRegistries = entries
		}
	}
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = cfg.Profile
	}
	if len(cfg.ProfileRegistries) == 0 && strings.TrimSpace(cfg.Registry) != "" {
		cfg.ProfileRegistries = []string{strings.TrimSpace(cfg.Registry)}
	}
	return cfg, nil
}

func decodeSourceEntries(raw any) ([]string, error) {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		return profiles.ParseEngineProfileRegistrySourceEntries(v)
	case []any:
		out := make([]string, 0, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return nil, fmt.Errorf("profileRegistries[%d] must be a non-empty string", i)
			}
			out = append(out, strings.TrimSpace(s))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("profileRegistries must be a string or string array")
	}
}

func applyConfigRegistryOptions(ctx context.Context, cfg Config, opts *geppettomodule.Options) error {
	if opts == nil {
		return nil
	}
	if len(cfg.ProfileRegistries) > 0 {
		if !cfg.AllowRegistryLoad {
			return fmt.Errorf("geppetto provider profileRegistries require allowRegistryLoad=true")
		}
		specs, err := profiles.ParseRegistrySourceSpecs(cfg.ProfileRegistries)
		if err != nil {
			return fmt.Errorf("geppetto provider profileRegistries: %w", err)
		}
		chain, err := profiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
		if err != nil {
			return fmt.Errorf("geppetto provider load profileRegistries: %w", err)
		}
		opts.EngineProfileRegistry = chain
		opts.EngineProfileRegistrySpec = append([]string(nil), cfg.ProfileRegistries...)
	}
	if strings.TrimSpace(cfg.DefaultProfile) != "" {
		profileSlug, err := profiles.ParseEngineProfileSlug(cfg.DefaultProfile)
		if err != nil {
			return fmt.Errorf("geppetto provider defaultProfile: %w", err)
		}
		opts.UseDefaultProfileResolve = true
		opts.DefaultProfileResolve.EngineProfileSlug = profileSlug
	}
	if strings.TrimSpace(cfg.Registry) != "" && len(cfg.ProfileRegistries) == 0 {
		registrySlug, err := profiles.ParseRegistrySlug(cfg.Registry)
		if err == nil {
			opts.UseDefaultProfileResolve = true
			opts.DefaultProfileResolve.RegistrySlug = registrySlug
		}
	}
	return nil
}

func applyConfigStorageOptions(ctx context.Context, cfg Config, host any, opts *geppettomodule.Options) error {
	if opts == nil {
		return nil
	}
	if cfg.Turns != nil && !cfg.EnableStorage {
		return fmt.Errorf("geppetto provider turns config requires enableStorage=true")
	}
	if !cfg.EnableStorage {
		return nil
	}
	storageHost, ok := host.(StorageHostServices)
	if !ok || storageHost == nil {
		return fmt.Errorf("geppetto provider enableStorage requires GeppettoTurnStores host capability")
	}
	storage, err := storageHost.GeppettoTurnStores(ctx, cfg)
	if err != nil {
		return fmt.Errorf("geppetto provider host turn stores: %w", err)
	}
	opts.EnableStorage = true
	if storage.Default != nil {
		opts.DefaultTurnStore = storage.Default
	}
	if len(storage.Stores) > 0 {
		if opts.TurnStores == nil {
			opts.TurnStores = map[string]geppettomodule.TurnStore{}
		}
		for name, store := range storage.Stores {
			if strings.TrimSpace(name) == "" || store == nil {
				continue
			}
			opts.TurnStores[strings.TrimSpace(name)] = store
		}
	}
	if cfg.Turns != nil && cfg.Turns.Default && opts.DefaultTurnStore != nil {
		opts.DefaultPersister = opts.DefaultTurnStore
	}
	return nil
}

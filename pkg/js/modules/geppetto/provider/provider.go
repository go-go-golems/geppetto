package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/geppetto/pkg/engineprofiles"
	geppettomodule "github.com/go-go-golems/geppetto/pkg/js/modules/geppetto"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "geppetto"

const configSectionSlug = "geppetto"

type Config struct {
	DefaultProfileRegistries []string `json:"defaultProfileRegistries,omitempty"`
	DefaultProfile           string   `json:"defaultProfile,omitempty"`
}

type HostServices interface {
	GeppettoOptions(ctx context.Context, cfg Config) (geppettomodule.Options, error)
}

var configSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "defaultProfileRegistries": {
      "description": "Default Geppetto engine profile registry sources to load (YAML path, yaml:PATH, yaml://PATH, SQLite path, sqlite:PATH, sqlite-dsn:DSN).",
      "oneOf": [
        {"type": "string"},
        {"type": "array", "items": {"type": "string"}}
      ]
    },
    "defaultProfile": {"type": "string", "description": "Default engine profile slug for gp.inferenceProfiles.resolve() when no profile is supplied."}
  },
  "additionalProperties": false
}`)

func Register(registry *providerapi.ProviderRegistry) error {
	capability := capability{}
	return registry.Package(PackageID,
		providerapi.Module{
			Name:         geppettomodule.ModuleName,
			DefaultAs:    geppettomodule.ModuleName,
			Description:  "Geppetto inference, turns, engine, profile, and runner helpers exposed as require(\"geppetto\").",
			ConfigSchema: configSchema,
			NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
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
				return geppettomodule.NewLoader(opts), nil
			},
		},
		providerapi.WithPackageCapability(capability),
	)
}

func decodeConfig(data json.RawMessage) (Config, error) {
	cfg := Config{}
	if len(data) > 0 && string(data) != "null" {
		type configAlias Config
		var raw struct {
			*configAlias
			DefaultProfileRegistries any `json:"defaultProfileRegistries"`
		}
		raw.configAlias = (*configAlias)(&cfg)
		if err := json.Unmarshal(data, &raw); err != nil {
			return Config{}, err
		}
		if raw.DefaultProfileRegistries != nil {
			entries, err := decodeSourceEntries(raw.DefaultProfileRegistries, "defaultProfileRegistries")
			if err != nil {
				return Config{}, err
			}
			cfg.DefaultProfileRegistries = entries
		}
	}
	return cfg, nil
}

func decodeSourceEntries(raw any, fieldName string) ([]string, error) {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		return engineprofiles.ParseEngineProfileRegistrySourceEntries(v)
	case []any:
		out := make([]string, 0, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return nil, fmt.Errorf("%s[%d] must be a non-empty string", fieldName, i)
			}
			out = append(out, strings.TrimSpace(s))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s must be a string or string array", fieldName)
	}
}

func applyConfigRegistryOptions(ctx context.Context, cfg Config, opts *geppettomodule.Options) error {
	if opts == nil {
		return nil
	}
	if len(cfg.DefaultProfileRegistries) > 0 {
		specs, err := engineprofiles.ParseRegistrySourceSpecs(cfg.DefaultProfileRegistries)
		if err != nil {
			return fmt.Errorf("geppetto provider defaultProfileRegistries: %w", err)
		}
		chain, err := engineprofiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
		if err != nil {
			return fmt.Errorf("geppetto provider load defaultProfileRegistries: %w", err)
		}
		opts.EngineProfileRegistry = chain
		opts.EngineProfileRegistrySpec = append([]string(nil), cfg.DefaultProfileRegistries...)
	}
	if strings.TrimSpace(cfg.DefaultProfile) != "" {
		profileSlug, err := engineprofiles.ParseEngineProfileSlug(cfg.DefaultProfile)
		if err != nil {
			return fmt.Errorf("geppetto provider defaultProfile: %w", err)
		}
		opts.UseDefaultProfileResolve = true
		opts.DefaultProfileResolve.EngineProfileSlug = profileSlug
	}
	return nil
}

type capability struct{}

func (capability) CapabilityID() string { return "geppetto-config" }

func (capability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
	section, err := schema.NewSection(configSectionSlug, "Geppetto",
		schema.WithPrefix("geppetto-"),
		schema.WithFields(
			fields.New("default-profile-registries", fields.TypeStringList, fields.WithHelp("Default Geppetto profile registry sources for this module instance")),
			fields.New("default-profile", fields.TypeString, fields.WithHelp("Default Geppetto engine profile slug for this module instance")),
		),
	)
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

func (capability) XGojaConfigSection(providerapi.SectionRequest, providerapi.ModuleDescriptor) (schema.Section, error) {
	return xgojaConfigSection()
}

func (capability) XGojaConfigFromGlazed(_ context.Context, req providerapi.XGojaConfigRequest) (*values.SectionValues, error) {
	out, err := values.NewSectionValues(req.ConfigSection)
	if err != nil {
		return nil, err
	}
	if req.GlazedValues == nil {
		return out, nil
	}
	if err := copyGlazedField(req.GlazedValues, out, "default-profile-registries", "defaultProfileRegistries"); err != nil {
		return nil, err
	}
	if err := copyGlazedField(req.GlazedValues, out, "default-profile", "defaultProfile"); err != nil {
		return nil, err
	}
	return out, nil
}

func xgojaConfigSection() (schema.Section, error) {
	return schema.NewSection(configSectionSlug+"-xgoja", "Geppetto xgoja config",
		schema.WithFields(
			fields.New("defaultProfileRegistries", fields.TypeStringList),
			fields.New("defaultProfile", fields.TypeString),
		),
	)
}

func copyGlazedField(vals *values.Values, out *values.SectionValues, glazedField, xgojaField string) error {
	field, ok := vals.GetField(configSectionSlug, glazedField)
	if !ok {
		return nil
	}
	definition, ok := out.Section.GetDefinitions().Get(xgojaField)
	if !ok {
		return fmt.Errorf("xgoja config field %q not found", xgojaField)
	}
	return out.Fields.UpdateWithLog(xgojaField, definition, field.Value, field.Log...)
}

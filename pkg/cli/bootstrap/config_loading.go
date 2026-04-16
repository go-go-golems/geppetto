package bootstrap

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

// DefaultConfigFileMapper converts a raw YAML config (map[string]interface{})
// into section-keyed layers (map[string]map[string]interface{}).
// Non-map top-level values are silently skipped.
func DefaultConfigFileMapper(rawConfig interface{}) (map[string]map[string]interface{}, error) {
	configMap, ok := rawConfig.(map[string]interface{})
	if !ok {
		return nil, nil
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

func resolveConfigMiddleware(cfg AppBootstrapConfig, parsed *values.Values) (sources.Middleware, *ResolvedCLIConfigFiles, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	configFiles, err := ResolveCLIConfigFilesResolved(cfg, parsed)
	if err != nil {
		return nil, nil, err
	}

	return sources.FromResolvedFiles(
		configFiles.Files,
		sources.WithConfigFileMapper(cfg.ConfigFileMapper),
		sources.WithParseOptions(fields.WithSource("config")),
	), configFiles, nil
}

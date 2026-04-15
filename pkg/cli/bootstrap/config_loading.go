package bootstrap

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

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

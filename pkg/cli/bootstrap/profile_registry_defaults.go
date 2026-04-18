package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultProfileRegistrySources(cfg AppBootstrapConfig) []string {
	appName := strings.TrimSpace(cfg.AppName)
	if appName == "" {
		return nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configDir) == "" {
		return nil
	}

	path := filepath.Join(configDir, appName, "profiles.yaml")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil
	}

	return []string{path}
}

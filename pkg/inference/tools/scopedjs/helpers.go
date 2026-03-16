package scopedjs

import "strings"

func NormalizeNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	ret := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		ret = append(ret, trimmed)
	}
	return ret
}

func ensureSentence(v string) string {
	if v == "" {
		return v
	}
	if strings.HasSuffix(v, ".") || strings.HasSuffix(v, "!") || strings.HasSuffix(v, "?") {
		return v
	}
	return v + "."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneManifest(in EnvironmentManifest) EnvironmentManifest {
	return EnvironmentManifest{
		Modules:        append([]ModuleDoc(nil), in.Modules...),
		Globals:        append([]GlobalDoc(nil), in.Globals...),
		Helpers:        append([]HelperDoc(nil), in.Helpers...),
		BootstrapFiles: append([]string(nil), in.BootstrapFiles...),
	}
}

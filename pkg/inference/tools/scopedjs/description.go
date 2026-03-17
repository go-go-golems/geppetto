package scopedjs

import (
	"fmt"
	"strings"
)

func BuildDescription(desc ToolDescription, manifest EnvironmentManifest, opts EvalOptions) string {
	parts := make([]string, 0, 8)

	summary := strings.TrimSpace(desc.Summary)
	if summary == "" {
		summary = "Execute JavaScript inside a prepared scoped runtime."
	}
	parts = append(parts, ensureSentence(summary))

	if modules := renderModules(manifest.Modules); modules != "" {
		parts = append(parts, "Available modules: "+modules+".")
	}
	if globals := renderGlobals(manifest.Globals); globals != "" {
		parts = append(parts, "Available globals: "+globals+".")
	}
	if helpers := renderHelpers(manifest.Helpers); helpers != "" {
		parts = append(parts, "Helpers: "+helpers+".")
	}
	if bootstrap := strings.Join(NormalizeNonEmptyStrings(manifest.BootstrapFiles), ", "); bootstrap != "" {
		parts = append(parts, "Preloaded bootstrap files: "+bootstrap+".")
	}

	parts = append(parts, "Use return to provide the final result.")

	switch opts.StateMode {
	case StatePerCall:
		parts = append(parts, "Each call uses a fresh runtime.")
	case StatePerSession:
		parts = append(parts, "Runtime state persists within the current session.")
	case StateShared:
		parts = append(parts, "Runtime state may be shared across calls.")
	}

	for _, note := range desc.Notes {
		trimmed := strings.TrimSpace(note)
		if trimmed == "" {
			continue
		}
		parts = append(parts, ensureSentence(trimmed))
	}

	snippets := NormalizeNonEmptyStrings(desc.StarterSnippets)
	if len(snippets) > 0 {
		parts = append(parts, fmt.Sprintf("Starter snippets: %s.", strings.Join(snippets, " | ")))
	}

	return strings.Join(parts, " ")
}

func renderModules(mods []ModuleDoc) string {
	parts := make([]string, 0, len(mods))
	for _, mod := range mods {
		name := strings.TrimSpace(mod.Name)
		if name == "" {
			continue
		}
		if len(mod.Exports) == 0 {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", name, strings.Join(NormalizeNonEmptyStrings(mod.Exports), ", ")))
	}
	return strings.Join(parts, "; ")
}

func renderGlobals(globals []GlobalDoc) string {
	parts := make([]string, 0, len(globals))
	for _, global := range globals {
		name := strings.TrimSpace(global.Name)
		if name == "" {
			continue
		}
		typ := strings.TrimSpace(global.Type)
		if typ == "" {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", name, typ))
	}
	return strings.Join(parts, ", ")
}

func renderHelpers(helpers []HelperDoc) string {
	parts := make([]string, 0, len(helpers))
	for _, helper := range helpers {
		switch {
		case strings.TrimSpace(helper.Signature) != "":
			parts = append(parts, strings.TrimSpace(helper.Signature))
		case strings.TrimSpace(helper.Name) != "":
			parts = append(parts, strings.TrimSpace(helper.Name))
		}
	}
	return strings.Join(parts, " | ")
}

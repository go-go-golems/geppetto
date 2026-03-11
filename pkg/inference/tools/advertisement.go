package tools

import (
	"context"
	"sort"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
)

// AdvertisedToolDefinitionsFromContext returns the runtime-advertised tool schemas from the
// live registry in context. Persisted turn snapshots are intentionally excluded from this path.
func AdvertisedToolDefinitionsFromContext(ctx context.Context) []engine.ToolDefinition {
	reg, ok := RegistryFrom(ctx)
	if !ok || reg == nil {
		return nil
	}

	defs := make([]engine.ToolDefinition, 0, len(reg.ListTools()))
	for _, td := range reg.ListTools() {
		defs = append(defs, engine.ToolDefinition{
			Name:        td.Name,
			Description: td.Description,
			Parameters:  td.Parameters,
			Examples:    []engine.ToolExample{},
			Tags:        append([]string(nil), td.Tags...),
			Version:     td.Version,
		})
	}

	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})

	return defs
}

package middlewarecfg

import (
	"encoding/json"
	"sort"
)

// DebugPathTrace is a stable debug view for one resolved path.
type DebugPathTrace struct {
	Path  string      `json:"path"`
	Value any         `json:"value,omitempty"`
	Steps []ParseStep `json:"steps,omitempty"`
}

// DebugPayload is a stable debug serialization model for resolved middleware config.
type DebugPayload struct {
	Config  map[string]any      `json:"config,omitempty"`
	Sources []ResolvedSourceRef `json:"sources,omitempty"`
	Paths   []DebugPathTrace    `json:"paths,omitempty"`
}

// BuildDebugPayload builds a deterministic debug payload for resolved config and traces.
func (r *ResolvedConfig) BuildDebugPayload() *DebugPayload {
	if r == nil {
		return &DebugPayload{}
	}

	paths := r.debugPathsInStableOrder()
	tracePaths := make([]DebugPathTrace, 0, len(paths))
	for _, path := range paths {
		trace, ok := r.Trace[path]
		if ok {
			tracePaths = append(tracePaths, DebugPathTrace{
				Path:  path,
				Value: copyAny(trace.Value),
				Steps: r.History(path),
			})
			continue
		}
		if value, ok := r.PathValues[path]; ok {
			tracePaths = append(tracePaths, DebugPathTrace{
				Path:  path,
				Value: copyAny(value),
			})
			continue
		}
		tracePaths = append(tracePaths, DebugPathTrace{Path: path})
	}

	sources := make([]ResolvedSourceRef, 0, len(r.Sources))
	for _, source := range r.Sources {
		sources = append(sources, ResolvedSourceRef{
			Name:  source.Name,
			Layer: source.Layer,
		})
	}

	return &DebugPayload{
		Config:  copyStringAnyMap(r.Config),
		Sources: sources,
		Paths:   tracePaths,
	}
}

// MarshalDebugPayload serializes deterministic resolved config debug payload to JSON.
func (r *ResolvedConfig) MarshalDebugPayload() ([]byte, error) {
	payload := r.BuildDebugPayload()
	return json.Marshal(payload)
}

func (r *ResolvedConfig) debugPathsInStableOrder() []string {
	if r == nil {
		return nil
	}
	if len(r.OrderedPaths) > 0 {
		return append([]string(nil), r.OrderedPaths...)
	}
	seen := map[string]struct{}{}
	paths := make([]string, 0, len(r.PathValues)+len(r.Trace))
	for path := range r.PathValues {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for path := range r.Trace {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

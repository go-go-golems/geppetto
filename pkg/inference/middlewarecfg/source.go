package middlewarecfg

import (
	"fmt"
	"sort"
	"strings"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/profiles"
)

// SourceLayer identifies a source precedence bucket for middleware configuration.
type SourceLayer string

const (
	SourceLayerSchemaDefaults SourceLayer = "schema-defaults"
	SourceLayerProfile        SourceLayer = "profile"
	SourceLayerConfigFile     SourceLayer = "config-file"
	SourceLayerEnvironment    SourceLayer = "environment"
	SourceLayerFlags          SourceLayer = "flags"
	SourceLayerRequest        SourceLayer = "request"
)

var sourceLayerPrecedence = map[SourceLayer]int{
	SourceLayerSchemaDefaults: 0,
	SourceLayerProfile:        1,
	SourceLayerConfigFile:     2,
	SourceLayerEnvironment:    3,
	SourceLayerFlags:          4,
	SourceLayerRequest:        5,
}

// Source provides middleware config payloads for a given middleware use instance.
type Source interface {
	Name() string
	Layer() SourceLayer
	Payload(def Definition, use gepprofiles.MiddlewareUse) (map[string]any, bool, error)
}

type orderedSource struct {
	source Source
	index  int
}

func canonicalOrderedSources(sources []Source) ([]orderedSource, error) {
	out := make([]orderedSource, 0, len(sources))
	for i, source := range sources {
		if source == nil {
			continue
		}
		layer := source.Layer()
		if _, ok := sourceLayerPrecedence[layer]; !ok {
			return nil, fmt.Errorf("unsupported middleware config source layer %q", layer)
		}
		if strings.TrimSpace(source.Name()) == "" {
			return nil, fmt.Errorf("middleware config source name is empty")
		}
		out = append(out, orderedSource{source: source, index: i})
	}

	sort.Slice(out, func(i, j int) bool {
		li := sourceLayerPrecedence[out[i].source.Layer()]
		lj := sourceLayerPrecedence[out[j].source.Layer()]
		if li != lj {
			return li < lj
		}
		ni := strings.TrimSpace(out[i].source.Name())
		nj := strings.TrimSpace(out[j].source.Name())
		if ni != nj {
			return ni < nj
		}
		return out[i].index < out[j].index
	})
	return out, nil
}

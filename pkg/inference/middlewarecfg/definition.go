package middlewarecfg

import (
	"context"

	gepmiddleware "github.com/go-go-golems/geppetto/pkg/inference/middleware"
)

// BuildDeps carries application-owned dependencies required to build middleware instances.
type BuildDeps struct {
	Values map[string]any
}

// Get retrieves a named dependency from the build dependency set.
func (d BuildDeps) Get(key string) (any, bool) {
	if len(d.Values) == 0 {
		return nil, false
	}
	v, ok := d.Values[key]
	return v, ok
}

// Clone returns a shallow copy of the dependency container.
func (d BuildDeps) Clone() BuildDeps {
	if len(d.Values) == 0 {
		return BuildDeps{}
	}
	values := make(map[string]any, len(d.Values))
	for key, value := range d.Values {
		values[key] = value
	}
	return BuildDeps{Values: values}
}

// Definition is the middleware contract consumed by schema-first runtime composition.
type Definition interface {
	Name() string
	ConfigJSONSchema() map[string]any
	Build(ctx context.Context, deps BuildDeps, cfg any) (gepmiddleware.Middleware, error)
}

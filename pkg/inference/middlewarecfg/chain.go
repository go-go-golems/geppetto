package middlewarecfg

import (
	"context"
	"fmt"
	"strings"

	gepmiddleware "github.com/go-go-golems/geppetto/pkg/inference/middleware"
	gepprofiles "github.com/go-go-golems/geppetto/pkg/profiles"
)

// ResolvedInstance binds one middleware use to its resolved config and definition.
type ResolvedInstance struct {
	Key      string                    `json:"key,omitempty"`
	Use      gepprofiles.MiddlewareUse `json:"use"`
	Resolved *ResolvedConfig           `json:"resolved,omitempty"`
	Def      Definition                `json:"-"`
}

// MiddlewareInstanceKey returns a stable diagnostic key for one middleware use.
func MiddlewareInstanceKey(use gepprofiles.MiddlewareUse, index int) string {
	name := strings.TrimSpace(use.Name)
	if name == "" {
		name = "middleware"
	}
	if id := strings.TrimSpace(use.ID); id != "" {
		return fmt.Sprintf("%s#%s", name, id)
	}
	return fmt.Sprintf("%s[%d]", name, index)
}

// MiddlewareUseIsEnabled reports whether a middleware use should be built.
func MiddlewareUseIsEnabled(use gepprofiles.MiddlewareUse) bool {
	return use.Enabled == nil || *use.Enabled
}

// BuildChain builds middleware instances from resolved configs in deterministic input order.
func BuildChain(ctx context.Context, deps BuildDeps, resolved []ResolvedInstance) ([]gepmiddleware.Middleware, error) {
	if ctx == nil {
		return nil, fmt.Errorf("build middleware chain: context is nil")
	}
	if len(resolved) == 0 {
		return nil, nil
	}

	chain := make([]gepmiddleware.Middleware, 0, len(resolved))
	for i, instance := range resolved {
		instanceKey := strings.TrimSpace(instance.Key)
		if instanceKey == "" {
			instanceKey = MiddlewareInstanceKey(instance.Use, i)
		}

		if !MiddlewareUseIsEnabled(instance.Use) {
			continue
		}
		if isNilDefinition(instance.Def) {
			return nil, fmt.Errorf("build middleware %s: definition is nil", instanceKey)
		}

		var cfg any
		if instance.Resolved != nil {
			cfg = copyAny(instance.Resolved.Config)
		}

		mw, err := instance.Def.Build(ctx, deps.Clone(), cfg)
		if err != nil {
			return nil, fmt.Errorf("build middleware %s: %w", instanceKey, err)
		}
		if mw == nil {
			return nil, fmt.Errorf("build middleware %s: definition returned nil middleware", instanceKey)
		}
		chain = append(chain, mw)
	}

	return chain, nil
}

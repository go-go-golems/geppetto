package middlewarecfg

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// DefinitionRegistry provides registration and lookup for middleware definitions.
type DefinitionRegistry interface {
	RegisterDefinition(def Definition) error
	GetDefinition(name string) (Definition, bool)
	ListDefinitions() []Definition
}

// InMemoryDefinitionRegistry stores middleware definitions in memory.
type InMemoryDefinitionRegistry struct {
	mu          sync.RWMutex
	definitions map[string]Definition
}

// NewInMemoryDefinitionRegistry returns an empty in-memory definition registry.
func NewInMemoryDefinitionRegistry() *InMemoryDefinitionRegistry {
	return &InMemoryDefinitionRegistry{
		definitions: map[string]Definition{},
	}
}

var _ DefinitionRegistry = (*InMemoryDefinitionRegistry)(nil)

// RegisterDefinition adds a middleware definition keyed by its declared name.
func (r *InMemoryDefinitionRegistry) RegisterDefinition(def Definition) error {
	if r == nil {
		return fmt.Errorf("definition registry is nil")
	}
	if isNilDefinition(def) {
		return fmt.Errorf("middleware definition is nil")
	}

	name := normalizeDefinitionName(def.Name())
	if name == "" {
		return fmt.Errorf("middleware definition name is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.definitions[name]; ok {
		return fmt.Errorf("middleware definition already registered: %s", name)
	}
	r.definitions[name] = def
	return nil
}

// GetDefinition returns a definition by middleware name.
func (r *InMemoryDefinitionRegistry) GetDefinition(name string) (Definition, bool) {
	if r == nil {
		return nil, false
	}
	key := normalizeDefinitionName(name)
	if key == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.definitions[key]
	return def, ok
}

// ListDefinitions returns all registered definitions sorted by middleware name.
func (r *InMemoryDefinitionRegistry) ListDefinitions() []Definition {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.definitions) == 0 {
		return nil
	}

	names := make([]string, 0, len(r.definitions))
	for name := range r.definitions {
		names = append(names, name)
	}
	sort.Strings(names)

	defs := make([]Definition, 0, len(names))
	for _, name := range names {
		defs = append(defs, r.definitions[name])
	}
	return defs
}

func normalizeDefinitionName(name string) string {
	return strings.TrimSpace(name)
}

func isNilDefinition(def Definition) bool {
	if def == nil {
		return true
	}
	v := reflect.ValueOf(def)
	kind := v.Kind()
	if kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Interface ||
		kind == reflect.Map ||
		kind == reflect.Pointer ||
		kind == reflect.Slice {
		return v.IsNil()
	}
	return false
}

package tools

import (
	"fmt"
	"sync"
)

// ToolRegistry manages available tools with thread-safe operations
type ToolRegistry interface {
	RegisterTool(name string, def ToolDefinition) error
	GetTool(name string) (*ToolDefinition, error)
	ListTools() []ToolDefinition
	UnregisterTool(name string) error

	// Thread-safe registry operations
	Clone() ToolRegistry
	Merge(other ToolRegistry) ToolRegistry
}

// InMemoryToolRegistry is a thread-safe in-memory implementation of ToolRegistry
type InMemoryToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolDefinition
}

// NewInMemoryToolRegistry creates a new in-memory tool registry
func NewInMemoryToolRegistry() *InMemoryToolRegistry {
	return &InMemoryToolRegistry{
		tools: make(map[string]ToolDefinition),
	}
}

// RegisterTool registers a new tool in the registry
func (r *InMemoryToolRegistry) RegisterTool(name string, def ToolDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if def.Name != "" && def.Name != name {
		return fmt.Errorf("tool definition name (%s) does not match registry name (%s)", def.Name, name)
	}

	// Ensure the definition has the correct name
	def.Name = name
	r.tools[name] = def
	return nil
}

// GetTool retrieves a tool by name
func (r *InMemoryToolRegistry) GetTool(name string) (*ToolDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Return a copy to prevent external modifications
	toolCopy := tool
	return &toolCopy, nil
}

// ListTools returns a list of all registered tools
func (r *InMemoryToolRegistry) ListTools() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// UnregisterTool removes a tool from the registry
func (r *InMemoryToolRegistry) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool not found: %s", name)
	}

	delete(r.tools, name)
	return nil
}

// Clone creates a deep copy of the registry
func (r *InMemoryToolRegistry) Clone() ToolRegistry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cloned := NewInMemoryToolRegistry()
	for name, tool := range r.tools {
		cloned.tools[name] = tool
	}

	return cloned
}

// Merge creates a new registry that contains tools from both registries
// If there are conflicts, tools from the other registry take precedence
func (r *InMemoryToolRegistry) Merge(other ToolRegistry) ToolRegistry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	merged := NewInMemoryToolRegistry()

	// Add tools from this registry first
	for name, tool := range r.tools {
		merged.tools[name] = tool
	}

	// Add tools from other registry (may overwrite)
	for _, tool := range other.ListTools() {
		merged.tools[tool.Name] = tool
	}

	return merged
}

// HasTool checks if a tool exists in the registry
func (r *InMemoryToolRegistry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[name]
	return exists
}

// Count returns the number of tools in the registry
func (r *InMemoryToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

package tools

// EngineFactory provides tool configuration for engines
type EngineFactory struct {
	defaultConfig  ToolConfig
	defaultRegistry ToolRegistry
}

// NewEngineFactory creates a new tool-enabled engine factory
func NewEngineFactory() *EngineFactory {
	return &EngineFactory{
		defaultConfig:   DefaultToolConfig(),
		defaultRegistry: NewInMemoryToolRegistry(),
	}
}

// WithToolRegistry sets the default tool registry for engines created by this factory
func (f *EngineFactory) WithToolRegistry(registry ToolRegistry) *EngineFactory {
	f.defaultRegistry = registry
	return f
}

// WithToolConfig sets the default tool configuration for engines created by this factory
func (f *EngineFactory) WithToolConfig(config ToolConfig) *EngineFactory {
	f.defaultConfig = config
	return f
}

// CreateOrchestrator creates a new inference orchestrator with the given engine
func (f *EngineFactory) CreateOrchestrator(baseEngine Engine, opts ...OrchestratorOption) *InferenceOrchestrator {
	config := &OrchestratorConfig{
		ToolRegistry: f.defaultRegistry,
		ToolConfig:   f.defaultConfig,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	return NewInferenceOrchestrator(baseEngine, config.ToolRegistry, config.ToolConfig)
}

// OrchestratorConfig holds configuration for creating orchestrators
type OrchestratorConfig struct {
	ToolRegistry ToolRegistry
	ToolConfig   ToolConfig
}

// OrchestratorOption is a function that modifies orchestrator configuration
type OrchestratorOption func(*OrchestratorConfig)

// WithOrchestratorRegistry sets the tool registry for the orchestrator
func WithOrchestratorRegistry(registry ToolRegistry) OrchestratorOption {
	return func(config *OrchestratorConfig) {
		config.ToolRegistry = registry
	}
}

// WithOrchestratorConfig sets the tool configuration for the orchestrator
func WithOrchestratorConfig(toolConfig ToolConfig) OrchestratorOption {
	return func(config *OrchestratorConfig) {
		config.ToolConfig = toolConfig
	}
}

// WithOrchestratorTools adds specific tools to the orchestrator's registry
func WithOrchestratorTools(tools map[string]ToolDefinition) OrchestratorOption {
	return func(config *OrchestratorConfig) {
		// Clone the registry to avoid modifying the original
		config.ToolRegistry = config.ToolRegistry.Clone()
		
		for name, tool := range tools {
			config.ToolRegistry.RegisterTool(name, tool)
		}
	}
}

// InferenceOrchestratorBuilder provides a fluent interface for building orchestrators
type InferenceOrchestratorBuilder struct {
	engine   Engine
	registry ToolRegistry
	config   ToolConfig
	tools    map[string]ToolDefinition
}

// NewOrchestratorBuilder creates a new orchestrator builder
func NewOrchestratorBuilder(baseEngine Engine) *InferenceOrchestratorBuilder {
	return &InferenceOrchestratorBuilder{
		engine:   baseEngine,
		registry: NewInMemoryToolRegistry(),
		config:   DefaultToolConfig(),
		tools:    make(map[string]ToolDefinition),
	}
}

// WithRegistry sets the tool registry
func (b *InferenceOrchestratorBuilder) WithRegistry(registry ToolRegistry) *InferenceOrchestratorBuilder {
	b.registry = registry
	return b
}

// WithConfig sets the tool configuration
func (b *InferenceOrchestratorBuilder) WithConfig(config ToolConfig) *InferenceOrchestratorBuilder {
	b.config = config
	return b
}

// WithTool adds a tool to the orchestrator
func (b *InferenceOrchestratorBuilder) WithTool(name string, tool ToolDefinition) *InferenceOrchestratorBuilder {
	b.tools[name] = tool
	return b
}

// WithToolFromFunc adds a tool created from a function
func (b *InferenceOrchestratorBuilder) WithToolFromFunc(name, description string, fn interface{}) *InferenceOrchestratorBuilder {
	tool, err := NewToolFromFunc(name, description, fn)
	if err != nil {
		// In a real implementation, you might want to handle this error differently
		panic(err)
	}
	b.tools[name] = *tool
	return b
}

// Build creates the final orchestrator
func (b *InferenceOrchestratorBuilder) Build() *InferenceOrchestrator {
	// Clone the registry to avoid modifying the original
	finalRegistry := b.registry.Clone()
	
	// Add tools to the registry
	for name, tool := range b.tools {
		finalRegistry.RegisterTool(name, tool)
	}
	
	return NewInferenceOrchestrator(b.engine, finalRegistry, b.config)
}

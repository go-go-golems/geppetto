# Complete Inference Engine Guide for New Developers

This comprehensive guide explains how the Geppetto inference engine system works, how engines are created, how to parse layers and set settings, and how to implement streaming inference with events. This guide is based on the existing codebase patterns and provides practical examples for new developers.

## Table of Contents

1. [Overview and Architecture](#overview-and-architecture)
2. [Core Components](#core-components)
3. [Engine Creation and Factory Pattern](#engine-creation-and-factory-pattern)
4. [Parameter Layers and Settings](#parameter-layers-and-settings)
5. [Conversation Management](#conversation-management)
6. [Streaming Inference with Events](#streaming-inference-with-events)
7. [Complete Implementation Examples](#complete-implementation-examples)
8. [Best Practices and Patterns](#best-practices-and-patterns)

## Overview and Architecture

The Geppetto inference engine system provides a unified interface for AI inference across multiple providers (OpenAI, Claude, Gemini, etc.) with support for both blocking and streaming execution modes. The system is built around several key architectural principles:

### Key Design Principles

1. **Provider Abstraction**: Engines abstract away provider-specific details
2. **Event-Driven Architecture**: Streaming inference uses events for real-time updates
3. **Layered Configuration**: Parameters are organized into logical layers
4. **Factory Pattern**: Engine creation is handled through factories
5. **Middleware Support**: Engines can be wrapped with middleware for logging, caching, etc.

### System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Application Layer                           │
├─────────────────────────────────────────────────────────────────┤
│  Commands (WriterCommand, GlazeCommand, BareCommand)          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Simple Inference│  │ Streaming       │  │ Pinocchio       │ │
│  │ Command         │  │ Inference       │  │ Command         │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                    Engine Layer                               │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Engine Factory  │  │ Engine          │  │ Engine Options  │ │
│  │ (StandardEngine │  │ Interface       │  │ (WithSink, etc.)│ │
│  │  Factory)       │  │ (RunInference)  │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                    Provider Layer                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ OpenAI Engine   │  │ Claude Engine   │  │ Gemini Engine   │ │
│  │ (OpenAI API)    │  │ (Anthropic API) │  │ (Google API)    │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                    Event System                               │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ EventSink       │  │ EventRouter     │  │ Event Handlers  │ │
│  │ Interface       │  │ (Watermill)     │  │ (Printers)      │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                    Configuration Layer                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Parameter       │  │ StepSettings    │  │ ParsedLayers    │ │
│  │ Layers          │  │ (API Keys, etc.)│  │ (Runtime Values)│ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Engine Interface

The core abstraction is the `Engine` interface, which defines how AI inference is performed:

**File**: `geppetto/pkg/inference/engine/engine.go`
```go
type Engine interface {
    // RunInference processes a conversation and returns an AI-generated message.
    // The engine handles both streaming and non-streaming modes based on configuration.
    // Events are published through all registered EventSinks during inference.
    RunInference(ctx context.Context, messages conversation.Conversation) (*conversation.Message, error)
}
```

### 2. Engine Factory

The factory pattern provides a unified way to create engines for different AI providers:

**File**: `geppetto/pkg/inference/engine/factory/factory.go`
```go
type EngineFactory interface {
    // CreateEngine creates an Engine instance based on the provided settings.
    // The actual provider is determined from settings.Chat.ApiType.
    CreateEngine(settings *settings.StepSettings, options ...engine.Option) (engine.Engine, error)
    
    // SupportedProviders returns a list of provider names this factory supports.
    SupportedProviders() []string
    
    // DefaultProvider returns the name of the default provider used when
    // settings.Chat.ApiType is nil or not specified.
    DefaultProvider() string
}
```

### 3. Event System

The event system enables streaming inference and real-time updates:

**File**: `geppetto/pkg/events/sink.go`
```go
type EventSink interface {
    // PublishEvent publishes an event to the sink.
    // Returns an error if the event could not be published.
    PublishEvent(event Event) error
}
```

## Engine Creation and Factory Pattern

### Standard Engine Factory

The `StandardEngineFactory` is the default implementation that supports multiple AI providers:

**File**: `geppetto/pkg/inference/engine/factory/factory.go`
```go
type StandardEngineFactory struct {
    // ClaudeTools are the tools to pass to Claude engines
    // This can be empty for basic text generation
    ClaudeTools []api.Tool
}

func NewStandardEngineFactory(claudeTools ...api.Tool) *StandardEngineFactory {
    return &StandardEngineFactory{
        ClaudeTools: claudeTools,
    }
}
```

### Engine Creation Process

The factory determines the appropriate engine based on the API type specified in settings:

```go
func (f *StandardEngineFactory) CreateEngine(settings *settings.StepSettings, options ...engine.Option) (engine.Engine, error) {
    if settings == nil {
        return nil, errors.New("settings cannot be nil")
    }

    // Determine provider from settings
    provider := f.DefaultProvider()
    if settings.Chat != nil && settings.Chat.ApiType != nil {
        provider = strings.ToLower(string(*settings.Chat.ApiType))
    }

    // Validate that we have the required settings
    if err := f.validateSettings(settings, provider); err != nil {
        return nil, errors.Wrapf(err, "invalid settings for provider %s", provider)
    }

    // Create engine based on provider
    switch provider {
    case string(types.ApiTypeOpenAI), string(types.ApiTypeAnyScale), string(types.ApiTypeFireworks):
        return openai.NewOpenAIEngine(settings, options...)

    case string(types.ApiTypeClaude), "anthropic":
        return claude.NewClaudeEngine(settings, f.ClaudeTools, options...)

    case string(types.ApiTypeGemini):
        // TODO: Implement GeminiEngine when available
        return nil, errors.Errorf("provider %s is not yet implemented", provider)

    default:
        supported := strings.Join(f.SupportedProviders(), ", ")
        return nil, errors.Errorf("unsupported provider %s. Supported providers: %s", provider, supported)
    }
}
```

### Supported Providers

The factory supports the following AI providers:

- **OpenAI**: GPT models via OpenAI API
- **AnyScale**: OpenAI-compatible models via AnyScale
- **Fireworks**: OpenAI-compatible models via Fireworks
- **Claude**: Claude models via Anthropic API
- **Gemini**: Gemini models via Google API (planned)

### Helper Functions

The factory provides convenient helper functions for common use cases:

**File**: `geppetto/pkg/inference/engine/factory/helpers.go`
```go
// NewEngineFromStepSettings creates an engine directly from step settings.
func NewEngineFromStepSettings(stepSettings *settings.StepSettings, options ...engine.Option) (engine.Engine, error) {
    factory := NewStandardEngineFactory()
    return factory.CreateEngine(stepSettings, options...)
}

// NewEngineFromParsedLayers creates an engine from parsed layers.
func NewEngineFromParsedLayers(parsedLayers *layers.ParsedLayers, options ...engine.Option) (engine.Engine, error) {
    // Create step settings
    stepSettings, err := settings.NewStepSettings()
    if err != nil {
        return nil, err
    }

    // Update step settings from parsed layers
    err = stepSettings.UpdateFromParsedLayers(parsedLayers)
    if err != nil {
        return nil, err
    }

    // Create engine using step settings
    return NewEngineFromStepSettings(stepSettings, options...)
}
```

## Parameter Layers and Settings

### Parameter Layer System

The parameter layer system organizes configuration into logical groups, making it easier to manage complex settings across different AI providers and use cases.

**File**: `geppetto/pkg/layers/layers.go`
```go
// CreateGeppettoLayers returns parameter layers for Geppetto AI settings.
func CreateGeppettoLayers(opts ...CreateOption) ([]cmdlayers.ParameterLayer, error) {
    // Apply options
    var co createOptions
    for _, opt := range opts {
        opt(&co)
    }
    
    // Determine StepSettings
    var ss *settings.StepSettings
    if co.stepSettings == nil {
        var err error
        ss, err = settings.NewStepSettings()
        if err != nil {
            return nil, err
        }
    } else {
        ss = co.stepSettings
    }

    // Create individual parameter layers
    chatParameterLayer, err := settings.NewChatParameterLayer(cmdlayers.WithDefaults(ss.Chat))
    if err != nil {
        return nil, err
    }

    clientParameterLayer, err := settings.NewClientParameterLayer(cmdlayers.WithDefaults(ss.Client))
    if err != nil {
        return nil, err
    }

    claudeParameterLayer, err := claude.NewParameterLayer(cmdlayers.WithDefaults(ss.Claude))
    if err != nil {
        return nil, err
    }

    geminiParameterLayer, err := gemini.NewParameterLayer(cmdlayers.WithDefaults(ss.Gemini))
    if err != nil {
        return nil, err
    }

    openaiParameterLayer, err := openai.NewParameterLayer(cmdlayers.WithDefaults(ss.OpenAI))
    if err != nil {
        return nil, err
    }

    embeddingsParameterLayer, err := embeddingsconfig.NewEmbeddingsParameterLayer(cmdlayers.WithDefaults(ss.Embeddings))
    if err != nil {
        return nil, err
    }

    // Assemble layers
    result := []cmdlayers.ParameterLayer{
        chatParameterLayer,
        clientParameterLayer,
        claudeParameterLayer,
        geminiParameterLayer,
        openaiParameterLayer,
        embeddingsParameterLayer,
    }
    return result, nil
}
```

### Step Settings Structure

The `StepSettings` struct provides a unified configuration interface for all AI providers:

**File**: `geppetto/pkg/steps/ai/settings/settings-step.go`
```go
type StepSettings struct {
    API    *APISettings     `yaml:"api_keys,omitempty"`
    Chat   *ChatSettings    `yaml:"chat,omitempty" glazed.layer:"ai-chat"`
    OpenAI *openai.Settings `yaml:"openai,omitempty" glazed.layer:"openai-chat"`
    Client *ClientSettings  `yaml:"client,omitempty" glazed.layer:"ai-client"`
    Claude *claude.Settings `yaml:"claude,omitempty" glazed.layer:"claude-chat"`
    Gemini *gemini.Settings `yaml:"gemini,omitempty" glazed.layer:"gemini-chat"`
    Ollama *ollama.Settings `yaml:"ollama,omitempty" glazed.layer:"ollama-chat"`
    Embeddings *config.EmbeddingsConfig `yaml:"embeddings,omitempty" glazed.layer:"embeddings"`
}
```

### Chat Settings

The `ChatSettings` struct defines common chat parameters:

**File**: `geppetto/pkg/steps/ai/settings/settings-chat.go`
```go
type ChatSettings struct {
    Engine            *string           `yaml:"engine,omitempty" glazed.parameter:"ai-engine"`
    ApiType           *types.ApiType    `yaml:"api_type,omitempty" glazed.parameter:"ai-api-type"`
    MaxResponseTokens *int              `yaml:"max_response_tokens,omitempty" glazed.parameter:"ai-max-response-tokens"`
    TopP              *float64          `yaml:"top_p,omitempty" glazed.parameter:"ai-top-p"`
    Temperature       *float64          `yaml:"temperature,omitempty" glazed.parameter:"ai-temperature"`
    Stop              []string          `yaml:"stop,omitempty" glazed.parameter:"ai-stop"`
    Stream            bool              `yaml:"stream,omitempty"`
    APIKeys           map[string]string `yaml:"api_keys,omitempty" glazed.parameter:"*-api-key"`

    // Caching settings
    CacheType       string `yaml:"cache_type,omitempty" glazed.parameter:"ai-cache-type"`
    CacheMaxSize    int64  `yaml:"cache_max_size,omitempty" glazed.parameter:"ai-cache-max-size"`
    CacheMaxEntries int    `yaml:"cache_max_entries,omitempty" glazed.parameter:"ai-cache-max-entries"`
    CacheDirectory  string `yaml:"cache_directory,omitempty" glazed.parameter:"ai-cache-directory"`
}
```

### Creating Step Settings

Step settings can be created in several ways:

```go
// Create default settings
stepSettings, err := settings.NewStepSettings()
if err != nil {
    return err
}

// Create from YAML file
file, err := os.Open("config.yaml")
if err != nil {
    return err
}
defer file.Close()

stepSettings, err = settings.NewStepSettingsFromYAML(file)
if err != nil {
    return err
}

// Create from parsed layers
stepSettings, err = settings.NewStepSettingsFromParsedLayers(parsedLayers)
if err != nil {
    return err
}
```

### Updating Settings from Parsed Layers

Settings can be updated from parsed layers to reflect command-line arguments, environment variables, and configuration files:

```go
func (ss *StepSettings) UpdateFromParsedLayers(parsedLayers *layers.ParsedLayers) error {
    // Update each section from the corresponding layer
    if err := ss.Chat.UpdateFromParsedLayers(parsedLayers); err != nil {
        return err
    }
    
    if err := ss.OpenAI.UpdateFromParsedLayers(parsedLayers); err != nil {
        return err
    }
    
    if err := ss.Claude.UpdateFromParsedLayers(parsedLayers); err != nil {
        return err
    }
    
    // ... update other sections
    
    return nil
}
```

## Conversation Management

### Conversation Manager

The conversation manager handles the construction and management of AI conversations:

**File**: `geppetto/pkg/conversation/manager.go`
```go
type Manager interface {
    GetConversation() Conversation
    GetMessage(ID NodeID) (*Message, bool)
    AppendMessages(messages ...*Message) error
    // ... other methods
}
```

### Manager Builder

The `ManagerBuilder` provides a fluent interface for constructing conversation managers:

**File**: `geppetto/pkg/conversation/builder/builder.go`
```go
type ManagerBuilder struct {
    systemPrompt string
    messages     []*conversation.Message
    prompt       string
    variables    map[string]interface{}
    images       []string

    autosaveEnabled  bool
    autosaveTemplate string
    autosavePath     string
}

func NewManagerBuilder() *ManagerBuilder {
    return &ManagerBuilder{
        variables: make(map[string]interface{}),
    }
}

func (b *ManagerBuilder) WithSystemPrompt(systemPrompt string) *ManagerBuilder {
    b.systemPrompt = systemPrompt
    return b
}

func (b *ManagerBuilder) WithPrompt(prompt string) *ManagerBuilder {
    b.prompt = prompt
    return b
}

func (b *ManagerBuilder) WithVariables(variables map[string]interface{}) *ManagerBuilder {
    if b.variables == nil {
        b.variables = make(map[string]interface{})
    }
    for k, v := range variables {
        b.variables[k] = v
    }
    return b
}

func (b *ManagerBuilder) WithImages(images []string) *ManagerBuilder {
    b.images = images
    return b
}
```

### Building a Conversation Manager

```go
b := builder.NewManagerBuilder().
    WithSystemPrompt("You are a helpful assistant. Answer the question in a short and concise manner.").
    WithPrompt(s.Prompt)

manager, err := b.Build()
if err != nil {
    log.Error().Err(err).Msg("Failed to build conversation manager")
    return err
}

conversation_ := manager.GetConversation()
```

### Conversation Initialization

The builder automatically initializes the conversation with system prompts, messages, and user prompts:

```go
func (b *ManagerBuilder) initializeConversation(manager conversation.Manager) error {
    // Add system prompt if provided
    if b.systemPrompt != "" {
        systemPromptTemplate, err := templating.CreateTemplate("system-prompt").Parse(b.systemPrompt)
        if err != nil {
            return errors.Wrap(err, "failed to parse system prompt template")
        }

        var systemPromptBuffer strings.Builder
        err = systemPromptTemplate.Execute(&systemPromptBuffer, b.variables)
        if err != nil {
            return errors.Wrap(err, "failed to execute system prompt template")
        }

        if err := manager.AppendMessages(conversation.NewChatMessage(
            conversation.RoleSystem,
            systemPromptBuffer.String(),
        )); err != nil {
            return errors.Wrap(err, "failed to append system prompt message")
        }
    }

    // Add existing messages
    for _, message_ := range b.messages {
        // ... process and add messages
    }

    // Add user prompt
    if b.prompt != "" {
        promptTemplate, err := templating.CreateTemplate("prompt").Parse(b.prompt)
        if err != nil {
            return errors.Wrap(err, "failed to parse prompt template")
        }

        var promptBuffer strings.Builder
        err = promptTemplate.Execute(&promptBuffer, b.variables)
        if err != nil {
            return errors.Wrap(err, "failed to execute prompt template")
        }

        // Handle images if provided
        images := []*conversation.ImageContent{}
        for _, imagePath := range b.images {
            image, err := conversation.NewImageContentFromFile(imagePath)
            if err != nil {
                return errors.Wrap(err, "failed to create image content")
            }
            images = append(images, image)
        }

        messageContent := &conversation.ChatMessageContent{
            Role:   conversation.RoleUser,
            Text:   promptBuffer.String(),
            Images: images,
        }
        if err := manager.AppendMessages(conversation.NewMessage(messageContent)); err != nil {
            return errors.Wrap(err, "failed to append prompt message with images")
        }
    }

    return nil
}
```

## Streaming Inference with Events

### Event System Overview

The event system enables real-time streaming of inference results. It consists of several key components:

1. **EventSink Interface**: Defines how events are published
2. **WatermillSink**: Implements event publishing via Watermill message bus
3. **EventRouter**: Manages event routing and distribution
4. **Event Handlers**: Process and display events

### EventSink Interface

**File**: `geppetto/pkg/events/sink.go`
```go
type EventSink interface {
    // PublishEvent publishes an event to the sink.
    // Returns an error if the event could not be published.
    PublishEvent(event Event) error
}
```

### WatermillSink Implementation

**File**: `geppetto/pkg/inference/middleware/sink_watermill.go`
```go
type WatermillSink struct {
    publisher message.Publisher
    topic     string
}

func NewWatermillSink(publisher message.Publisher, topic string) *WatermillSink {
    return &WatermillSink{
        publisher: publisher,
        topic:     topic,
    }
}

func (w *WatermillSink) PublishEvent(event events.Event) error {
    // Serialize the event to JSON
    payload, err := json.Marshal(event)
    if err != nil {
        log.Error().Err(err).Msg("Failed to marshal event to JSON")
        return err
    }

    // Create watermill message
    msg := message.NewMessage(watermill.NewUUID(), payload)

    // Publish to the topic
    err = w.publisher.Publish(w.topic, msg)
    if err != nil {
        log.Error().Err(err).Str("topic", w.topic).Msg("Failed to publish event to watermill")
        return err
    }

    log.Trace().Str("topic", w.topic).Str("event_type", string(event.Type())).Msg("Published event to watermill")
    return nil
}
```

### EventRouter

**File**: `geppetto/pkg/events/event-router.go`
```go
type EventRouter struct {
    logger     watermill.LoggerAdapter
    Publisher  message.Publisher
    Subscriber message.Subscriber
    router     *message.Router
    verbose    bool
}

func NewEventRouter(options ...EventRouterOption) (*EventRouter, error) {
    ret := &EventRouter{
        logger: watermill.NopLogger{},
    }

    for _, o := range options {
        o(ret)
    }

    goPubSub := gochannel.NewGoChannel(gochannel.Config{
        BlockPublishUntilSubscriberAck: true,
    }, ret.logger)
    ret.Publisher = goPubSub
    ret.Subscriber = goPubSub

    router, err := message.NewRouter(message.RouterConfig{}, ret.logger)
    if err != nil {
        return nil, err
    }

    ret.router = router
    return ret, nil
}

func (e *EventRouter) AddHandler(name string, topic string, f func(msg *message.Message) error) {
    e.router.AddNoPublishHandler(name, topic, e.Subscriber, f)
}

func (e *EventRouter) Run(ctx context.Context) error {
    return e.router.Run(ctx)
}

func (e *EventRouter) Close() error {
    log.Debug().Msg("Closing publisher")
    err := e.Publisher.Close()
    if err != nil {
        log.Error().Err(err).Msg("Failed to close pubsub")
    }
    log.Debug().Msg("Publisher closed")
    return err
}
```

### Engine Options

**File**: `geppetto/pkg/inference/engine/options.go`
```go
type Option func(*Config) error

type Config struct {
    EventSinks []events.EventSink
}

func WithSink(sink events.EventSink) Option {
    return func(config *Config) error {
        config.EventSinks = append(config.EventSinks, sink)
        return nil
    }
}

func ApplyOptions(config *Config, options ...Option) error {
    for _, option := range options {
        if err := option(config); err != nil {
            return err
        }
    }
    return nil
}
```

### Event Printer Functions

**File**: `geppetto/pkg/events/step-printer-func.go`
```go
func StepPrinterFunc(name string, w io.Writer) func(msg *message.Message) error {
    return func(msg *message.Message) error {
        var event Event
        if err := json.Unmarshal(msg.Payload, &event); err != nil {
            return err
        }

        switch e := event.(type) {
        case *EventPartialCompletion:
            fmt.Fprint(w, e.Delta)
        case *EventFinal:
            fmt.Fprintln(w)
        case *EventToolCall:
            // Handle tool calls
        case *EventToolResult:
            // Handle tool results
        }
        return nil
    }
}
```

**File**: `geppetto/pkg/events/printer.go`
```go
type PrinterFormat string
const (
    FormatText PrinterFormat = "text"
    FormatJSON PrinterFormat = "json"
    FormatYAML PrinterFormat = "yaml"
)

type PrinterOptions struct {
    Format          PrinterFormat
    Name            string
    IncludeMetadata bool
    Full            bool
}

func NewStructuredPrinter(w io.Writer, options PrinterOptions) func(msg *message.Message) error {
    return func(msg *message.Message) error {
        var event Event
        if err := json.Unmarshal(msg.Payload, &event); err != nil {
            return err
        }

        switch options.Format {
        case FormatJSON:
            return json.NewEncoder(w).Encode(event)
        case FormatYAML:
            return yaml.NewEncoder(w).Encode(event)
        default:
            return fmt.Fprintf(w, "%s\n", event.String())
        }
    }
}
```

## Complete Implementation Examples

### Simple Inference (Non-Streaming)

**File**: `geppetto/cmd/examples/simple-inference/main.go`

```go
type SimpleInferenceCommand struct {
    *cmds.CommandDescription
}

type SimpleInferenceSettings struct {
    PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
    Debug            bool   `glazed.parameter:"debug"`
    WithLogging      bool   `glazed.parameter:"with-logging"`
    Prompt           string `glazed.parameter:"prompt"`
}

func (c *SimpleInferenceCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
    log.Info().Msg("Starting simple inference command")

    s := &SimpleInferenceSettings{}
    err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
    if err != nil {
        return errors.Wrap(err, "failed to initialize settings")
    }

    if s.Debug {
        b, err := yaml.Marshal(parsedLayers)
        if err != nil {
            return err
        }
        fmt.Fprintln(w, "=== Parsed Layers Debug ===")
        fmt.Fprintln(w, string(b))
        fmt.Fprintln(w, "=========================")
        return nil
    }

    // Create engine
    engine, err := factory.NewEngineFromParsedLayers(parsedLayers)
    if err != nil {
        log.Error().Err(err).Msg("Failed to create engine")
        return errors.Wrap(err, "failed to create engine")
    }

    // Add logging middleware if requested
    if s.WithLogging {
        loggingMiddleware := func(next middleware.HandlerFunc) middleware.HandlerFunc {
            return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
                logger := log.With().Int("message_count", len(messages)).Logger()
                logger.Info().Msg("Starting inference")

                result, err := next(ctx, messages)
                if err != nil {
                    logger.Error().Err(err).Msg("Inference failed")
                } else {
                    logger.Info().Int("result_message_count", len(result)).Msg("Inference completed")
                }
                return result, err
            }
        }
        engine = middleware.NewEngineWithMiddleware(engine, loggingMiddleware)
    }

    // Build conversation manager
    b := builder.NewManagerBuilder().
        WithSystemPrompt("You are a helpful assistant. Answer the question in a short and concise manner.").
        WithPrompt(s.Prompt)

    manager, err := b.Build()
    if err != nil {
        log.Error().Err(err).Msg("Failed to build conversation manager")
        return err
    }

    conversation_ := manager.GetConversation()

    // Run inference
    msg, err := engine.RunInference(ctx, conversation_)
    if err != nil {
        log.Error().Err(err).Msg("Inference failed")
        return fmt.Errorf("inference failed: %w", err)
    }

    if err := manager.AppendMessages(msg); err != nil {
        log.Error().Err(err).Msg("Failed to append message to conversation")
        return fmt.Errorf("failed to append message: %w", err)
    }

    messages := manager.GetConversation()

    fmt.Fprintln(w, "\n=== Final Conversation ===")
    for _, msg := range messages {
        if chatMsg, ok := msg.Content.(*conversation.ChatMessageContent); ok {
            fmt.Fprintf(w, "%s: %s\n", chatMsg.Role, chatMsg.Text)
        } else {
            fmt.Fprintf(w, "%s: %s\n", msg.Content.ContentType(), msg.Content.String())
        }
    }

    log.Info().Int("total_messages", len(messages)).Msg("Simple inference command completed successfully")
    return nil
}
```

### Streaming Inference

**File**: `geppetto/cmd/examples/simple-streaming-inference/main.go`

```go
type SimpleStreamingInferenceCommand struct {
    *cmds.CommandDescription
}

type SimpleStreamingInferenceSettings struct {
    PinocchioProfile string `glazed.parameter:"pinocchio-profile"`
    Debug            bool   `glazed.parameter:"debug"`
    WithLogging      bool   `glazed.parameter:"with-logging"`
    Prompt           string `glazed.parameter:"prompt"`
    OutputFormat     string `glazed.parameter:"output-format"`
    WithMetadata     bool   `glazed.parameter:"with-metadata"`
    FullOutput       bool   `glazed.parameter:"full-output"`
    Verbose          bool   `glazed.parameter:"verbose"`
}

func (c *SimpleStreamingInferenceCommand) RunIntoWriter(ctx context.Context, parsedLayers *layers.ParsedLayers, w io.Writer) error {
    log.Info().Msg("Starting simple streaming inference command")

    s := &SimpleStreamingInferenceSettings{}
    err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
    if err != nil {
        return errors.Wrap(err, "failed to initialize settings")
    }

    if s.Debug {
        b, err := yaml.Marshal(parsedLayers)
        if err != nil {
            return err
        }
        fmt.Fprintln(w, "=== Parsed Layers Debug ===")
        fmt.Fprintln(w, string(b))
        fmt.Fprintln(w, "=========================")
        return nil
    }

    // 1. Create event router
    routerOptions := []events.EventRouterOption{}
    if s.Verbose {
        routerOptions = append(routerOptions, events.WithVerbose(true))
    }
    
    router, err := events.NewEventRouter(routerOptions...)
    if err != nil {
        return errors.Wrap(err, "failed to create event router")
    }
    defer func() {
        if router != nil {
            _ = router.Close()
        }
    }()

    // 2. Create watermill sink
    watermillSink := middleware.NewWatermillSink(router.Publisher, "chat")
    
    // 3. Add printer handler based on output format
    if s.OutputFormat == "" {
        router.AddHandler("chat", "chat", events.StepPrinterFunc("", w))
    } else {
        printer := events.NewStructuredPrinter(w, events.PrinterOptions{
            Format:          events.PrinterFormat(s.OutputFormat),
            Name:            "",
            IncludeMetadata: s.WithMetadata,
            Full:            s.FullOutput,
        })
        router.AddHandler("chat", "chat", printer)
    }

    // 4. Create engine with sink
    engineOptions := []engine.Option{
        engine.WithSink(watermillSink),
    }
    
    engine, err := factory.NewEngineFromParsedLayers(parsedLayers, engineOptions...)
    if err != nil {
        log.Error().Err(err).Msg("Failed to create engine")
        return errors.Wrap(err, "failed to create engine")
    }

    // Add logging middleware if requested
    if s.WithLogging {
        loggingMiddleware := func(next middleware.HandlerFunc) middleware.HandlerFunc {
            return func(ctx context.Context, messages conversation.Conversation) (conversation.Conversation, error) {
                logger := log.With().Int("message_count", len(messages)).Logger()
                logger.Info().Msg("Starting inference")

                result, err := next(ctx, messages)
                if err != nil {
                    logger.Error().Err(err).Msg("Inference failed")
                } else {
                    logger.Info().Int("result_message_count", len(result)).Msg("Inference completed")
                }
                return result, err
            }
        }
        engine = middleware.NewEngineWithMiddleware(engine, loggingMiddleware)
    }

    // Build conversation manager
    b := builder.NewManagerBuilder().
        WithSystemPrompt("You are a helpful assistant. Answer the question in a short and concise manner.").
        WithPrompt(s.Prompt)

    manager, err := b.Build()
    if err != nil {
        log.Error().Err(err).Msg("Failed to build conversation manager")
        return err
    }

    conversation_ := manager.GetConversation()

    // 5. Start router and run inference in parallel
    eg := errgroup.Group{}
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    eg.Go(func() error {
        defer cancel()
        return router.Run(ctx)
    })

    eg.Go(func() error {
        defer cancel()
        <-router.Running()
        
        // Run inference
        msg, err := engine.RunInference(ctx, conversation_)
        if err != nil {
            log.Error().Err(err).Msg("Inference failed")
            return fmt.Errorf("inference failed: %w", err)
        }
        
        // Append result
        if err := manager.AppendMessages(msg); err != nil {
            log.Error().Err(err).Msg("Failed to append message to conversation")
            return fmt.Errorf("failed to append message: %w", err)
        }
        
        return nil
    })

    err = eg.Wait()
    if err != nil {
        return err
    }

    messages := manager.GetConversation()

    fmt.Fprintln(w, "\n=== Final Conversation ===")
    for _, msg := range messages {
        if chatMsg, ok := msg.Content.(*conversation.ChatMessageContent); ok {
            fmt.Fprintf(w, "%s: %s\n", chatMsg.Role, chatMsg.Text)
        } else {
            fmt.Fprintf(w, "%s: %s\n", msg.Content.ContentType(), msg.Content.String())
        }
    }

    log.Info().Int("total_messages", len(messages)).Msg("Simple streaming inference command completed successfully")
    return nil
}
```

## Best Practices and Patterns

### 1. Error Handling

Always provide specific, actionable error messages:

```go
// Good: Specific and actionable
if s.Port < 1 || s.Port > 65535 {
    return fmt.Errorf("port %d is invalid; must be between 1 and 65535", s.Port)
}

// Poor: Vague and frustrating
if !isValidPort(s.Port) {
    return errors.New("invalid port")
}
```

### 2. Resource Cleanup

Always ensure proper cleanup of resources, especially for streaming inference:

```go
defer func() {
    if router != nil {
        _ = router.Close()
    }
}()
```

### 3. Context Cancellation

Use context cancellation for coordinated shutdown:

```go
ctx, cancel := context.WithCancel(ctx)
defer cancel()

eg := errgroup.Group{}
eg.Go(func() error {
    defer cancel()
    return router.Run(ctx)
})

eg.Go(func() error {
    defer cancel()
    <-router.Running()
    return runInference(ctx)
})

return eg.Wait()
```

### 4. Settings Validation

Validate settings before expensive operations:

```go
func (c *MyCommand) validateSettings(s *MyCommandSettings) error {
    if s.Count < 0 {
        return errors.New("count must be non-negative")
    }
    if s.Count > 1000 {
        return errors.New("count cannot exceed 1000")
    }
    return nil
}
```

### 5. Performance Considerations

Design for streaming to handle large datasets:

```go
// Good: Processes data as it arrives
scanner := bufio.NewScanner(reader)
for scanner.Scan() {
    row := processLine(scanner.Text())
    if err := gp.AddRow(ctx, row); err != nil {
        return err
    }
}

// Problematic: Loads everything into memory first
allData := loadAllDataIntoMemory() // What if this is 10GB?
for _, item := range allData {
    // Process items...
}
```

### 6. Type Safety

Use settings structs with `glazed.parameter` tags:

```go
// Good: Type-safe and clear
type BackupSettings struct {
    Source      string        `glazed.parameter:"source"`
    Destination string        `glazed.parameter:"destination"`
    MaxAge      time.Duration `glazed.parameter:"max-age"`
    DryRun      bool          `glazed.parameter:"dry-run"`
}

// Avoid: Manual parameter extraction
source, _ := parsedLayers.GetString("source")
maxAge, _ := parsedLayers.GetString("max-age") // Bug waiting to happen!
```

### 7. Documentation

Write clear help text with examples:

```go
parameters.NewParameterDefinition(
    "filter",
    parameters.ParameterTypeString,
    parameters.WithHelp("Filter results using SQL-like syntax. Examples: 'status = \"active\"', 'created_at > \"2023-01-01\"'"),
)
```

## Key Files and APIs

### Core Engine Files

- `geppetto/pkg/inference/engine/engine.go` - Engine interface
- `geppetto/pkg/inference/engine/factory/factory.go` - Engine factory
- `geppetto/pkg/inference/engine/factory/helpers.go` - Helper functions
- `geppetto/pkg/inference/engine/options.go` - Engine options

### Settings and Layers

- `geppetto/pkg/steps/ai/settings/settings-step.go` - Step settings structure
- `geppetto/pkg/steps/ai/settings/settings-chat.go` - Chat settings
- `geppetto/pkg/layers/layers.go` - Parameter layer creation

### Conversation Management

- `geppetto/pkg/conversation/manager.go` - Manager interface
- `geppetto/pkg/conversation/manager-impl.go` - Manager implementation
- `geppetto/pkg/conversation/builder/builder.go` - Manager builder

### Event System

- `geppetto/pkg/events/sink.go` - EventSink interface
- `geppetto/pkg/events/event-router.go` - EventRouter
- `geppetto/pkg/events/step-printer-func.go` - Step printer
- `geppetto/pkg/events/printer.go` - Structured printer
- `geppetto/pkg/inference/middleware/sink_watermill.go` - WatermillSink

### Example Implementations

- `geppetto/cmd/examples/simple-inference/main.go` - Simple inference
- `geppetto/cmd/examples/simple-streaming-inference/main.go` - Streaming inference

### Middleware

- `geppetto/pkg/inference/middleware/middleware.go` - Middleware system

## Next Steps

1. **Start with Simple Inference**: Begin with the non-streaming example to understand the basics
2. **Add Streaming**: Progress to streaming inference for real-time updates
3. **Customize Settings**: Learn to configure different AI providers and parameters
4. **Add Middleware**: Implement custom middleware for logging, caching, etc.
5. **Build Custom Commands**: Create your own commands using the patterns shown

This guide provides a comprehensive foundation for working with the Geppetto inference engine system. The modular design allows you to start simple and gradually add complexity as needed. 
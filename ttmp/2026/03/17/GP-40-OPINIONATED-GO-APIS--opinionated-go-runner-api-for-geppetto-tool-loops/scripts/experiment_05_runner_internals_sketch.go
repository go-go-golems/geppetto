//go:build ignore

// Experiment 05: Runner Internals — Implementation Sketch
//
// This shows how the runner package would be implemented internally,
// wrapping geppetto's existing types. This is the "how it works under
// the hood" companion to the "how you use it" experiments above.
//
// This file is a design sketch and is excluded from normal builds.

package runner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop"
	"github.com/go-go-golems/geppetto/pkg/inference/toolloop/enginebuilder"
	"github.com/go-go-golems/geppetto/pkg/profiles"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	aitypes "github.com/go-go-golems/geppetto/pkg/steps/ai/types"
	"github.com/go-go-golems/geppetto/pkg/turns"
)

// --- Types ---

type Result struct {
	Text       string
	ToolCalls  []ToolCallRecord
	Turn       *turns.Turn
	Usage      *turns.InferenceUsage
	StopReason string
}

type ToolCallRecord struct {
	Name   string
	Args   map[string]any
	Result string
	Error  error
}

type Runner struct {
	config  config
	session *session.Session
}

type config struct {
	model             string
	provider          string
	systemPrompt      string
	tools             []toolDef
	middlewares        []middleware.Middleware
	eventSinks        []events.EventSink
	onDelta           func(string)
	maxTokens         int
	temperature       float64
	maxToolIterations int
	toolTimeout       time.Duration
	engine            engine.Engine // escape hatch
	stepSettings      *aisettings.StepSettings
	profileName       string
	profileRegistries []string
}

type toolDef struct {
	name        string
	description string
	fn          any
}

type Option func(*config)

// --- Option functions ---

func System(prompt string) Option      { return func(c *config) { c.systemPrompt = prompt } }
func Model(name string) Option         { return func(c *config) { c.model = name } }
func MaxTokens(n int) Option           { return func(c *config) { c.maxTokens = n } }
func Temperature(t float64) Option     { return func(c *config) { c.temperature = t } }
func MaxTools(n int) Option            { return func(c *config) { c.maxToolIterations = n } }
func Timeout(d time.Duration) Option   { return func(c *config) { c.toolTimeout = d } }
func Sink(s events.EventSink) Option   { return func(c *config) { c.eventSinks = append(c.eventSinks, s) } }
func WithEngine(e engine.Engine) Option { return func(c *config) { c.engine = e } }

func Tool(name, description string, fn any) Option {
	return func(c *config) {
		c.tools = append(c.tools, toolDef{name: name, description: description, fn: fn})
	}
}

func Stream(fn func(string)) Option {
	return func(c *config) { c.onDelta = fn }
}

func Middleware(mw ...middleware.Middleware) Option {
	return func(c *config) { c.middlewares = append(c.middlewares, mw...) }
}

func Profile(name string, registries ...string) Option {
	return func(c *config) {
		c.profileName = name
		c.profileRegistries = registries
	}
}

// --- Core implementation ---

func Run(ctx context.Context, prompt string, opts ...Option) (*Result, error) {
	cfg := applyDefaults(opts)

	eng, err := resolveEngine(cfg)
	if err != nil {
		return nil, fmt.Errorf("engine setup: %w", err)
	}

	registry, err := buildToolRegistry(cfg)
	if err != nil {
		return nil, fmt.Errorf("tool registry: %w", err)
	}

	sinks := cfg.eventSinks
	if cfg.onDelta != nil {
		sinks = append(sinks, newDeltaSink(cfg.onDelta))
	}

	// Build the seed turn
	tb := turns.NewTurnBuilder()
	if cfg.systemPrompt != "" {
		tb.AddSystemBlock(cfg.systemPrompt)
	}
	tb.AddUserBlock(prompt)
	turn, err := tb.Build()
	if err != nil {
		return nil, fmt.Errorf("turn construction: %w", err)
	}

	// Create session and wire the builder
	sess := session.NewSession()
	sess.Builder = &enginebuilder.Builder{
		Base:       eng,
		Middlewares: cfg.middlewares,
		Registry:   registry,
		LoopConfig: toolloop.NewLoopConfig().WithMaxIterations(cfg.maxToolIterations),
		ToolConfig: tools.DefaultToolConfig().WithExecutionTimeout(cfg.toolTimeout),
		EventSinks: sinks,
	}
	sess.Append(turn)

	// Run inference
	handle, err := sess.StartInference(ctx)
	if err != nil {
		return nil, fmt.Errorf("start inference: %w", err)
	}

	resultTurn, err := handle.Wait()
	if err != nil {
		return nil, fmt.Errorf("inference: %w", err)
	}

	return buildResult(resultTurn), nil
}

func New(opts ...Option) *Runner {
	cfg := applyDefaults(opts)
	return &Runner{config: cfg}
}

func (r *Runner) Run(ctx context.Context, prompt string, extraOpts ...Option) (*Result, error) {
	// Merge extra options with runner config
	merged := r.config // copy
	for _, opt := range extraOpts {
		opt(&merged)
	}

	// For stateless Run(), create a fresh session each time
	return Run(ctx, prompt, func(c *config) { *c = merged })
}

func (r *Runner) Chat(ctx context.Context, prompt string, extraOpts ...Option) (*Result, error) {
	// For Chat(), maintain session across calls
	if r.session == nil {
		eng, err := resolveEngine(r.config)
		if err != nil {
			return nil, err
		}
		registry, err := buildToolRegistry(r.config)
		if err != nil {
			return nil, err
		}
		sinks := r.config.eventSinks
		if r.config.onDelta != nil {
			sinks = append(sinks, newDeltaSink(r.config.onDelta))
		}

		r.session = session.NewSession()
		r.session.Builder = &enginebuilder.Builder{
			Base:       eng,
			Middlewares: r.config.middlewares,
			Registry:   registry,
			LoopConfig: toolloop.NewLoopConfig().WithMaxIterations(r.config.maxToolIterations),
			ToolConfig: tools.DefaultToolConfig().WithExecutionTimeout(r.config.toolTimeout),
			EventSinks: sinks,
		}

		// Create initial turn with system prompt
		tb := turns.NewTurnBuilder()
		if r.config.systemPrompt != "" {
			tb.AddSystemBlock(r.config.systemPrompt)
		}
		tb.AddUserBlock(prompt)
		turn, _ := tb.Build()
		r.session.Append(turn)
	} else {
		// Append user message to existing conversation
		r.session.AppendNewTurnFromUserPrompt(prompt)
	}

	handle, err := r.session.StartInference(ctx)
	if err != nil {
		return nil, err
	}
	resultTurn, err := handle.Wait()
	if err != nil {
		return nil, err
	}
	return buildResult(resultTurn), nil
}

// Session returns the underlying geppetto Session for advanced use.
func (r *Runner) Session() *session.Session { return r.session }

// --- Internal helpers ---

func applyDefaults(opts []Option) config {
	cfg := config{
		maxTokens:         4096,
		temperature:       0,
		maxToolIterations: 10,
		toolTimeout:       30 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

func resolveEngine(cfg config) (engine.Engine, error) {
	// Escape hatch: user provided their own engine
	if cfg.engine != nil {
		return cfg.engine, nil
	}

	// Profile-based resolution
	if cfg.profileName != "" {
		return resolveEngineFromProfile(cfg)
	}

	// Auto-detection from environment
	return resolveEngineFromEnv(cfg)
}

func resolveEngineFromProfile(cfg config) (engine.Engine, error) {
	// Open profile registry and resolve settings
	registry, err := profiles.OpenChainedRegistry(cfg.profileRegistries)
	if err != nil {
		return nil, fmt.Errorf("open profile registry: %w", err)
	}
	defer registry.Close()

	profile, err := registry.GetProfile(cfg.profileName)
	if err != nil {
		return nil, fmt.Errorf("get profile %q: %w", cfg.profileName, err)
	}

	stepSettings := profile.StepSettings()
	applyConfigOverrides(stepSettings, cfg)
	return factory.NewEngineFromStepSettings(stepSettings)
}

func resolveEngineFromEnv(cfg config) (engine.Engine, error) {
	stepSettings := aisettings.NewStepSettings()

	// Auto-detect provider from environment
	switch {
	case os.Getenv("ANTHROPIC_API_KEY") != "":
		apiType := aitypes.ApiTypeClaude
		stepSettings.Chat.ApiType = &apiType
		stepSettings.Chat.APIKeys = map[string]string{"claude": os.Getenv("ANTHROPIC_API_KEY")}
		if cfg.model == "" {
			cfg.model = "claude-sonnet-4-20250514"
		}
	case os.Getenv("OPENAI_API_KEY") != "":
		apiType := aitypes.ApiTypeOpenAI
		stepSettings.Chat.ApiType = &apiType
		stepSettings.Chat.APIKeys = map[string]string{"openai": os.Getenv("OPENAI_API_KEY")}
		if cfg.model == "" {
			cfg.model = "gpt-4o"
		}
	default:
		return nil, fmt.Errorf("no API key found. Set ANTHROPIC_API_KEY or OPENAI_API_KEY")
	}

	applyConfigOverrides(stepSettings, cfg)
	return factory.NewEngineFromStepSettings(stepSettings)
}

func applyConfigOverrides(s *aisettings.StepSettings, cfg config) {
	if cfg.model != "" {
		s.Chat.ModelName = &cfg.model
	}
	if cfg.maxTokens > 0 {
		s.Chat.MaxResponseTokens = &cfg.maxTokens
	}
	streaming := true
	s.Chat.Stream = &streaming
}

func buildToolRegistry(cfg config) (tools.ToolRegistry, error) {
	if len(cfg.tools) == 0 {
		return nil, nil // No tools = single-pass inference
	}

	registry := tools.NewInMemoryToolRegistry()
	for _, t := range cfg.tools {
		def, err := tools.NewToolFromFunc(t.name, t.description, t.fn)
		if err != nil {
			return nil, fmt.Errorf("register tool %q: %w", t.name, err)
		}
		if err := registry.RegisterTool(t.name, *def); err != nil {
			return nil, fmt.Errorf("register tool %q: %w", t.name, err)
		}
	}
	return registry, nil
}

func buildResult(t *turns.Turn) *Result {
	result := &Result{Turn: t}

	// Extract text from assistant blocks
	for _, block := range t.Blocks {
		switch block.Kind {
		case turns.BlockKindLLMText:
			if text, ok := block.Payload["text"].(string); ok {
				result.Text += text
			}
		case turns.BlockKindToolCall:
			record := ToolCallRecord{
				Name: block.Payload["name"].(string),
			}
			if args, ok := block.Payload["arguments"].(map[string]any); ok {
				record.Args = args
			}
			result.ToolCalls = append(result.ToolCalls, record)
		}
	}

	// Extract usage and stop reason from turn metadata
	if ir, ok := turns.GetTurnData[*turns.InferenceResult](t, turns.InferenceResultKey); ok {
		result.StopReason = ir.StopReason
		result.Usage = ir.Usage
	}

	return result
}

// newDeltaSink creates an EventSink that calls fn for each text delta.
type deltaSink struct {
	fn func(string)
}

func newDeltaSink(fn func(string)) events.EventSink {
	return &deltaSink{fn: fn}
}

func (s *deltaSink) PublishEvent(e events.Event) error {
	if e.Type() == events.EventTypePartialCompletion {
		if text, ok := e.(*events.EventPartialCompletion); ok {
			s.fn(text.Delta)
		}
	}
	return nil
}

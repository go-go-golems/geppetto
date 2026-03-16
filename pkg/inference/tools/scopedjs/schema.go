package scopedjs

import (
	"context"
	"time"

	"github.com/dop251/goja_nodejs/require"
	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
)

type ToolDescription struct {
	Summary         string
	StarterSnippets []string
	Notes           []string
}

type ToolDefinitionSpec struct {
	Name        string
	Description ToolDescription
	Tags        []string
	Version     string
}

type StateMode string

const (
	StatePerCall    StateMode = "per_call"
	StatePerSession StateMode = "per_session"
	StateShared     StateMode = "shared"
)

type EvalOptions struct {
	Timeout        time.Duration
	MaxOutputChars int
	CaptureConsole bool
	StateMode      StateMode
}

func DefaultEvalOptions() EvalOptions {
	return EvalOptions{
		Timeout:        5 * time.Second,
		MaxOutputChars: 16_000,
		CaptureConsole: true,
		StateMode:      StatePerCall,
	}
}

type ScopeResolver[Scope any] func(ctx context.Context) (Scope, error)

type EnvironmentSpec[Scope any, Meta any] struct {
	RuntimeLabel string
	Tool         ToolDefinitionSpec
	DefaultEval  EvalOptions
	Configure    func(ctx context.Context, b *Builder, scope Scope) (Meta, error)
}

type BuildResult[Meta any] struct {
	Runtime  *gojengine.Runtime
	Meta     Meta
	Manifest EnvironmentManifest
	Cleanup  func() error
}

type EvalInput struct {
	Code  string         `json:"code"`
	Input map[string]any `json:"input,omitempty"`
}

type ConsoleLine struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

type EvalOutput struct {
	Result     any           `json:"result,omitempty"`
	Console    []ConsoleLine `json:"console,omitempty"`
	Error      string        `json:"error,omitempty"`
	DurationMs int64         `json:"durationMs,omitempty"`
}

type ModuleDoc struct {
	Name        string
	Description string
	Exports     []string
}

type GlobalDoc struct {
	Name        string
	Type        string
	Description string
}

type HelperDoc struct {
	Name        string
	Signature   string
	Description string
}

type EnvironmentManifest struct {
	Modules        []ModuleDoc
	Globals        []GlobalDoc
	Helpers        []HelperDoc
	BootstrapFiles []string
}

type ModuleRegistrar func(*require.Registry) error

type GlobalBinding func(ctx *gojengine.RuntimeContext) error

type moduleEntry struct {
	name     string
	register ModuleRegistrar
	doc      ModuleDoc
}

type globalEntry struct {
	name string
	bind GlobalBinding
	doc  GlobalDoc
}

type bootstrapEntry struct {
	name     string
	source   string
	filePath string
}

type Builder struct {
	modules          []moduleEntry
	nativeModules    []ggjmodules.NativeModule
	globals          []globalEntry
	initializers     []gojengine.RuntimeInitializer
	bootstrapEntries []bootstrapEntry
	manifest         EnvironmentManifest
}

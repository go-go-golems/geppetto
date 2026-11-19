// Package modelsource lists and resolves models from more than one credential
// source behind one reference syntax.
//
// A reference names a source and a model in that source:
//
//	pi:lunaroute/deepseek-4.1-flash   a model in Pi's agent directory (auth.json, models.json, models-store.json)
//	profile:claude                    an engine profile from a profile registry (e.g. pinocchio's profiles.yaml)
//	profile:team/claude               the same, with an explicit registry slug
//
// A reference without a known prefix is offered to each source in order, so
// "lunaroute/deepseek-4.1-flash" resolves through Pi when no profile claims it.
//
// Sources never expose credentials. Resolve returns payload settings (no keys)
// and a constructor for an engine whose connection is already bound to the
// credential's own endpoint (see the credential-scope topic).
package modelsource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

// ErrUnknownModel is returned (wrapped) when no source knows a reference.
var ErrUnknownModel = errors.New("unknown model")

// Model describes one model a source can run. It never carries a secret.
type Model struct {
	// Ref is the full reference, including the source prefix.
	Ref string `json:"ref" yaml:"ref"`
	// Source is the prefix of the source that owns the model ("pi", "profile").
	Source string `json:"source" yaml:"source"`
	// Label is a human-readable name.
	Label string `json:"label,omitempty" yaml:"label,omitempty"`
	// API is the wire protocol the engine speaks (e.g. "claude", "open-responses", "openai-completions").
	API string `json:"api,omitempty" yaml:"api,omitempty"`
	// ContextWindow and MaxOutputTokens are 0 when the source does not know them.
	ContextWindow   int  `json:"contextWindow,omitempty" yaml:"context_window,omitempty"`
	MaxOutputTokens int  `json:"maxOutputTokens,omitempty" yaml:"max_output_tokens,omitempty"`
	Reasoning       bool `json:"reasoning,omitempty" yaml:"reasoning,omitempty"`
	Images          bool `json:"images,omitempty" yaml:"images,omitempty"`
	// ThinkingLevels lists accepted Options.Thinking values, when the source knows them.
	ThinkingLevels []string `json:"thinkingLevels,omitempty" yaml:"thinking_levels,omitempty"`
	// Cost is USD per million tokens, when known.
	Cost *settings.ModelCost `json:"cost,omitempty" yaml:"cost,omitempty"`
}

// Options shape one resolution.
type Options struct {
	// Thinking is a thinking level ("off", "minimal", "low", "medium", "high",
	// "xhigh"). Sources that cannot map it ignore it; "" uses the source default.
	Thinking string
	// MaxTokens bounds one response; 0 uses the model's or profile's own limit.
	MaxTokens int
}

// Resolved is a model ready to run.
type Resolved struct {
	Model Model
	// Settings are payload settings without credential maps; read them, do not
	// try to use them to build a second engine with different credentials.
	Settings *settings.InferenceSettings
	build    func() (engine.Engine, error)
}

// Engine creates an engine. Engines are cheap; create one per run.
func (r *Resolved) Engine() (engine.Engine, error) {
	if r == nil || r.build == nil {
		return nil, errors.New("modelsource: resolved model has no engine constructor")
	}
	return r.build()
}

// Source lists and resolves the models of one credential source.
type Source interface {
	// Prefix is the reference prefix without the colon ("pi", "profile").
	Prefix() string
	// Models lists the models that can run now (they have a credential).
	Models(ctx context.Context) ([]Model, error)
	// Resolve resolves a reference with or without the source's prefix.
	Resolve(ctx context.Context, ref string, o Options) (*Resolved, error)
}

// splitRef returns the prefix and the rest of "prefix:rest", or "" and ref.
func splitRef(ref string) (string, string) {
	ref = strings.TrimSpace(ref)
	if i := strings.Index(ref, ":"); i > 0 && !strings.Contains(ref[:i], "/") {
		return ref[:i], ref[i+1:]
	}
	return "", ref
}

func unknown(ref string) error { return fmt.Errorf("%w: %s", ErrUnknownModel, ref) }

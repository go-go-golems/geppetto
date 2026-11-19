package modelsource

import (
	"context"
	"errors"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/pi"
)

// PiPrefix is the reference prefix of models from Pi's agent directory.
const PiPrefix = "pi"

type piSource struct{ dir *pi.AgentDir }

// FromPi serves the models in a loaded Pi agent directory. Only models with a
// configured credential and an API Geppetto can speak are listed.
func FromPi(dir *pi.AgentDir) Source { return &piSource{dir: dir} }

func (p *piSource) Prefix() string { return PiPrefix }

func (p *piSource) Models(_ context.Context) ([]Model, error) {
	ms := p.dir.Available()
	out := make([]Model, 0, len(ms))
	for _, m := range ms {
		out = append(out, piModel(m))
	}
	return out, nil
}

func piModel(m pi.Model) Model {
	label := m.Name
	if label == "" {
		label = m.ID
	}
	return Model{
		Ref: PiPrefix + ":" + m.Ref(), Source: PiPrefix, Label: label, API: m.API,
		ContextWindow: m.ContextWindow, MaxOutputTokens: m.MaxTokens,
		Reasoning: m.Reasoning, Images: m.Images(), ThinkingLevels: m.ThinkingLevels(),
	}
}

func (p *piSource) Resolve(ctx context.Context, ref string, o Options) (*Resolved, error) {
	if prefix, rest := splitRef(ref); prefix == PiPrefix {
		ref = rest
	} else if prefix != "" {
		return nil, unknown(ref)
	}
	r, err := p.dir.Resolve(ctx, ref, pi.ResolveOptions{Thinking: o.Thinking, MaxTokens: o.MaxTokens})
	if err != nil {
		if errors.Is(err, pi.ErrUnknownModel) {
			return nil, unknown(PiPrefix + ":" + ref)
		}
		return nil, err
	}
	return &Resolved{Model: piModel(r.Model), Settings: r.Settings, build: func() (engine.Engine, error) { return r.Engine() }}, nil
}

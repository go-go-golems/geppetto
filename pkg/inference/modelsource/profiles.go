package modelsource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/engine/factory"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

// ProfilePrefix is the reference prefix of engine profiles.
const ProfilePrefix = "profile"

type profileSource struct{ reg engineprofiles.Registry }

// FromProfiles serves the engine profiles of a registry (for example a
// ChainedRegistry over pinocchio's profiles.yaml). A profile is listed when its
// resolved settings name an engine; credentials are checked when it is resolved.
func FromProfiles(reg engineprofiles.Registry) Source { return &profileSource{reg: reg} }

func (p *profileSource) Prefix() string { return ProfilePrefix }

// Close closes the registry when it owns resources (a ChainedRegistry does).
func (p *profileSource) Close() error {
	if c, ok := p.reg.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (p *profileSource) Models(ctx context.Context) ([]Model, error) {
	regs, err := p.reg.ListRegistries(ctx)
	if err != nil {
		return nil, err
	}
	type entry struct {
		reg  engineprofiles.RegistrySlug
		prof *engineprofiles.EngineProfile
	}
	var all []entry
	count := map[engineprofiles.EngineProfileSlug]int{}
	for _, r := range regs {
		profs, err := p.reg.ListEngineProfiles(ctx, r.Slug)
		if err != nil {
			return nil, err
		}
		for _, prof := range profs {
			all = append(all, entry{r.Slug, prof})
			count[prof.Slug]++
		}
	}
	out := make([]Model, 0, len(all))
	for _, e := range all {
		res, err := p.reg.ResolveEngineProfile(ctx, engineprofiles.ResolveInput{RegistrySlug: e.reg, EngineProfileSlug: e.prof.Slug})
		if err != nil || res.InferenceSettings == nil || res.InferenceSettings.Chat == nil || res.InferenceSettings.Chat.Engine == nil {
			continue // a base layer or an incomplete profile, not a runnable model
		}
		name := string(e.prof.Slug)
		if count[e.prof.Slug] > 1 {
			name = string(e.reg) + "/" + name
		}
		out = append(out, profileModel(ProfilePrefix+":"+name, e.prof.DisplayName, res.InferenceSettings))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return out, nil
}

func profileModel(ref, display string, s *settings.InferenceSettings) Model {
	m := Model{Ref: ref, Source: ProfilePrefix, Label: display}
	if s.Chat != nil {
		if s.Chat.Engine != nil && m.Label == "" {
			m.Label = *s.Chat.Engine
		}
		if s.Chat.ApiType != nil {
			m.API = string(*s.Chat.ApiType)
		}
	}
	if mi := s.ModelInfo; mi != nil {
		if mi.ContextWindow != nil {
			m.ContextWindow = *mi.ContextWindow
		}
		if mi.MaxOutputTokens != nil {
			m.MaxOutputTokens = *mi.MaxOutputTokens
		}
		if mi.Reasoning != nil {
			m.Reasoning = *mi.Reasoning
		}
		for _, in := range mi.Input {
			if in == settings.InputModalityImage {
				m.Images = true
			}
		}
		if mi.Cost != nil {
			m.Cost = mi.Cost.Clone()
		}
	}
	return m
}

func (p *profileSource) Resolve(ctx context.Context, ref string, o Options) (*Resolved, error) {
	full := ref
	if prefix, rest := splitRef(ref); prefix == ProfilePrefix {
		ref = rest
	} else if prefix != "" {
		return nil, unknown(ref)
	} else {
		full = ProfilePrefix + ":" + ref
	}
	in := engineprofiles.ResolveInput{}
	slug := ref
	if i := strings.Index(ref, "/"); i >= 0 {
		rs, err := engineprofiles.ParseRegistrySlug(ref[:i])
		if err != nil {
			return nil, unknown(full)
		}
		in.RegistrySlug, slug = rs, ref[i+1:]
	}
	ps, err := engineprofiles.ParseEngineProfileSlug(slug)
	if err != nil {
		return nil, unknown(full)
	}
	in.EngineProfileSlug = ps
	res, err := p.reg.ResolveEngineProfile(ctx, in)
	if err != nil {
		if errors.Is(err, engineprofiles.ErrProfileNotFound) || errors.Is(err, engineprofiles.ErrRegistryNotFound) {
			return nil, unknown(full)
		}
		return nil, err
	}
	if res.InferenceSettings == nil || res.InferenceSettings.Chat == nil || res.InferenceSettings.Chat.Engine == nil {
		return nil, fmt.Errorf("modelsource: profile %s names no engine", full)
	}
	// Profiles are overlays; engines need every provider section, so start from defaults.
	base, err := settings.NewInferenceSettings()
	if err != nil {
		return nil, err
	}
	s, err := engineprofiles.MergeInferenceSettings(base, res.InferenceSettings)
	if err != nil {
		return nil, fmt.Errorf("modelsource: profile %s: %w", full, err)
	}
	if o.MaxTokens > 0 {
		n := o.MaxTokens
		s.Chat.MaxResponseTokens = &n
	}
	display := ""
	if prof, err := p.reg.GetEngineProfile(ctx, res.RegistrySlug, res.EngineProfileSlug); err == nil && prof != nil {
		display = prof.DisplayName
	}
	payload := settings.PayloadSettings(s)
	return &Resolved{
		Model:    profileModel(full, display, s),
		Settings: payload,
		build:    func() (engine.Engine, error) { return factory.NewEngineFromSettings(s) },
	}, nil
}

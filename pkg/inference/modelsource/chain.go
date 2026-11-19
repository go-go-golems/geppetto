package modelsource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/pi"
)

// Set is an ordered list of sources. A prefixed reference goes to the source
// with that prefix; an unprefixed one is offered to each source in order.
type Set struct {
	sources []Source
}

// Chain combines sources. Later sources with a prefix already present are ignored.
func Chain(srcs ...Source) *Set {
	s := &Set{}
	seen := map[string]bool{}
	for _, src := range srcs {
		if src == nil || seen[src.Prefix()] {
			continue
		}
		seen[src.Prefix()] = true
		s.sources = append(s.sources, src)
	}
	return s
}

// Sources returns the prefixes in order.
func (s *Set) Sources() []string {
	out := make([]string, 0, len(s.sources))
	for _, src := range s.sources {
		out = append(out, src.Prefix())
	}
	return out
}

// Models lists the models of every source, in source order. A source that
// fails to list is skipped and its error returned alongside the others' models.
func (s *Set) Models(ctx context.Context) ([]Model, error) {
	var out []Model
	var errs []error
	for _, src := range s.sources {
		ms, err := src.Models(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", src.Prefix(), err))
			continue
		}
		out = append(out, ms...)
	}
	return out, errors.Join(errs...)
}

// Resolve resolves a reference.
func (s *Set) Resolve(ctx context.Context, ref string, o Options) (*Resolved, error) {
	prefix, _ := splitRef(ref)
	if prefix != "" {
		for _, src := range s.sources {
			if src.Prefix() == prefix {
				return src.Resolve(ctx, ref, o)
			}
		}
		return nil, unknown(ref)
	}
	for _, src := range s.sources {
		r, err := src.Resolve(ctx, ref, o)
		if err == nil {
			return r, nil
		}
		if !errors.Is(err, ErrUnknownModel) {
			return nil, err
		}
	}
	return nil, unknown(ref)
}

// Close closes every source that owns resources.
func (s *Set) Close() error {
	var errs []error
	for _, src := range s.sources {
		if c, ok := src.(io.Closer); ok {
			errs = append(errs, c.Close())
		}
	}
	return errors.Join(errs...)
}

// LoadOptions selects the default sources.
type LoadOptions struct {
	// PiDir is Pi's agent directory; "" uses $PI_CODING_AGENT_DIR or ~/.pi/agent.
	PiDir string
	// SkipPi disables the Pi source.
	SkipPi bool
	// PiOptions are passed to pi.LoadAgentDir.
	PiOptions []pi.Option
	// ProfileSources are registry source specs (yaml:PATH, sqlite:PATH, a bare
	// .yaml path). Empty uses <UserConfigDir>/<AppName>/profiles.yaml when it exists.
	ProfileSources []string
	// AppName names the default profiles directory; "" means "pinocchio".
	AppName string
	// SkipProfiles disables the profile source.
	SkipProfiles bool
}

// Load builds the default set: engine profiles first, then Pi. A source whose
// files do not exist is skipped and reported in the returned notes; a source
// that exists but cannot be read is an error.
func Load(ctx context.Context, o LoadOptions) (*Set, []string, error) {
	var srcs []Source
	var notes []string
	if !o.SkipProfiles {
		specs := o.ProfileSources
		if len(specs) == 0 {
			app := o.AppName
			if app == "" {
				app = "pinocchio"
			}
			if dir, err := os.UserConfigDir(); err == nil {
				p := filepath.Join(dir, app, "profiles.yaml")
				if st, err := os.Stat(p); err == nil && !st.IsDir() {
					specs = []string{p}
				} else {
					notes = append(notes, "profiles: "+p+" not found")
				}
			}
		}
		if len(specs) > 0 {
			parsed, err := engineprofiles.ParseRegistrySourceSpecs(specs)
			if err != nil {
				return nil, notes, err
			}
			reg, err := engineprofiles.NewChainedRegistryFromSourceSpecs(ctx, parsed)
			if err != nil {
				return nil, notes, fmt.Errorf("profiles %s: %w", strings.Join(specs, ","), err)
			}
			srcs = append(srcs, FromProfiles(reg))
		}
	}
	if !o.SkipPi {
		piDir := o.PiDir
		if piDir == "" {
			piDir = os.Getenv(pi.EnvAgentDir)
		}
		dir := pi.DefaultDir(piDir)
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			a, err := pi.LoadAgentDir(dir, o.PiOptions...)
			if err != nil {
				return nil, notes, fmt.Errorf("pi %s: %w", dir, err)
			}
			srcs = append(srcs, FromPi(a))
		} else {
			notes = append(notes, "pi: "+dir+" not found")
		}
	}
	return Chain(srcs...), notes, nil
}

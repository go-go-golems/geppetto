package profiles

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type StackResolverOptions struct {
	MaxDepth int
}

type ProfileStackLayer struct {
	RegistrySlug RegistrySlug
	ProfileSlug  ProfileSlug
	Profile      *Profile
}

type stackResolverIdentity struct {
	registrySlug RegistrySlug
	profileSlug  ProfileSlug
}

func (i stackResolverIdentity) String() string {
	return fmt.Sprintf("%s/%s", i.registrySlug, i.profileSlug)
}

// ExpandProfileStack resolves a profile's stack into deterministic base->leaf layers.
// The result always includes the root profile as the final layer.
func (r *StoreRegistry) ExpandProfileStack(ctx context.Context, registrySlug RegistrySlug, profileSlug ProfileSlug, opts StackResolverOptions) ([]ProfileStackLayer, error) {
	if r == nil {
		return nil, fmt.Errorf("store registry is nil")
	}

	resolvedRegistrySlug := r.resolveRegistrySlug(registrySlug)
	registry, err := r.GetRegistry(ctx, resolvedRegistrySlug)
	if err != nil {
		return nil, err
	}
	resolvedProfileSlug, err := r.resolveProfileSlugForRegistry(profileSlug, registry)
	if err != nil {
		return nil, err
	}

	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = DefaultProfileStackValidationMaxDepth
	}

	root := stackResolverIdentity{
		registrySlug: resolvedRegistrySlug,
		profileSlug:  resolvedProfileSlug,
	}

	profileCache := map[stackResolverIdentity]*Profile{}
	loadProfile := func(identity stackResolverIdentity) (*Profile, error) {
		if profile, ok := profileCache[identity]; ok {
			return profile, nil
		}
		profile, err := r.GetProfile(ctx, identity.registrySlug, identity.profileSlug)
		if err != nil {
			return nil, err
		}
		profileCache[identity] = profile
		return profile, nil
	}

	if _, err := loadProfile(root); err != nil {
		return nil, err
	}

	visited := map[stackResolverIdentity]bool{}
	inPath := map[stackResolverIdentity]int{}
	path := make([]stackResolverIdentity, 0, 8)
	layers := make([]ProfileStackLayer, 0, 8)

	var walk func(current stackResolverIdentity) error
	walk = func(current stackResolverIdentity) error {
		if visited[current] {
			return nil
		}

		currentProfile, err := loadProfile(current)
		if err != nil {
			return err
		}

		inPath[current] = len(path)
		path = append(path, current)

		for i, ref := range currentProfile.Stack {
			field := fmt.Sprintf("registry.profiles[%s].stack[%d]", current.profileSlug, i)
			if err := ValidateProfileRef(ref, field); err != nil {
				return err
			}

			targetRegistry := current.registrySlug
			if !ref.RegistrySlug.IsZero() {
				targetRegistry = ref.RegistrySlug
			}
			target := stackResolverIdentity{
				registrySlug: targetRegistry,
				profileSlug:  ref.ProfileSlug,
			}

			if len(path)+1 > maxDepth {
				return &ValidationError{
					Field:  field,
					Reason: fmt.Sprintf("stack depth exceeds max_depth=%d while traversing %s -> %s", maxDepth, formatStackResolverPath(path), target.String()),
				}
			}

			if cycleStart, ok := inPath[target]; ok {
				cycle := append(append([]stackResolverIdentity(nil), path[cycleStart:]...), target)
				return &ValidationError{
					Field:  field,
					Reason: fmt.Sprintf("stack cycle detected: %s", formatStackResolverCycle(cycle)),
				}
			}

			if _, err := loadProfile(target); err != nil {
				switch {
				case errors.Is(err, ErrRegistryNotFound):
					return &ValidationError{
						Field:  field,
						Reason: fmt.Sprintf("referenced registry %q not found", target.registrySlug),
					}
				case errors.Is(err, ErrProfileNotFound):
					return &ValidationError{
						Field:  field,
						Reason: fmt.Sprintf("referenced profile %q not found in registry %q", target.profileSlug, target.registrySlug),
					}
				default:
					return err
				}
			}

			if err := walk(target); err != nil {
				return err
			}
		}

		path = path[:len(path)-1]
		delete(inPath, current)
		visited[current] = true
		layers = append(layers, ProfileStackLayer{
			RegistrySlug: current.registrySlug,
			ProfileSlug:  current.profileSlug,
			Profile:      currentProfile.Clone(),
		})
		return nil
	}

	if err := walk(root); err != nil {
		return nil, err
	}

	return layers, nil
}

func formatStackResolverPath(path []stackResolverIdentity) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, 0, len(path))
	for _, identity := range path {
		parts = append(parts, identity.String())
	}
	return strings.Join(parts, " -> ")
}

func formatStackResolverCycle(cycle []stackResolverIdentity) string {
	if len(cycle) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cycle))
	for _, identity := range cycle {
		parts = append(parts, identity.String())
	}
	return strings.Join(parts, " -> ")
}

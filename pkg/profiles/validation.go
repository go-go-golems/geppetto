package profiles

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const DefaultProfileStackValidationMaxDepth = 32

type StackValidationOptions struct {
	MaxDepth                    int
	AllowUnresolvedExternalRefs bool
}

func ValidateRegistrySlug(slug RegistrySlug) error {
	if slug.IsZero() {
		return &ValidationError{Field: "registry.slug", Reason: "must not be empty"}
	}
	if _, err := ParseRegistrySlug(slug.String()); err != nil {
		return &ValidationError{Field: "registry.slug", Reason: err.Error()}
	}
	return nil
}

func ValidateProfileSlug(slug ProfileSlug) error {
	if slug.IsZero() {
		return &ValidationError{Field: "profile.slug", Reason: "must not be empty"}
	}
	if _, err := ParseProfileSlug(slug.String()); err != nil {
		return &ValidationError{Field: "profile.slug", Reason: err.Error()}
	}
	return nil
}

func ValidateRuntimeSpec(spec RuntimeSpec) error {
	if err := validateMiddlewareUses(spec.Middlewares, "runtime.middlewares"); err != nil {
		return err
	}

	for i, tool := range spec.Tools {
		if strings.TrimSpace(tool) == "" {
			return &ValidationError{Field: fmt.Sprintf("runtime.tools[%d]", i), Reason: "must not be empty"}
		}
	}

	return nil
}

func validateMiddlewareUses(middlewares []MiddlewareUse, fieldPrefix string) error {
	seenIDs := map[string]int{}
	for i, mw := range middlewares {
		name := strings.TrimSpace(mw.Name)
		if name == "" {
			return &ValidationError{
				Field:  fmt.Sprintf("%s[%d].name", fieldPrefix, i),
				Reason: "must not be empty",
			}
		}

		id := strings.TrimSpace(mw.ID)
		if mw.ID != "" && id == "" {
			return &ValidationError{
				Field:  fmt.Sprintf("%s[%d].id", fieldPrefix, i),
				Reason: "must not be empty",
			}
		}

		if id == "" {
			continue
		}
		if firstIndex, ok := seenIDs[id]; ok {
			return &ValidationError{
				Field:  fmt.Sprintf("%s[%d].id", fieldPrefix, i),
				Reason: fmt.Sprintf("duplicate middleware instance id %q (first seen at %s[%d].id)", id, fieldPrefix, firstIndex),
			}
		}
		seenIDs[id] = i
	}
	return nil
}

func ValidatePolicySpec(policy PolicySpec) error {
	allow := map[string]struct{}{}
	for i, key := range policy.AllowedOverrideKeys {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			return &ValidationError{Field: fmt.Sprintf("policy.allowed_override_keys[%d]", i), Reason: "must not be empty"}
		}
		allow[normalized] = struct{}{}
	}

	deny := map[string]struct{}{}
	for i, key := range policy.DeniedOverrideKeys {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			return &ValidationError{Field: fmt.Sprintf("policy.denied_override_keys[%d]", i), Reason: "must not be empty"}
		}
		if _, ok := allow[normalized]; ok {
			return &ValidationError{Field: "policy.override_keys", Reason: fmt.Sprintf("key %q appears in both allow and deny lists", normalized)}
		}
		deny[normalized] = struct{}{}
	}

	_ = deny
	return nil
}

func ValidateProfile(profile *Profile) error {
	if profile == nil {
		return &ValidationError{Field: "profile", Reason: "must not be nil"}
	}
	if err := ValidateProfileSlug(profile.Slug); err != nil {
		return err
	}
	if err := ValidateRuntimeSpec(profile.Runtime); err != nil {
		return err
	}
	if err := ValidatePolicySpec(profile.Policy); err != nil {
		return err
	}
	if err := ValidateProfileExtensions(profile.Extensions); err != nil {
		return err
	}
	for i, ref := range profile.Stack {
		if err := ValidateProfileRef(ref, fmt.Sprintf("profile.stack[%d]", i)); err != nil {
			return err
		}
	}
	return nil
}

func ValidateProfileRef(ref ProfileRef, fieldPrefix string) error {
	if fieldPrefix == "" {
		fieldPrefix = "profile.stack"
	}
	if ref.ProfileSlug.IsZero() {
		return &ValidationError{Field: fieldPrefix + ".profile_slug", Reason: "must not be empty"}
	}
	if _, err := ParseProfileSlug(ref.ProfileSlug.String()); err != nil {
		return &ValidationError{Field: fieldPrefix + ".profile_slug", Reason: err.Error()}
	}
	if !ref.RegistrySlug.IsZero() {
		if _, err := ParseRegistrySlug(ref.RegistrySlug.String()); err != nil {
			return &ValidationError{Field: fieldPrefix + ".registry_slug", Reason: err.Error()}
		}
	}
	return nil
}

func ValidateProfileExtensions(extensions map[string]any) error {
	for rawKey, rawValue := range extensions {
		key, err := ParseExtensionKey(rawKey)
		if err != nil {
			return &ValidationError{
				Field:  fmt.Sprintf("profile.extensions[%s]", strings.TrimSpace(rawKey)),
				Reason: err.Error(),
			}
		}
		if _, err := json.Marshal(rawValue); err != nil {
			return &ValidationError{
				Field:  fmt.Sprintf("profile.extensions[%s]", key),
				Reason: fmt.Sprintf("payload must be JSON-serializable: %v", err),
			}
		}
	}
	return nil
}

func ValidateRegistry(registry *ProfileRegistry) error {
	if registry == nil {
		return &ValidationError{Field: "registry", Reason: "must not be nil"}
	}
	if err := ValidateRegistrySlug(registry.Slug); err != nil {
		return err
	}

	if len(registry.Profiles) > 0 && registry.DefaultProfileSlug.IsZero() {
		return &ValidationError{Field: "registry.default_profile_slug", Reason: "must be set when profiles are present"}
	}

	for slug, profile := range registry.Profiles {
		if err := ValidateProfileSlug(slug); err != nil {
			return err
		}
		if profile == nil {
			return &ValidationError{Field: fmt.Sprintf("registry.profiles[%s]", slug), Reason: "must not be nil"}
		}
		if err := ValidateProfile(profile); err != nil {
			return err
		}
		if profile.Slug != slug {
			return &ValidationError{Field: fmt.Sprintf("registry.profiles[%s].slug", slug), Reason: "map key and profile slug must match"}
		}
	}

	if !registry.DefaultProfileSlug.IsZero() {
		if err := ValidateProfileSlug(registry.DefaultProfileSlug); err != nil {
			return err
		}
		if len(registry.Profiles) > 0 {
			if _, ok := registry.Profiles[registry.DefaultProfileSlug]; !ok {
				return &ValidationError{Field: "registry.default_profile_slug", Reason: "default profile does not exist in registry"}
			}
		}
	}

	if err := ValidateProfileStackTopology([]*ProfileRegistry{registry}, StackValidationOptions{
		MaxDepth:                    DefaultProfileStackValidationMaxDepth,
		AllowUnresolvedExternalRefs: true,
	}); err != nil {
		return err
	}

	return nil
}

type stackNode struct {
	registrySlug RegistrySlug
	profileSlug  ProfileSlug
}

func (n stackNode) String() string {
	return fmt.Sprintf("%s/%s", n.registrySlug, n.profileSlug)
}

func ValidateProfileStackTopology(registries []*ProfileRegistry, opts StackValidationOptions) error {
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = DefaultProfileStackValidationMaxDepth
	}

	graph := map[stackNode]*Profile{}
	knownRegistries := map[RegistrySlug]struct{}{}
	for _, registry := range registries {
		if registry == nil {
			continue
		}
		knownRegistries[registry.Slug] = struct{}{}
		for profileSlug, profile := range registry.Profiles {
			if profile == nil {
				continue
			}
			node := stackNode{registrySlug: registry.Slug, profileSlug: profileSlug}
			graph[node] = profile
		}
	}

	if len(graph) == 0 {
		return nil
	}

	nodes := make([]stackNode, 0, len(graph))
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].registrySlug != nodes[j].registrySlug {
			return nodes[i].registrySlug < nodes[j].registrySlug
		}
		return nodes[i].profileSlug < nodes[j].profileSlug
	})

	visited := map[stackNode]bool{}
	inPath := map[stackNode]int{}
	path := make([]stackNode, 0, 8)

	var walk func(node stackNode) error
	walk = func(node stackNode) error {
		if visited[node] {
			return nil
		}
		inPath[node] = len(path)
		path = append(path, node)

		profile := graph[node]
		for i, ref := range profile.Stack {
			targetRegistry := node.registrySlug
			if !ref.RegistrySlug.IsZero() {
				targetRegistry = ref.RegistrySlug
			}
			target := stackNode{
				registrySlug: targetRegistry,
				profileSlug:  ref.ProfileSlug,
			}

			field := fmt.Sprintf("registry.profiles[%s].stack[%d]", node.profileSlug, i)
			if _, ok := graph[target]; !ok {
				if !ref.RegistrySlug.IsZero() && opts.AllowUnresolvedExternalRefs {
					// Only bypass unresolved refs that point to registries outside
					// the validated registry set.
					if _, isKnownRegistry := knownRegistries[ref.RegistrySlug]; !isKnownRegistry {
						continue
					}
				}
				return &ValidationError{
					Field:  field,
					Reason: fmt.Sprintf("referenced profile %q not found in registry %q", target.profileSlug, target.registrySlug),
				}
			}

			if cycleStart, ok := inPath[target]; ok {
				cycle := append(append([]stackNode(nil), path[cycleStart:]...), target)
				return &ValidationError{
					Field:  field,
					Reason: fmt.Sprintf("stack cycle detected: %s", formatStackCycle(cycle)),
				}
			}

			if len(path)+1 > maxDepth {
				return &ValidationError{
					Field:  field,
					Reason: fmt.Sprintf("stack depth exceeds max_depth=%d while traversing %s -> %s", maxDepth, formatStackPath(path), target.String()),
				}
			}

			if err := walk(target); err != nil {
				return err
			}
		}

		path = path[:len(path)-1]
		delete(inPath, node)
		visited[node] = true
		return nil
	}

	for _, node := range nodes {
		if err := walk(node); err != nil {
			return err
		}
	}

	return nil
}

func formatStackPath(path []stackNode) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, 0, len(path))
	for _, node := range path {
		parts = append(parts, node.String())
	}
	return strings.Join(parts, " -> ")
}

func formatStackCycle(cycle []stackNode) string {
	if len(cycle) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cycle))
	for _, node := range cycle {
		parts = append(parts, node.String())
	}
	return strings.Join(parts, " -> ")
}

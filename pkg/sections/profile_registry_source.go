package sections

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/profiles"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

// GatherFlagsFromProfileRegistry loads profile flags via the profile registry abstraction.
// It is drop-in compatible with sources.GatherFlagsFromProfiles for migration use.
func GatherFlagsFromProfileRegistry(
	profileRegistrySources []string,
	profile string,
	options ...fields.ParseOption,
) sources.Middleware {
	return func(next sources.HandlerFunc) sources.HandlerFunc {
		return func(schema_ *schema.Schema, parsedValues *values.Values) error {
			err := next(schema_, parsedValues)
			if err != nil {
				return err
			}

			if strings.TrimSpace(profile) == "" {
				profile = "default"
			}

			specs, err := profiles.ParseRegistrySourceSpecs(profileRegistrySources)
			if err != nil {
				return err
			}
			registry, err := profiles.NewChainedRegistryFromSourceSpecs(context.Background(), specs)
			if err != nil {
				return err
			}
			defer func() {
				_ = registry.Close()
			}()

			profileSlug, err := profiles.ParseProfileSlug(profile)
			if err != nil {
				return err
			}

			resolved, err := registry.ResolveEffectiveProfile(context.Background(), profiles.ResolveInput{
				ProfileSlug: profileSlug,
			})
			if err != nil {
				return err
			}

			patchMap, err := sourceMapFromSectionPatch(resolved.EffectiveRuntime.StepSettingsPatch)
			if err != nil {
				return err
			}
			if len(patchMap) == 0 {
				return nil
			}
			return sources.FromMap(patchMap, options...)(sources.Identity)(schema_, parsedValues)
		}
	}
}

func sourceMapFromSectionPatch(raw map[string]any) (map[string]map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := map[string]map[string]interface{}{}
	for sectionSlugRaw, sectionRaw := range raw {
		sectionSlug := strings.TrimSpace(sectionSlugRaw)
		if sectionSlug == "" {
			return nil, fmt.Errorf("step settings patch section slug cannot be empty")
		}
		sectionMap, ok := sectionRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("step settings patch section %q must be an object", sectionSlug)
		}
		fieldsMap := map[string]interface{}{}
		for fieldName, value := range sectionMap {
			fieldsMap[fieldName] = value
		}
		out[sectionSlug] = fieldsMap
	}
	return out, nil
}

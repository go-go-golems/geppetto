package sections

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	defaultProfileFile string,
	profileFile string,
	profile string,
	defaultProfileName string,
	options ...fields.ParseOption,
) sources.Middleware {
	return func(next sources.HandlerFunc) sources.HandlerFunc {
		return func(schema_ *schema.Schema, parsedValues *values.Values) error {
			err := next(schema_, parsedValues)
			if err != nil {
				return err
			}

			if defaultProfileName == "" {
				defaultProfileName = "default"
			}
			if strings.TrimSpace(profile) == "" {
				profile = defaultProfileName
			}

			_, err = os.Stat(profileFile)
			if os.IsNotExist(err) {
				if profileFile != defaultProfileFile {
					return fmt.Errorf("profile file %s does not exist", profileFile)
				}
				if profile != defaultProfileName {
					return fmt.Errorf("profile file %s does not exist (requested profile %s)", profileFile, profile)
				}
				return nil
			}
			if err != nil {
				return err
			}

			store, err := profiles.NewYAMLFileProfileStore(profileFile, profiles.MustRegistrySlug("default"))
			if err != nil {
				return err
			}
			defer func() {
				_ = store.Close()
			}()

			registry, err := profiles.NewStoreRegistry(store, profiles.MustRegistrySlug("default"))
			if err != nil {
				return err
			}

			profileSlug, err := profiles.ParseProfileSlug(profile)
			if err != nil {
				return err
			}

			resolved, err := registry.ResolveEffectiveProfile(context.Background(), profiles.ResolveInput{
				RegistrySlug: profiles.MustRegistrySlug("default"),
				ProfileSlug:  profileSlug,
			})
			if err != nil {
				if errors.Is(err, profiles.ErrProfileNotFound) || errors.Is(err, profiles.ErrRegistryNotFound) {
					if profile != defaultProfileName {
						return fmt.Errorf("profile %s not found in %s", profile, profileFile)
					}
					return nil
				}
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

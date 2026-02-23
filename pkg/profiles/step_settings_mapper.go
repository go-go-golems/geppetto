package profiles

import (
	"fmt"
	"strings"

	embeddingsconfig "github.com/go-go-golems/geppetto/pkg/embeddings/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func newStepSettingsSchema() (*schema.Schema, error) {
	chatSection, err := settings.NewChatValueSection()
	if err != nil {
		return nil, err
	}

	clientSection, err := settings.NewClientValueSection()
	if err != nil {
		return nil, err
	}

	claudeSection, err := claude.NewValueSection()
	if err != nil {
		return nil, err
	}

	geminiSection, err := gemini.NewValueSection()
	if err != nil {
		return nil, err
	}

	openaiSection, err := openai.NewValueSection()
	if err != nil {
		return nil, err
	}

	embeddingsSection, err := embeddingsconfig.NewEmbeddingsValueSection()
	if err != nil {
		return nil, err
	}

	inferenceSection, err := settings.NewInferenceValueSection()
	if err != nil {
		return nil, err
	}

	return schema.NewSchema(schema.WithSections(
		chatSection,
		clientSection,
		claudeSection,
		geminiSection,
		openaiSection,
		embeddingsSection,
		inferenceSection,
	)), nil
}

func ApplyStepSettingsPatch(base *settings.StepSettings, patch map[string]any) (*settings.StepSettings, error) {
	var err error
	resolved := base
	if resolved == nil {
		resolved, err = settings.NewStepSettings()
		if err != nil {
			return nil, err
		}
	} else {
		resolved = resolved.Clone()
	}

	if len(patch) == 0 {
		return resolved, nil
	}

	schema_, err := newStepSettingsSchema()
	if err != nil {
		return nil, err
	}

	parsed := values.New()
	err = schema_.ForEachE(func(_ string, section schema.Section) error {
		parsed.GetOrCreate(section)
		return nil
	})
	if err != nil {
		return nil, err
	}

	patchMap, err := normalizeSectionPatchMap(patch)
	if err != nil {
		return nil, err
	}

	err = sources.Execute(
		schema_,
		parsed,
		sources.FromMap(patchMap),
	)
	if err != nil {
		return nil, err
	}

	if err := resolved.UpdateFromParsedValues(parsed); err != nil {
		return nil, err
	}

	return resolved, nil
}

func MergeStepSettingsPatches(base map[string]any, overlay map[string]any) (map[string]any, error) {
	if len(base) == 0 && len(overlay) == 0 {
		return nil, nil
	}

	merged := map[string]any{}
	baseMap, err := normalizeSectionPatchMap(base)
	if err != nil {
		return nil, err
	}
	for sectionSlug, sectionValues := range baseMap {
		merged[sectionSlug] = deepCopyStringAnyMap(sectionValues)
	}

	overlayMap, err := normalizeSectionPatchMap(overlay)
	if err != nil {
		return nil, err
	}
	for sectionSlug, sectionValues := range overlayMap {
		current, _ := toStringAnyMap(merged[sectionSlug])
		if current == nil {
			current = map[string]any{}
		}
		for fieldName, value := range sectionValues {
			current[fieldName] = deepCopyAny(value)
		}
		merged[sectionSlug] = current
	}

	if len(merged) == 0 {
		return nil, nil
	}
	return merged, nil
}

func normalizeSectionPatchMap(raw map[string]any) (map[string]map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	out := map[string]map[string]interface{}{}
	for sectionSlugRaw, sectionRaw := range raw {
		sectionSlug := strings.TrimSpace(sectionSlugRaw)
		if sectionSlug == "" {
			return nil, &ValidationError{Field: "runtime.step_settings_patch", Reason: "section slug cannot be empty"}
		}
		sectionMap, ok := toStringAnyMap(sectionRaw)
		if !ok {
			return nil, &ValidationError{Field: fmt.Sprintf("runtime.step_settings_patch.%s", sectionSlug), Reason: "section patch must be an object"}
		}
		fieldMap := map[string]interface{}{}
		for fieldNameRaw, value := range sectionMap {
			fieldName := strings.TrimSpace(fieldNameRaw)
			if fieldName == "" {
				return nil, &ValidationError{Field: fmt.Sprintf("runtime.step_settings_patch.%s", sectionSlug), Reason: "field name cannot be empty"}
			}
			fieldMap[fieldName] = deepCopyAny(value)
		}
		out[sectionSlug] = fieldMap
	}

	return out, nil
}

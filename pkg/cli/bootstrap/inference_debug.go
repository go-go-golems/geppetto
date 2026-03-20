package bootstrap

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/embeddings/config"
	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/gemini"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/ollama"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"gopkg.in/yaml.v3"
)

const InferenceDebugSectionSlug = "debug-settings"

type InferenceDebugSettings struct {
	PrintInferenceSettings bool `glazed:"print-inference-settings"`
}

type InferenceDebugOutputOptions struct {
	CommandBase    *aisettings.InferenceSettings
	ParsedForTrace *values.Values
}

type InferenceSettingSource struct {
	Value any                `yaml:"value"`
	Log   []fields.ParseStep `yaml:"log,omitempty"`
}

type traceLeaf struct {
	Value any
	Log   []fields.ParseStep
}

type flattenedLeaf struct {
	Path  string
	Value any
}

var inferenceSectionFieldPathMap = map[string]map[string]string{
	aisettings.AiChatSlug:      buildStructFieldPathMap(reflect.TypeOf(aisettings.ChatSettings{})),
	"openai-chat":              buildStructFieldPathMap(reflect.TypeOf(openai.Settings{})),
	aisettings.AiClientSlug:    buildStructFieldPathMap(reflect.TypeOf(aisettings.ClientSettings{})),
	"claude-chat":              buildStructFieldPathMap(reflect.TypeOf(claude.Settings{})),
	"gemini-chat":              buildStructFieldPathMap(reflect.TypeOf(gemini.Settings{})),
	"ollama-chat":              buildStructFieldPathMap(reflect.TypeOf(ollama.Settings{})),
	config.EmbeddingsSlug:      buildStructFieldPathMap(reflect.TypeOf(config.EmbeddingsConfig{})),
	aisettings.AiInferenceSlug: buildStructFieldPathMap(reflect.TypeOf(engine.InferenceConfig{})),
}

func NewInferenceDebugSection() (schema.Section, error) {
	return schema.NewSection(
		InferenceDebugSectionSlug,
		"Inference debug",
		schema.WithFields(
			fields.New(
				"print-inference-settings",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Print the final resolved inference settings together with source logs and exit"),
			),
		),
	)
}

func BuildInferenceTraceParsedValues(cfg AppBootstrapConfig, parsed *values.Values) (*values.Values, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	baseSections, err := cfg.BuildBaseSections()
	if err != nil {
		return nil, err
	}

	traceParsed := values.New()
	baseSchema := schema.NewSchema(schema.WithSections(baseSections...))
	configFiles, err := ResolveCLIConfigFiles(cfg, parsed)
	if err != nil {
		return nil, err
	}
	if err := sources.Execute(
		baseSchema,
		traceParsed,
		sources.FromEnv(cfg.normalizedEnvPrefix(), fields.WithSource("env")),
		sources.FromFiles(
			configFiles,
			sources.WithConfigFileMapper(cfg.ConfigFileMapper),
			sources.WithParseOptions(fields.WithSource("config")),
		),
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	); err != nil {
		return nil, err
	}

	if parsed != nil {
		if err := traceParsed.Merge(parsed); err != nil {
			return nil, err
		}
	}
	return traceParsed, nil
}

func BuildInferenceSettingsSourceTrace(
	commandBase *aisettings.InferenceSettings,
	parsed *values.Values,
	resolved *ResolvedCLIEngineSettings,
) (map[string]any, error) {
	if resolved == nil || resolved.FinalInferenceSettings == nil {
		return nil, fmt.Errorf("resolved final inference settings are required")
	}

	leaves := map[string]*traceLeaf{}

	if commandBase != nil {
		if err := applySettingsSource(leaves, commandBase, fields.ParseStep{
			Source: "command",
			Metadata: map[string]any{
				"kind": "command-baseline",
			},
		}, false); err != nil {
			return nil, err
		}
	}

	applyParsedValueSources(leaves, parsed)

	if resolved.ResolvedEngineProfile != nil && resolved.ResolvedEngineProfile.InferenceSettings != nil {
		if err := applyProfileSource(leaves, resolved.ResolvedEngineProfile); err != nil {
			return nil, err
		}
	}

	finalMap, err := inferenceSettingsToMap(resolved.FinalInferenceSettings)
	if err != nil {
		return nil, err
	}

	out := map[string]any{}
	for _, leaf := range flattenMap(finalMap, nil) {
		entry, ok := leaves[leaf.Path]
		if !ok {
			entry = &traceLeaf{
				Value: leaf.Value,
				Log: []fields.ParseStep{{
					Source: "implicit-defaults",
					Value:  leaf.Value,
				}},
			}
		}
		setNestedValue(out, strings.Split(leaf.Path, "."), &InferenceSettingSource{
			Value: leaf.Value,
			Log:   cloneParseSteps(entry.Log),
		})
	}

	return out, nil
}

func WriteInferenceSettingsDebugYAML(
	w io.Writer,
	resolved *ResolvedCLIEngineSettings,
	opts InferenceDebugOutputOptions,
) error {
	if resolved == nil || resolved.FinalInferenceSettings == nil {
		return fmt.Errorf("resolved final inference settings are required")
	}

	trace, err := BuildInferenceSettingsSourceTrace(opts.CommandBase, opts.ParsedForTrace, resolved)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"settings": resolved.FinalInferenceSettings,
		"sources":  trace,
	}

	return writeRedactedYAML(w, payload)
}

func HandleInferenceDebugOutput(
	w io.Writer,
	cfg AppBootstrapConfig,
	parsed *values.Values,
	settings InferenceDebugSettings,
	resolved *ResolvedCLIEngineSettings,
	opts InferenceDebugOutputOptions,
) (bool, error) {
	if !settings.PrintInferenceSettings {
		return false, nil
	}
	if resolved == nil || resolved.FinalInferenceSettings == nil {
		return true, fmt.Errorf("resolved final inference settings are required")
	}

	parsedForTrace := opts.ParsedForTrace
	if parsedForTrace == nil {
		var err error
		parsedForTrace, err = BuildInferenceTraceParsedValues(cfg, parsed)
		if err != nil {
			return true, err
		}
	}

	return true, WriteInferenceSettingsDebugYAML(w, resolved, InferenceDebugOutputOptions{
		CommandBase:    opts.CommandBase,
		ParsedForTrace: parsedForTrace,
	})
}

func applyParsedValueSources(leaves map[string]*traceLeaf, parsed *values.Values) {
	if parsed == nil {
		return
	}

	parsed.ForEach(func(sectionSlug string, sv *values.SectionValues) {
		if sv == nil || sv.Fields == nil {
			return
		}
		sv.Fields.ForEach(func(fieldName string, fv *fields.FieldValue) {
			if fv == nil {
				return
			}
			path, ok := inferencePathForParsedField(sectionSlug, fieldName)
			if !ok {
				return
			}

			if existing, found := leaves[path]; found && len(existing.Log) > 0 {
				existing.Value = fv.Value
				existing.Log = append(cloneParseSteps(existing.Log), cloneParseSteps(fv.Log)...)
				return
			}

			leaves[path] = &traceLeaf{
				Value: fv.Value,
				Log:   cloneParseSteps(fv.Log),
			}
		})
	})
}

func applyProfileSource(leaves map[string]*traceLeaf, resolved *gepprofiles.ResolvedEngineProfile) error {
	if resolved == nil || resolved.InferenceSettings == nil {
		return nil
	}

	metadata := map[string]any{
		"registry_slug": resolved.RegistrySlug.String(),
		"profile_slug":  resolved.EngineProfileSlug.String(),
	}
	if source, ok := resolved.Metadata["profile.source"]; ok {
		metadata["profile_source"] = source
	}
	if lineage, ok := resolved.Metadata["profile.stack.lineage"]; ok {
		metadata["stack_lineage"] = lineage
	}

	return applySettingsSource(leaves, resolved.InferenceSettings, fields.ParseStep{
		Source:   "profile",
		Metadata: metadata,
	}, true)
}

func applySettingsSource(leaves map[string]*traceLeaf, settings *aisettings.InferenceSettings, step fields.ParseStep, appendMode bool) error {
	settingsMap, err := inferenceSettingsToMap(settings)
	if err != nil {
		return err
	}
	for _, leaf := range flattenMap(settingsMap, nil) {
		step_ := cloneParseStep(step)
		step_.Value = leaf.Value
		if appendMode {
			existing, ok := leaves[leaf.Path]
			if !ok {
				leaves[leaf.Path] = &traceLeaf{
					Value: leaf.Value,
					Log:   []fields.ParseStep{step_},
				}
				continue
			}
			existing.Value = leaf.Value
			existing.Log = append(existing.Log, step_)
			continue
		}
		leaves[leaf.Path] = &traceLeaf{
			Value: leaf.Value,
			Log:   []fields.ParseStep{step_},
		}
	}
	return nil
}

func flattenMap(in map[string]any, prefix []string) []flattenedLeaf {
	if len(in) == 0 {
		return nil
	}
	ret := []flattenedLeaf{}
	for key, value := range in {
		path := append(append([]string(nil), prefix...), key)
		switch typed := value.(type) {
		case map[string]any:
			ret = append(ret, flattenMap(typed, path)...)
		case []any:
			for i, item := range typed {
				itemPath := append(path, fmt.Sprintf("%d", i))
				if itemMap, ok := item.(map[string]any); ok {
					ret = append(ret, flattenMap(itemMap, itemPath)...)
					continue
				}
				ret = append(ret, flattenedLeaf{
					Path:  strings.Join(itemPath, "."),
					Value: item,
				})
			}
		default:
			ret = append(ret, flattenedLeaf{
				Path:  strings.Join(path, "."),
				Value: value,
			})
		}
	}
	return ret
}

func setNestedValue(root map[string]any, path []string, value any) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		root[path[0]] = value
		return
	}
	child, ok := root[path[0]].(map[string]any)
	if !ok {
		child = map[string]any{}
		root[path[0]] = child
	}
	setNestedValue(child, path[1:], value)
}

func buildStructFieldPathMap(t reflect.Type) map[string]string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	ret := map[string]string{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		glazedTag := firstTagComponent(field.Tag.Get("glazed"))
		if glazedTag == "" || glazedTag == "-" || strings.Contains(glazedTag, "*") {
			continue
		}
		yamlTag := firstTagComponent(field.Tag.Get("yaml"))
		if yamlTag == "" {
			yamlTag = strings.ToLower(field.Name)
		}
		ret[glazedTag] = yamlTag
	}
	return ret
}

func firstTagComponent(tag string) string {
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func inferencePathForParsedField(sectionSlug, fieldName string) (string, bool) {
	if strings.HasSuffix(fieldName, "-api-key") {
		if sectionSlug == config.EmbeddingsSlug {
			return "embeddings.api_keys." + fieldName, true
		}
		return "api_keys.api_keys." + fieldName, true
	}
	if strings.HasSuffix(fieldName, "-base-url") {
		if sectionSlug == config.EmbeddingsSlug {
			return "embeddings.base_urls." + fieldName, true
		}
		return "api_keys.base_urls." + fieldName, true
	}

	fieldMap, ok := inferenceSectionFieldPathMap[sectionSlug]
	if !ok {
		return "", false
	}
	path, ok := fieldMap[fieldName]
	if !ok {
		return "", false
	}

	switch sectionSlug {
	case aisettings.AiChatSlug:
		return "chat." + path, true
	case "openai-chat":
		return "openai." + path, true
	case aisettings.AiClientSlug:
		return "client." + path, true
	case "claude-chat":
		return "claude." + path, true
	case "gemini-chat":
		return "gemini." + path, true
	case "ollama-chat":
		return "ollama." + path, true
	case config.EmbeddingsSlug:
		return "embeddings." + path, true
	case aisettings.AiInferenceSlug:
		return "inference." + path, true
	default:
		return "", false
	}
}

func inferenceSettingsToMap(in *aisettings.InferenceSettings) (map[string]any, error) {
	if in == nil {
		return nil, nil
	}
	b, err := yaml.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("marshal inference settings: %w", err)
	}
	if len(b) == 0 {
		return nil, nil
	}
	var out map[string]any
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("decode inference settings as map: %w", err)
	}
	return out, nil
}

func writeRedactedYAML(w io.Writer, value any) error {
	raw, err := yaml.Marshal(value)
	if err != nil {
		return err
	}

	var decoded any
	if err := yaml.Unmarshal(raw, &decoded); err != nil {
		return err
	}

	encoder := yaml.NewEncoder(w)
	defer func() {
		_ = encoder.Close()
	}()
	return encoder.Encode(redactSensitiveValues(decoded, nil))
}

func redactSensitiveValues(value any, path []string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = redactSensitiveValues(item, append(path, key))
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = redactSensitiveValues(item, path)
		}
		return out
	case string:
		if isSensitivePath(path) && strings.TrimSpace(typed) != "" {
			return "***"
		}
		return typed
	default:
		return value
	}
}

func isSensitivePath(path []string) bool {
	if len(path) == 0 {
		return false
	}

	last := strings.ToLower(strings.TrimSpace(path[len(path)-1]))
	if isSensitiveLeafKey(last) {
		return true
	}

	if last != "value" && last != "map-value" {
		return false
	}

	for _, part := range path[:len(path)-1] {
		if isSensitiveLeafKey(strings.ToLower(strings.TrimSpace(part))) {
			return true
		}
	}
	return false
}

func isSensitiveLeafKey(key string) bool {
	return key == "authorization" || strings.HasSuffix(key, "-api-key")
}

func cloneParseSteps(in []fields.ParseStep) []fields.ParseStep {
	if len(in) == 0 {
		return nil
	}
	out := make([]fields.ParseStep, 0, len(in))
	for _, step := range in {
		out = append(out, cloneParseStep(step))
	}
	return out
}

func cloneParseStep(in fields.ParseStep) fields.ParseStep {
	out := fields.ParseStep{
		Source: in.Source,
		Value:  in.Value,
	}
	if len(in.Metadata) > 0 {
		out.Metadata = make(map[string]any, len(in.Metadata))
		for k, v := range in.Metadata {
			out.Metadata[k] = v
		}
	}
	return out
}

package gemini

import (
	"context"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	gepsession "github.com/go-go-golems/geppetto/pkg/inference/session"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/runtimeattrib"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
	moderngenai "google.golang.org/genai"
)

func (e *GeminiEngine) runModernInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	apiType := e.settings.Chat.ApiType
	if apiType == nil {
		return nil, errors.New("no chat api type specified")
	}
	apiKey, ok := e.settings.API.APIKeys[string(*apiType)+"-api-key"]
	if !ok || apiKey == "" {
		return nil, errors.Errorf("missing API key %s", string(*apiType)+"-api-key")
	}
	baseURL := e.settings.API.BaseUrls[string(*apiType)+"-base-url"]
	httpClient, err := settings.EnsureHTTPClient(e.settings.Client)
	if err != nil {
		return nil, errors.Wrap(err, "resolve gemini HTTP client")
	}

	clientConfig := &moderngenai.ClientConfig{
		APIKey:     apiKey,
		Backend:    moderngenai.BackendGeminiAPI,
		HTTPClient: httpClient,
		HTTPOptions: moderngenai.HTTPOptions{
			BaseURL:    baseURL,
			APIVersion: modernGeminiAPIVersion(e.settings),
		},
	}
	client, err := moderngenai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create modern gemini client")
	}

	modelName := *e.settings.Chat.Engine
	config, err := e.buildModernGenerateContentConfig(ctx, t)
	if err != nil {
		return nil, err
	}
	contents, err := buildModernGeminiContentsFromTurn(t)
	if err != nil {
		return nil, err
	}

	registry, _ := tools.RegistryFrom(ctx)
	if registry != nil {
		if hint := buildToolSignatureHint(registry); hint != "" {
			contents = append([]*moderngenai.Content{{Role: string(moderngenai.RoleUser), Parts: []*moderngenai.Part{moderngenai.NewPartFromText(hint)}}}, contents...)
		}
	}

	startTime := time.Now()
	metadata := e.geminiEventMetadata(t, modelName)
	inferenceScopeID := metadata.InferenceID
	if inferenceScopeID == "" {
		inferenceScopeID = metadata.ID.String()
	}
	providerCallIndex := 0
	if idx, ok := gepsession.ProviderCallIndexFromContext(ctx); ok {
		providerCallIndex = idx
	}
	providerCallCorr := geminiProviderCallCorrelation(metadata, inferenceScopeID, modelName, providerCallIndex)
	e.publishEvent(ctx, events.NewProviderCallStartedEvent(metadata, providerCallCorr))

	streamState := newModernGeminiStreamState(providerCallCorr)
	var terminalErr error
	for resp, err := range client.Models.GenerateContentStream(ctx, modelName, contents, config) {
		if err != nil {
			terminalErr = err
			break
		}
		e.publishProviderRecord(ctx, metadata, providerCallCorr, "gemini.modern.stream.chunk", resp)
		for _, event := range reduceModernGeminiResponse(metadata, streamState, resp) {
			e.publishEvent(ctx, event)
		}
	}
	if terminalErr != nil {
		log.Error().Err(terminalErr).Msg("Modern Gemini stream receive failed")
	}

	result, completionEvents := completeModernGeminiStream(t, &metadata, streamState, startTime, terminalErr)
	settings.ApplyModelInfoCost(&result, e.settings.ModelInfo)
	if err := engine.PersistInferenceResult(t, result); err != nil {
		log.Warn().Err(err).Msg("Gemini: failed to persist canonical inference_result")
	}
	for _, event := range completionEvents {
		e.publishEvent(ctx, event)
	}
	if terminalErr != nil {
		return t, terminalErr
	}
	return t, nil
}

func (e *GeminiEngine) geminiEventMetadata(t *turns.Turn, modelName string) events.EventMetadata {
	metadata := events.EventMetadata{
		ID: uuid.New(),
		LLMInferenceData: events.LLMInferenceData{
			Model:       modelName,
			Usage:       nil,
			StopReason:  nil,
			Temperature: e.settings.Chat.Temperature,
			TopP:        e.settings.Chat.TopP,
			MaxTokens:   e.settings.Chat.MaxResponseTokens,
		},
	}
	if t != nil {
		if sid, ok, err := turns.KeyTurnMetaSessionID.Get(t.Metadata); err == nil && ok {
			metadata.SessionID = sid
		}
		if iid, ok, err := turns.KeyTurnMetaInferenceID.Get(t.Metadata); err == nil && ok {
			metadata.InferenceID = iid
		}
		metadata.TurnID = t.ID
	}
	metadata.Extra = map[string]interface{}{}
	metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()
	runtimeattrib.AddRuntimeAttributionToExtra(metadata.Extra, t)
	return metadata
}

func modernGeminiAPIVersion(s *settings.InferenceSettings) string {
	if s != nil && s.Gemini != nil && strings.TrimSpace(s.Gemini.APIVersion) != "" {
		return strings.TrimSpace(s.Gemini.APIVersion)
	}
	return "v1beta"
}

func (e *GeminiEngine) buildModernGenerateContentConfig(ctx context.Context, t *turns.Turn) (*moderngenai.GenerateContentConfig, error) {
	config := &moderngenai.GenerateContentConfig{}
	if e.settings.Chat.Temperature != nil {
		v := float32(*e.settings.Chat.Temperature)
		config.Temperature = &v
	}
	if e.settings.Chat.TopP != nil {
		v := float32(*e.settings.Chat.TopP)
		config.TopP = &v
	}
	if e.settings.Chat.MaxResponseTokens != nil {
		config.MaxOutputTokens = clampIntToInt32(*e.settings.Chat.MaxResponseTokens)
	}
	if len(e.settings.Chat.Stop) > 0 {
		config.StopSequences = append([]string(nil), e.settings.Chat.Stop...)
	}
	if infCfg := engine.ResolveInferenceConfig(t, e.settings.Inference); infCfg != nil {
		if infCfg.Temperature != nil {
			v := float32(*infCfg.Temperature)
			config.Temperature = &v
		}
		if infCfg.TopP != nil {
			v := float32(*infCfg.TopP)
			config.TopP = &v
		}
		if infCfg.MaxResponseTokens != nil {
			config.MaxOutputTokens = clampIntToInt32(*infCfg.MaxResponseTokens)
		}
	}
	if e.settings.Gemini != nil {
		thinking := &moderngenai.ThinkingConfig{}
		setThinking := false
		if e.settings.Gemini.IncludeThoughts != nil {
			thinking.IncludeThoughts = *e.settings.Gemini.IncludeThoughts
			setThinking = true
		}
		if e.settings.Gemini.ThinkingBudget != nil {
			v := clampIntToInt32(*e.settings.Gemini.ThinkingBudget)
			thinking.ThinkingBudget = &v
			setThinking = true
		}
		if strings.TrimSpace(e.settings.Gemini.ThinkingLevel) != "" {
			thinking.ThinkingLevel = moderngenai.ThinkingLevel(strings.TrimSpace(e.settings.Gemini.ThinkingLevel))
			setThinking = true
		}
		if setThinking {
			config.ThinkingConfig = thinking
		}
	}
	registry, _ := tools.RegistryFrom(ctx)
	if registry != nil {
		decls, err := modernGeminiToolDeclarations(registry)
		if err != nil {
			return nil, err
		}
		if len(decls) > 0 {
			config.Tools = []*moderngenai.Tool{{FunctionDeclarations: decls}}
			config.ToolConfig = &moderngenai.ToolConfig{FunctionCallingConfig: &moderngenai.FunctionCallingConfig{Mode: moderngenai.FunctionCallingConfigModeAuto}}
		}
	}
	return config, nil
}

func clampIntToInt32(v int) int32 {
	if v < 0 {
		return 0
	}
	if v > int(math.MaxInt32) {
		return math.MaxInt32
	}
	return int32(int64(v)) // #nosec G115
}

func modernGeminiToolDeclarations(reg tools.ToolRegistry) ([]*moderngenai.FunctionDeclaration, error) {
	if reg == nil {
		return nil, nil
	}
	var out []*moderngenai.FunctionDeclaration
	for _, td := range reg.ListTools() {
		desc := td.Description
		var paramNames []string
		if td.Parameters != nil && td.Parameters.Properties != nil {
			propsVal := reflect.ValueOf(td.Parameters.Properties)
			keysMethod := propsVal.MethodByName("Keys")
			if keysMethod.IsValid() {
				keys := keysMethod.Call(nil)
				if len(keys) == 1 {
					if ks, ok := keys[0].Interface().([]string); ok {
						paramNames = ks
					}
				}
			}
		}
		if len(paramNames) > 0 {
			desc = strings.TrimSpace(desc + " Parameters: " + strings.Join(paramNames, ", "))
		}
		out = append(out, &moderngenai.FunctionDeclaration{
			Name:        td.Name,
			Description: desc,
			Parameters:  convertJSONSchemaToModernGenAI(td.Parameters),
		})
	}
	return out, nil
}

func convertJSONSchemaToModernGenAI(s *jsonschema.Schema) *moderngenai.Schema {
	if s == nil {
		return nil
	}
	gs := &moderngenai.Schema{}
	switch s.Type {
	case "string":
		gs.Type = moderngenai.TypeString
	case "number":
		gs.Type = moderngenai.TypeNumber
	case "integer":
		gs.Type = moderngenai.TypeInteger
	case "boolean":
		gs.Type = moderngenai.TypeBoolean
	case "array":
		gs.Type = moderngenai.TypeArray
	case "object", "":
		gs.Type = moderngenai.TypeObject
		props := modernGeminiSchemaProperties(s)
		if len(props) > 0 {
			gs.Properties = props
		}
	default:
		gs.Type = moderngenai.TypeObject
	}
	return gs
}

func modernGeminiSchemaProperties(s *jsonschema.Schema) map[string]*moderngenai.Schema {
	if s == nil || s.Properties == nil {
		return nil
	}
	propsVal := reflect.ValueOf(s.Properties)
	keysMethod := propsVal.MethodByName("Keys")
	getMethod := propsVal.MethodByName("Get")
	if !keysMethod.IsValid() || !getMethod.IsValid() {
		return nil
	}
	keys := keysMethod.Call(nil)
	if len(keys) != 1 {
		return nil
	}
	ks, ok := keys[0].Interface().([]string)
	if !ok {
		return nil
	}
	ret := map[string]*moderngenai.Schema{}
	for _, k := range ks {
		res := getMethod.Call([]reflect.Value{reflect.ValueOf(k)})
		if len(res) >= 2 && res[1].IsValid() && res[1].Bool() {
			ret[k] = &moderngenai.Schema{Type: moderngenai.TypeString}
		}
	}
	return ret
}

func completeModernGeminiStream(
	t *turns.Turn,
	metadata *events.EventMetadata,
	state *modernGeminiStreamState,
	startedAt time.Time,
	terminalErr error,
) (engine.InferenceResult, []events.Event) {
	if metadata == nil {
		return engine.InferenceResult{}, nil
	}
	if state == nil {
		state = newModernGeminiStreamState(events.Correlation{})
	}
	if terminalErr != nil && strings.TrimSpace(state.finalStopReason) == "" {
		state.finalStopReason = "error"
	}
	out := make([]events.Event, 0, 4)
	if state.reasoningStarted {
		out = append(out, events.NewReasoningSegmentFinishedEventWithSource(*metadata, state.reasoningCorr, "provider", state.reasoning, state.finalStopReason))
	}
	if state.message != "" && state.textSegmentStarted {
		out = append(out, events.NewTextSegmentFinishedEvent(*metadata, state.textCorr, state.message, state.finalStopReason))
	}
	if err := appendModernGeminiStateBlocks(t, state); err != nil {
		terminalErr = err
		if strings.TrimSpace(state.finalStopReason) == "" {
			state.finalStopReason = "error"
		}
	}
	durationMs := time.Since(startedAt).Milliseconds()
	metadata.DurationMs = &durationMs
	if strings.TrimSpace(state.finalStopReason) != "" {
		metadata.StopReason = &state.finalStopReason
	}
	if state.finalUsage != nil {
		metadata.Usage = state.finalUsage
	}
	if len(state.finalUsageExtra) > 0 {
		if metadata.Extra == nil {
			metadata.Extra = map[string]any{}
		}
		for k, v := range state.finalUsageExtra {
			metadata.Extra[k] = v
		}
	}
	hasToolCalls := len(state.pendingCalls) > 0
	result := engine.BuildInferenceResultFromEventMetadata(*metadata, "gemini", hasToolCalls)
	if state.responseID != "" {
		result.ResponseID = state.responseID
	}
	if terminalErr != nil {
		result.FinishClass = engine.InferenceFinishClassError
	}
	for i, c := range state.pendingCalls {
		for j := range t.Blocks {
			if t.Blocks[j].Kind == turns.BlockKindToolCall {
				id, _ := t.Blocks[j].Payload[turns.PayloadKeyID].(string)
				if id == c.id {
					b, err := newModernGeminiToolCallBlock(c, geminiToolCorrelation(state.providerCallCorr, c.id, i))
					if err != nil {
						terminalErr = err
						continue
					}
					t.Blocks[j] = b
				}
			}
		}
	}
	if terminalErr != nil {
		out = append(out, events.NewErrorEvent(*metadata, terminalErr))
	}
	out = append(out, events.NewProviderCallFinishedEvent(
		*metadata,
		state.providerCallCorr,
		state.finalStopReason,
		string(result.FinishClass),
		metadata.Usage,
		metadata.DurationMs,
		hasToolCalls,
	))
	return result, out
}

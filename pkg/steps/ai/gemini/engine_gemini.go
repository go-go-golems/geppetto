package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/runtimeattrib"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	genai "github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GeminiEngine implements the Engine interface for Google's Gemini API
type GeminiEngine struct {
	settings            *settings.InferenceSettings
	observer            geppettoobs.Observer
	observabilityConfig geppettoobs.Config
}

// EngineOption configures a GeminiEngine.
type EngineOption func(*GeminiEngine)

// NewGeminiEngine creates a new Gemini inference engine with the given settings.
func NewGeminiEngine(settings *settings.InferenceSettings, opts ...EngineOption) (*GeminiEngine, error) {
	ret := &GeminiEngine{settings: settings, observabilityConfig: geppettoobs.DefaultConfig()}
	for _, opt := range opts {
		if opt != nil {
			opt(ret)
		}
	}
	ret.observabilityConfig = ret.observabilityConfig.Normalized()
	return ret, nil
}

type geminiAPIKeyTransport struct {
	base   http.RoundTripper
	apiKey string
}

func (t *geminiAPIKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	req2 := req.Clone(req.Context())
	urlCopy := *req.URL
	q := urlCopy.Query()
	if q.Get("key") == "" {
		q.Set("key", t.apiKey)
	}
	urlCopy.RawQuery = q.Encode()
	req2.URL = &urlCopy
	return base.RoundTrip(req2)
}

func geminiHTTPClientWithAPIKey(httpClient *http.Client, apiKey string) *http.Client {
	if httpClient == nil {
		return nil
	}
	clientCopy := *httpClient
	clientCopy.Transport = &geminiAPIKeyTransport{
		base:   httpClient.Transport,
		apiKey: apiKey,
	}
	return &clientCopy
}

func geminiClientOptions(apiKey, baseURL string, httpClient *http.Client) []option.ClientOption {
	opts := []option.ClientOption{option.WithAPIKey(apiKey)}
	if httpClient != nil && httpClient != http.DefaultClient {
		// Keep the API key as an explicit client option even when a custom HTTP
		// client is needed. The upstream SDK strips WithHTTPClient when creating
		// its cache client, so removing WithAPIKey here can accidentally force an
		// ADC lookup despite the resolved inference settings containing a Gemini
		// API key.
		opts = append(opts, option.WithHTTPClient(geminiHTTPClientWithAPIKey(httpClient, apiKey)))
	}
	if baseURL != "" {
		opts = append(opts, option.WithEndpoint(baseURL))
	}
	return opts
}

// convertJSONSchemaToGenAI converts an invopop jsonschema.Schema to a Gemini Schema (best-effort for common types).
func convertJSONSchemaToGenAI(s *jsonschema.Schema) *genai.Schema {
	if s == nil {
		return nil
	}
	gs := &genai.Schema{}
	// Type mapping
	switch s.Type {
	case "string":
		gs.Type = genai.TypeString
	case "number":
		gs.Type = genai.TypeNumber
	case "integer":
		gs.Type = genai.TypeInteger
	case "boolean":
		gs.Type = genai.TypeBoolean
	case "array":
		gs.Type = genai.TypeArray
		// Items optional: skip for now to avoid version differences
	case "object", "":
		// default to object when unspecified
		gs.Type = genai.TypeObject
		// Reflective traversal of Properties ordered map
		propsVal := reflect.ValueOf(s.Properties)
		if propsVal.IsValid() && !propsVal.IsNil() {
			keysMethod := propsVal.MethodByName("Keys")
			getMethod := propsVal.MethodByName("Get")
			if keysMethod.IsValid() && getMethod.IsValid() {
				keys := keysMethod.Call(nil)
				if len(keys) == 1 {
					if ks, ok := keys[0].Interface().([]string); ok {
						resultProps := map[string]*genai.Schema{}
						for _, k := range ks {
							res := getMethod.Call([]reflect.Value{reflect.ValueOf(k)})
							if len(res) >= 2 && res[1].IsValid() && res[1].Bool() {
								// Force simple scalar types to reduce risk of 400s
								resultProps[k] = &genai.Schema{Type: genai.TypeString}
							}
						}
						if len(resultProps) > 0 {
							gs.Properties = resultProps
						}
					}
				}
			}
		}
	default:
		gs.Type = genai.TypeObject
	}
	return gs
}

// RunInference processes a Turn using the Gemini API and appends result blocks.
func (e *GeminiEngine) RunInference(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
	if e.settings == nil || e.settings.Chat == nil || e.settings.Chat.Engine == nil {
		return nil, errors.New("no engine specified")
	}

	apiType_ := e.settings.Chat.ApiType
	if apiType_ == nil {
		return nil, errors.New("no chat api type specified")
	}

	// Build client
	apiKey, ok := e.settings.API.APIKeys[string(*apiType_)+"-api-key"]
	if !ok || apiKey == "" {
		return nil, errors.Errorf("missing API key %s", string(*apiType_)+"-api-key")
	}
	baseURL := e.settings.API.BaseUrls[string(*apiType_)+"-base-url"]
	httpClient, err := settings.EnsureHTTPClient(e.settings.Client)
	if err != nil {
		return nil, errors.Wrap(err, "resolve gemini HTTP client")
	}

	var client *genai.Client
	client, err = genai.NewClient(ctx, geminiClientOptions(apiKey, baseURL, httpClient)...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gemini client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close gemini client")
		}
	}()

	modelName := *e.settings.Chat.Engine
	model := client.GenerativeModel(modelName)

	// Configure generation if present
	if e.settings.Chat.Temperature != nil || e.settings.Chat.TopP != nil || e.settings.Chat.MaxResponseTokens != nil {
		cfg := genai.GenerationConfig{}
		if e.settings.Chat.Temperature != nil {
			v := float32(*e.settings.Chat.Temperature)
			cfg.Temperature = &v
		}
		if e.settings.Chat.TopP != nil {
			v := float32(*e.settings.Chat.TopP)
			cfg.TopP = &v
		}
		if e.settings.Chat.MaxResponseTokens != nil {
			// Clamp to [0, math.MaxInt32] and convert safely to int32
			mt := *e.settings.Chat.MaxResponseTokens
			var v int32
			if mt < 0 {
				log.Warn().Int("requested_max_tokens", mt).Msg("Negative MaxResponseTokens provided; clamping to 0")
				v = 0
			} else if mt > int(math.MaxInt32) {
				log.Warn().Int("requested_max_tokens", mt).Int("clamped_to", int(math.MaxInt32)).Msg("MaxResponseTokens exceeds int32; clamping")
				v = math.MaxInt32
			} else {
				// mt is within int32 range; convert via int64 to avoid int->int32 cast warning linters
				mt64 := int64(mt)
				v = int32(mt64) // #nosec G115
			}
			cfg.MaxOutputTokens = &v
		}
		model.GenerationConfig = cfg
	}

	// Apply per-turn InferenceConfig overrides (Turn.Data > InferenceSettings.Inference).
	if infCfg := engine.ResolveInferenceConfig(t, e.settings.Inference); infCfg != nil {
		if infCfg.Temperature != nil {
			v := float32(*infCfg.Temperature)
			model.Temperature = &v
		}
		if infCfg.TopP != nil {
			v := float32(*infCfg.TopP)
			model.TopP = &v
		}
		if infCfg.MaxResponseTokens != nil {
			mt := *infCfg.MaxResponseTokens
			var v int32
			if mt < 0 {
				v = 0
			} else if mt > int(math.MaxInt32) {
				v = math.MaxInt32
			} else {
				v = int32(int64(mt)) // #nosec G115
			}
			model.MaxOutputTokens = &v
		}
	}

	// Attach tools from context if present (tools + minimal parameters when safe).
	registry, _ := tools.RegistryFrom(ctx)
	if registry != nil {
		var toolDecls []*genai.FunctionDeclaration
		for _, td := range registry.ListTools() {
			fd := &genai.FunctionDeclaration{
				Name: td.Name,
			}
			// Enrich description with parameter names to guide the model
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
			fd.Description = desc
			// Minimal parameters to avoid 400s
			if ps := convertJSONSchemaToGenAI(td.Parameters); ps != nil {
				fd.Parameters = ps
			}
			toolDecls = append(toolDecls, fd)
		}
		if len(toolDecls) > 0 {
			model.Tools = []*genai.Tool{{FunctionDeclarations: toolDecls}}
			log.Debug().Int("gemini_tool_count", len(toolDecls)).Msg("Added tools to Gemini model")
		}
	}
	// Configure function calling mode if tools are present
	// (Removed explicit FunctionCallingConfig to maintain compatibility with SDK version)
	// if registry != nil { ... }

	// Build parts from Turn blocks (includes tool results)
	parts := e.buildPartsFromTurn(t)

	// Prepend a short, explicit tool signature hint to guide argument filling
	if registry != nil {
		if hint := buildToolSignatureHint(registry); hint != "" {
			parts = append([]genai.Part{genai.Text(hint)}, parts...)
		}
	}

	// Prepare metadata for events
	startTime := time.Now()
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
	if metadata.Extra == nil {
		metadata.Extra = map[string]interface{}{}
	}
	metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()
	runtimeattrib.AddRuntimeAttributionToExtra(metadata.Extra, t)

	// Publish provider-call start event. Gemini does not expose a stable
	// response ID before the stream starts, so the provider-call ID is scoped by
	// inference/message metadata and provider-call index.
	inferenceScopeID := metadata.InferenceID
	if inferenceScopeID == "" {
		inferenceScopeID = metadata.ID.String()
	}
	providerCallCorr := geminiProviderCallCorrelation(metadata, inferenceScopeID, modelName)
	e.publishEvent(ctx, events.NewProviderCallStartedEvent(metadata, providerCallCorr))

	// Streaming mode always on for engines in this architecture
	log.Debug().Int("num_blocks", len(t.Blocks)).Str("model", modelName).Msg("Gemini RunInference started (streaming)")
	iter := model.GenerateContentStream(ctx, parts...)

	message := ""
	chunkCount := 0
	finalStopReason := ""
	var finalUsage *events.Usage
	textSegmentStarted := false
	textSequence := int64(0)
	textCorr := events.Correlation{}
	toolCallIndex := 0
	var pendingCalls []struct {
		id, name string
		args     map[string]any
	}
	for {
		resp, err := iter.Next()
		if err == iterator.Done || errors.Is(err, io.EOF) {
			log.Debug().Int("chunks_received", chunkCount).Msg("Gemini stream completed")
			break
		}
		if err != nil {
			log.Error().Err(err).Int("chunks_received", chunkCount).Msg("Gemini stream receive failed")
			d := time.Since(startTime).Milliseconds()
			dm := int64(d)
			metadata.DurationMs = &dm
			e.publishEvent(ctx, events.NewErrorEvent(metadata, err))
			return nil, err
		}
		chunkCount++
		e.publishProviderRecord(ctx, metadata, providerCallCorr, "gemini.stream.chunk", resp)
		// Best-effort: capture provider usage metadata + finish reason so callers can
		// diagnose truncation (max tokens), refusals/safety, etc.
		//
		// We use reflection to avoid tight coupling to a specific generative-ai-go version.
		var chunkUsage *events.Usage
		if resp != nil {
			if u, ok := extractGeminiUsage(resp); ok {
				finalUsage = u
				chunkUsage = u
			}
		}
		// Extract text and function calls
		delta := ""
		chunkStopReason := ""
		if resp != nil && len(resp.Candidates) > 0 {
			for _, cand := range resp.Candidates {
				// Capture finish reason from any candidate that has it (keep the last seen).
				// Extract this BEFORE checking Content, as candidates can have FinishReason
				// even when Content is nil (e.g., safety blocked or empty responses).
				if fr, ok := extractGeminiFinishReason(cand); ok {
					finalStopReason = fr
					chunkStopReason = fr
				}
				if cand.Content == nil {
					continue
				}
				for _, p := range cand.Content.Parts {
					switch v := p.(type) {
					case genai.Text:
						delta += string(v)
					case genai.FunctionCall:
						var args map[string]any
						if v.Args != nil {
							args = v.Args
						}
						if args == nil {
							args = map[string]any{}
						}
						id := uuid.NewString()
						pendingCalls = append(pendingCalls, struct {
							id, name string
							args     map[string]any
						}{id: id, name: v.Name, args: args})
						inputBytes, _ := json.Marshal(args)
						toolCorr := geminiToolCorrelation(providerCallCorr, id, toolCallIndex)
						toolCallIndex++
						e.publishEvent(ctx, events.NewToolCallStartedEvent(metadata, toolCorr, id, v.Name))
						e.publishEvent(ctx, events.NewToolCallRequestedEvent(metadata, toolCorr, id, v.Name, string(inputBytes)))
					}
				}
			}
		}
		if chunkUsage != nil || chunkStopReason != "" {
			e.publishEvent(ctx, events.NewProviderCallMetadataUpdatedEvent(metadata, providerCallCorr, finalStopReason, "", finalUsage))
		}
		if delta != "" {
			if !textSegmentStarted {
				textSegmentStarted = true
				textCorr = geminiSegmentCorrelation(providerCallCorr, "", 0, events.SegmentTypeText)
				e.publishEvent(ctx, events.NewTextSegmentStartedEvent(metadata, textCorr, "assistant"))
			}
			message += delta
			textSequence++
			e.publishEvent(ctx, events.NewTextDeltaEvent(metadata, textCorr, delta, message, textSequence))
		}
	}

	if message != "" && textSegmentStarted {
		e.publishEvent(ctx, events.NewTextSegmentFinishedEvent(metadata, textCorr, message, finalStopReason))
	}

	// Append assistant text and tool_call blocks in the turn
	if message != "" {
		turns.AppendBlock(t, turns.NewAssistantTextBlock(message))
	}
	for _, c := range pendingCalls {
		turns.AppendBlock(t, turns.NewToolCallBlock(c.id, c.name, c.args))
	}

	// Set duration and publish final
	d := time.Since(startTime).Milliseconds()
	dm := int64(d)
	metadata.DurationMs = &dm

	// Populate turn metadata + event metadata (best-effort).
	if strings.TrimSpace(finalStopReason) != "" {
		metadata.StopReason = &finalStopReason
	}
	if finalUsage != nil {
		metadata.Usage = finalUsage
	}
	result := engine.BuildInferenceResultFromEventMetadata(metadata, "gemini", len(pendingCalls) > 0)
	if err := engine.PersistInferenceResult(t, result); err != nil {
		log.Warn().Err(err).Msg("Gemini: failed to persist canonical inference_result")
	}

	if strings.TrimSpace(finalStopReason) != "" || finalUsage != nil {
		e := log.Debug().
			Str("stop_reason", finalStopReason).
			Int("final_text_len", len(message)).
			Int("chunks_received", chunkCount).
			Int("tool_call_count", len(pendingCalls))
		if finalUsage != nil {
			e = e.Int("input_tokens", finalUsage.InputTokens).Int("output_tokens", finalUsage.OutputTokens)
		}
		e.Msg("Gemini RunInference completion metadata")
	}

	finishClass := string(result.FinishClass)
	e.publishEvent(ctx, events.NewProviderCallFinishedEvent(metadata, providerCallCorr, finalStopReason, finishClass, metadata.Usage, metadata.DurationMs, len(pendingCalls) > 0))

	log.Debug().Int("final_text_len", len(message)).Int("tool_call_count", len(pendingCalls)).Msg("Gemini RunInference completed (streaming)")
	return t, nil
}

func geminiProviderCallCorrelation(metadata events.EventMetadata, inferenceScopeID, modelName string) events.Correlation {
	corr := events.BuildProviderCallCorrelation("gemini", inferenceScopeID, "", 0, "")
	corr.SessionID = metadata.SessionID
	corr.TurnID = metadata.TurnID
	corr.Model = modelName
	return corr
}

func geminiSegmentCorrelation(providerCallCorr events.Correlation, providerObjectID string, segmentIndex int, segmentType string) events.Correlation {
	corr := events.BuildSegmentCorrelation(providerCallCorr, providerObjectID, segmentIndex, segmentType)
	corr.SessionID = providerCallCorr.SessionID
	corr.TurnID = providerCallCorr.TurnID
	corr.Model = providerCallCorr.Model
	return corr
}

func geminiToolCorrelation(providerCallCorr events.Correlation, toolCallID string, toolCallIndex int) events.Correlation {
	corr := geminiSegmentCorrelation(providerCallCorr, toolCallID, toolCallIndex, events.SegmentTypeTool)
	idx := int32(toolCallIndex)
	corr.ToolCallID = toolCallID
	corr.ToolCallIndex = &idx
	return corr
}

func extractGeminiFinishReason(c *genai.Candidate) (string, bool) {
	if c == nil {
		return "", false
	}
	v := reflect.ValueOf(c)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	f := v.FieldByName("FinishReason")
	if !f.IsValid() {
		return "", false
	}
	// Some SDK versions use 0 as "unspecified"; keep empty in that case.
	if (f.Kind() == reflect.Int || f.Kind() == reflect.Int32 || f.Kind() == reflect.Int64) && f.Int() == 0 {
		return "", false
	}
	if (f.Kind() == reflect.Uint || f.Kind() == reflect.Uint32 || f.Kind() == reflect.Uint64) && f.Uint() == 0 {
		return "", false
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", f.Interface()))
	if s == "" {
		return "", false
	}
	return s, true
}

func extractGeminiUsage(resp *genai.GenerateContentResponse) (*events.Usage, bool) {
	if resp == nil {
		return nil, false
	}
	v := reflect.ValueOf(resp)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	um := v.FieldByName("UsageMetadata")
	if !um.IsValid() {
		return nil, false
	}
	if um.Kind() == reflect.Ptr {
		if um.IsNil() {
			return nil, false
		}
		um = um.Elem()
	}
	if !um.IsValid() {
		return nil, false
	}

	prompt := int(extractIntField(um, "PromptTokenCount"))
	candidates := int(extractIntField(um, "CandidatesTokenCount"))
	total := int(extractIntField(um, "TotalTokenCount"))

	// We map prompt->input and candidates->output. If candidates is missing but total exists,
	// keep output at 0 rather than guessing.
	if prompt == 0 && candidates == 0 && total == 0 {
		return nil, false
	}
	return &events.Usage{
		InputTokens:  prompt,
		OutputTokens: candidates,
	}, true
}

func extractIntField(v reflect.Value, name string) int64 {
	if !v.IsValid() {
		return 0
	}
	f := v.FieldByName(name)
	if !f.IsValid() {
		return 0
	}
	// NOTE: reflect.Kind is an enum; we intentionally only handle numeric kinds here.
	//nolint:exhaustive
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return f.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		uval := f.Uint()
		// Check for overflow when converting uint64 to int64
		if uval > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(uval)
	default:
		return 0
	}
}

// buildPartsFromTurn converts Turn blocks into a flat slice of genai.Part, including tool results.
func (e *GeminiEngine) buildPartsFromTurn(t *turns.Turn) []genai.Part {
	if t == nil || len(t.Blocks) == 0 {
		return []genai.Part{}
	}
	// Build lookup from tool_call id to name (for FunctionResponse name)
	idToName := map[string]string{}
	for _, b := range t.Blocks {
		if b.Kind == turns.BlockKindToolCall {
			id, _ := b.Payload[turns.PayloadKeyID].(string)
			name, _ := b.Payload[turns.PayloadKeyName].(string)
			if id != "" && name != "" {
				idToName[id] = name
			}
		}
	}

	var parts []genai.Part
	for _, b := range t.Blocks {
		switch b.Kind {
		case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindLLMText, turns.BlockKindOther, turns.BlockKindReasoning:
			if txt, ok := b.Payload[turns.PayloadKeyText]; ok && txt != nil {
				switch sv := txt.(type) {
				case string:
					parts = append(parts, genai.Text(sv))
				case []byte:
					parts = append(parts, genai.Text(string(sv)))
				}
			}

		case turns.BlockKindToolCall:
			parts = append(parts, genai.FunctionCall{
				Name: b.Payload[turns.PayloadKeyName].(string),
				Args: b.Payload[turns.PayloadKeyArgs].(map[string]any),
			})

		case turns.BlockKindToolUse:
			// Add FunctionResponse for tool result
			id, _ := b.Payload[turns.PayloadKeyID].(string)
			res := b.Payload[turns.PayloadKeyResult]
			errStr, _ := b.Payload[turns.PayloadKeyError].(string)
			name := idToName[id]
			var response map[string]any
			switch rv := res.(type) {
			case string:
				// Attempt to parse JSON string into object; if fail, wrap
				var obj map[string]any
				if json.Unmarshal([]byte(rv), &obj) == nil {
					response = obj
				} else {
					response = map[string]any{"result": rv}
				}
			case map[string]any:
				response = rv
			default:
				bts, _ := json.Marshal(rv)
				var obj map[string]any
				if json.Unmarshal(bts, &obj) == nil {
					response = obj
				} else {
					response = map[string]any{"result": rv}
				}
			}
			if errStr != "" {
				response = map[string]any{"error": errStr, "result": response}
			}
			if name == "" {
				name = "result"
			}
			parts = append(parts, genai.FunctionResponse{Name: name, Response: response})
		}
	}
	return parts
}

func buildToolSignatureHint(reg tools.ToolRegistry) string {
	if reg == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("You can call the following tools using function calls with JSON arguments.\n")
	for _, td := range reg.ListTools() {
		b.WriteString("- ")
		b.WriteString(td.Name)
		if td.Description != "" {
			b.WriteString(": ")
			b.WriteString(td.Description)
		}
		// Attempt to extract parameter names
		params := []string{}
		if td.Parameters != nil && td.Parameters.Properties != nil {
			propsVal := reflect.ValueOf(td.Parameters.Properties)
			keysMethod := propsVal.MethodByName("Keys")
			getMethod := propsVal.MethodByName("Get")
			if keysMethod.IsValid() && getMethod.IsValid() {
				keys := keysMethod.Call(nil)
				if len(keys) == 1 {
					if ks, ok := keys[0].Interface().([]string); ok {
						params = append(params, ks...)
					}
				}
			}
		}
		if len(params) > 0 {
			b.WriteString(" Parameters: ")
			b.WriteString(strings.Join(params, ", "))
		}
		// Add explicit examples for common tools
		if td.Name == "get_weather" {
			b.WriteString(" Example: {\"location\": \"London\", \"units\": \"celsius\"}")
		}
		if td.Name == "calculator" {
			b.WriteString(" Example: {\"expression\": \"2 + 2\"}")
		}
		b.WriteString("\n")
	}
	b.WriteString("When appropriate, emit a function call with correctly filled JSON arguments.\n")
	return b.String()
}

// publishEvent publishes an event to all configured sinks and any sinks carried in context.
func (e *GeminiEngine) publishEvent(ctx context.Context, event events.Event) {
	e.publishEventRecord(ctx, event)
	events.PublishEventToContext(ctx, event)
}

var _ engine.Engine = (*GeminiEngine)(nil)

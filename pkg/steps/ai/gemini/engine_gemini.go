package gemini

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
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
	settings *settings.StepSettings
	config   *engine.Config
}

// NewGeminiEngine creates a new Gemini inference engine with the given settings and options.
func NewGeminiEngine(settings *settings.StepSettings, options ...engine.Option) (*GeminiEngine, error) {
	cfg := engine.NewConfig()
	if err := engine.ApplyOptions(cfg, options...); err != nil {
		return nil, err
	}
	return &GeminiEngine{settings: settings, config: cfg}, nil
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

	var client *genai.Client
	var err error
	if baseURL != "" {
		client, err = genai.NewClient(ctx, option.WithAPIKey(apiKey), option.WithEndpoint(baseURL))
	} else {
		client, err = genai.NewClient(ctx, option.WithAPIKey(apiKey))
	}
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
			mt := *e.settings.Chat.MaxResponseTokens
			if mt < 0 {
				log.Warn().Int("requested_max_tokens", mt).Msg("Negative MaxResponseTokens provided; clamping to 0")
				mt = 0
			}
			if mt > int(math.MaxInt32) {
				log.Warn().Int("requested_max_tokens", mt).Int("clamped_to", int(math.MaxInt32)).Msg("MaxResponseTokens exceeds int32; clamping")
				mt = int(math.MaxInt32)
			}
			v := int32(mt)
			cfg.MaxOutputTokens = &v
		}
		model.GenerationConfig = cfg
	}

	// Attach tools from Turn.Data if present (tools + minimal parameters when safe)
	var registry tools.ToolRegistry
	if t != nil && t.Data != nil {
		if regAny, ok := t.Data[turns.DataKeyToolRegistry]; ok && regAny != nil {
			if reg, ok := regAny.(tools.ToolRegistry); ok && reg != nil {
				registry = reg
				var toolDecls []*genai.FunctionDeclaration
				for _, td := range reg.ListTools() {
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
		metadata.RunID = t.RunID
		metadata.TurnID = t.ID
	}
	if metadata.Extra == nil {
		metadata.Extra = map[string]interface{}{}
	}
	metadata.Extra[events.MetadataSettingsSlug] = e.settings.GetMetadata()

	// Publish start event
	e.publishEvent(ctx, events.NewStartEvent(metadata))

	// Streaming mode always on for engines in this architecture
	log.Debug().Int("num_blocks", len(t.Blocks)).Str("model", modelName).Msg("Gemini RunInference started (streaming)")
	iter := model.GenerateContentStream(ctx, parts...)

	message := ""
	chunkCount := 0
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
		// Extract text and function calls
		delta := ""
		if resp != nil && len(resp.Candidates) > 0 {
			for _, cand := range resp.Candidates {
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
						// Publish ToolCall event with JSON string input
						inputBytes, _ := json.Marshal(args)
						e.publishEvent(ctx, events.NewToolCallEvent(metadata, events.ToolCall{ID: id, Name: v.Name, Input: string(inputBytes)}))
					}
				}
			}
		}
		if delta != "" {
			message += delta
			e.publishEvent(ctx, events.NewPartialCompletionEvent(metadata, delta, message))
		}
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
	e.publishEvent(ctx, events.NewFinalEvent(metadata, message))

	log.Debug().Int("final_text_len", len(message)).Int("tool_call_count", len(pendingCalls)).Msg("Gemini RunInference completed (streaming)")
	return t, nil
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
		case turns.BlockKindUser, turns.BlockKindSystem, turns.BlockKindLLMText, turns.BlockKindOther:
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
	for _, sink := range e.config.EventSinks {
		if err := sink.PublishEvent(event); err != nil {
			log.Warn().Err(err).Str("event_type", string(event.Type())).Msg("Failed to publish event to sink")
		}
	}
	events.PublishEventToContext(ctx, event)
}

var _ engine.Engine = (*GeminiEngine)(nil)

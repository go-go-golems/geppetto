package gemini

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/inference/tools"
	geppettoobs "github.com/go-go-golems/geppetto/pkg/observability"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/turns"
	genai "github.com/google/generative-ai-go/genai"
	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
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

	if e.settings.Chat.ApiType == nil {
		return nil, errors.New("no chat api type specified")
	}

	return e.runModernInference(ctx, t)
}

func geminiProviderCallCorrelation(metadata events.EventMetadata, inferenceScopeID, _ string, providerCallIndex int) events.Correlation {
	corr := events.BuildProviderCallCorrelation("gemini", inferenceScopeID, "", providerCallIndex, "")
	corr.SessionID = metadata.SessionID
	corr.TurnID = metadata.TurnID
	return corr
}

func geminiSegmentCorrelation(providerCallCorr events.Correlation, providerObjectID string, segmentIndex int, segmentType string) events.Correlation {
	corr := events.BuildSegmentCorrelation(providerCallCorr, providerObjectID, segmentIndex, segmentType)
	corr.SessionID = providerCallCorr.SessionID
	corr.TurnID = providerCallCorr.TurnID
	return corr
}

func geminiToolCorrelation(providerCallCorr events.Correlation, toolCallID string, toolCallIndex int) events.Correlation {
	corr := geminiSegmentCorrelation(providerCallCorr, toolCallID, toolCallIndex, events.SegmentTypeTool)
	corr.ToolCallID = toolCallID
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

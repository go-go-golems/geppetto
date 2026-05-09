package gemini

import (
	"errors"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/inference/engine"
	"github.com/go-go-golems/geppetto/pkg/turns"
	genai "github.com/google/generative-ai-go/genai"
	"github.com/invopop/jsonschema"
	"google.golang.org/api/option"
)

func TestConvertJSONSchemaToGenAI_ObjectType(t *testing.T) {
	s := &jsonschema.Schema{Type: "object"}
	gs := convertJSONSchemaToGenAI(s)
	if gs == nil {
		t.Fatalf("convertJSONSchemaToGenAI returned nil")
	}
	if gs.Type != genai.TypeObject {
		t.Fatalf("expected TypeObject, got %v", gs.Type)
	}
}

func TestConvertJSONSchemaToGenAI_ScalarTypes(t *testing.T) {
	cases := []struct {
		name     string
		inType   string
		expected genai.Type
	}{
		{"string", "string", genai.TypeString},
		{"number", "number", genai.TypeNumber},
		{"integer", "integer", genai.TypeInteger},
		{"boolean", "boolean", genai.TypeBoolean},
		{"array", "array", genai.TypeArray},
		{"object", "object", genai.TypeObject},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &jsonschema.Schema{Type: tc.inType}
			gs := convertJSONSchemaToGenAI(s)
			if gs == nil {
				t.Fatalf("nil result for %s", tc.inType)
			}
			if gs.Type != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, gs.Type)
			}
		})
	}
}

type geminiRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f geminiRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGeminiHTTPClientWithAPIKey_AppendsKeyQueryParam(t *testing.T) {
	baseClient := &http.Client{
		Transport: geminiRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.URL.Query().Get("key"); got != "test-key" {
				t.Fatalf("expected key query param to be injected, got %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
				Header:     http.Header{},
				Request:    req,
			}, nil
		}),
	}

	client := geminiHTTPClientWithAPIKey(baseClient, "test-key")
	req, err := http.NewRequest(http.MethodGet, "https://example.test/v1/models", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := client.Do(req); err != nil {
		t.Fatalf("client.Do: %v", err)
	}
}

func TestGeminiClientOptions_DefaultClientIncludesAPIKey(t *testing.T) {
	opts := geminiClientOptions("test-key", "", http.DefaultClient)
	assertGeminiClientOptions(t, opts, true, false, false)
}

func TestGeminiClientOptions_CustomClientKeepsAPIKeyAndHTTPClient(t *testing.T) {
	httpClient := &http.Client{Transport: http.DefaultTransport}
	opts := geminiClientOptions("test-key", "https://example.test", httpClient)
	assertGeminiClientOptions(t, opts, true, true, true)
}

func TestGeminiCanonicalCorrelationHelpersValidate(t *testing.T) {
	metadata := events.EventMetadata{
		SessionID:   "session-1",
		InferenceID: "inference-1",
		TurnID:      "turn-1",
	}

	providerCorr := geminiProviderCallCorrelation(metadata, metadata.InferenceID, "gemini-test", 0)
	if err := events.ValidateCanonicalEvent(events.NewProviderCallStartedEvent(metadata, providerCorr)); err != nil {
		t.Fatalf("provider-call correlation should validate: %v", err)
	}

	textCorr := geminiSegmentCorrelation(providerCorr, "", 0, events.SegmentTypeText)
	if err := events.ValidateCanonicalEvent(events.NewTextDeltaEvent(metadata, textCorr, "hi", "hi", 1)); err != nil {
		t.Fatalf("text correlation should validate: %v", err)
	}

	toolCorr := geminiToolCorrelation(providerCorr, "tool-1", 0)
	if err := events.ValidateCanonicalEvent(events.NewToolCallRequestedEvent(metadata, toolCorr, "tool-1", "lookup", `{"q":"x"}`)); err != nil {
		t.Fatalf("tool correlation should validate: %v", err)
	}
}

func TestGeminiProviderCallCorrelationUsesProviderCallIndex(t *testing.T) {
	metadata := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}

	for _, tt := range []struct {
		name              string
		providerCallIndex int
		wantProviderID    string
	}{
		{name: "first provider call", providerCallIndex: 0, wantProviderID: "gemini:inference-1:provider-call:0"},
		{name: "third provider call", providerCallIndex: 2, wantProviderID: "gemini:inference-1:provider-call:2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			corr := geminiProviderCallCorrelation(metadata, metadata.InferenceID, "gemini-test", tt.providerCallIndex)
			if corr.ProviderCallIndex != int32(tt.providerCallIndex) {
				t.Fatalf("ProviderCallIndex = %d, want %d", corr.ProviderCallIndex, tt.providerCallIndex)
			}
			if corr.ProviderCallID != tt.wantProviderID {
				t.Fatalf("ProviderCallID = %q, want %q", corr.ProviderCallID, tt.wantProviderID)
			}
			if corr.CorrelationKey != tt.wantProviderID {
				t.Fatalf("CorrelationKey = %q, want %q", corr.CorrelationKey, tt.wantProviderID)
			}
			if corr.Model != "gemini-test" {
				t.Fatalf("Model = %q, want gemini-test", corr.Model)
			}
			if corr.TurnID != metadata.TurnID {
				t.Fatalf("TurnID = %q, want %q", corr.TurnID, metadata.TurnID)
			}
		})
	}
}

func TestReduceGeminiStreamResponseReviewDerivedScenarios(t *testing.T) {
	metadata := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	providerCorr := geminiProviderCallCorrelation(metadata, metadata.InferenceID, "gemini-test", 0)

	tests := []struct {
		name      string
		chunks    []*genai.GenerateContentResponse
		wantTypes []events.EventType
		check     func(t *testing.T, state *geminiStreamState, got []events.Event)
	}{
		{
			name: "metadata-only final chunk does not create text segment",
			chunks: []*genai.GenerateContentResponse{{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
				UsageMetadata: &genai.UsageMetadata{
					PromptTokenCount:     2,
					CandidatesTokenCount: 3,
				},
			}},
			wantTypes: []events.EventType{events.EventTypeProviderCallMetadataUpdated},
			check: func(t *testing.T, state *geminiStreamState, got []events.Event) {
				t.Helper()
				if state.textSegmentStarted || state.message != "" {
					t.Fatalf("metadata-only chunk created text state: started=%v message=%q", state.textSegmentStarted, state.message)
				}
				if state.finalStopReason != "FinishReasonStop" {
					t.Fatalf("stop reason = %q, want FinishReasonStop", state.finalStopReason)
				}
				if state.finalUsage == nil || state.finalUsage.InputTokens != 2 || state.finalUsage.OutputTokens != 3 {
					t.Fatalf("usage = %#v, want input=2 output=3", state.finalUsage)
				}
			},
		},
		{
			name: "multiple text chunks accumulate monotonically",
			chunks: []*genai.GenerateContentResponse{
				geminiTextResponse("hello "),
				geminiTextResponse("world"),
			},
			wantTypes: []events.EventType{
				events.EventTypeTextSegmentStarted,
				events.EventTypeTextDelta,
				events.EventTypeTextDelta,
			},
			check: func(t *testing.T, state *geminiStreamState, got []events.Event) {
				t.Helper()
				if state.message != "hello world" {
					t.Fatalf("message = %q, want hello world", state.message)
				}
				lastDelta, ok := got[len(got)-1].(*events.EventTextDelta)
				if !ok || lastDelta.Text != "hello world" || lastDelta.Sequence != 2 {
					t.Fatalf("last delta = %#v, want accumulated text with sequence 2", got[len(got)-1])
				}
			},
		},
		{
			name: "complete function call emits executable tool request",
			chunks: []*genai.GenerateContentResponse{{
				Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []genai.Part{genai.FunctionCall{Name: "lookup", Args: map[string]any{"q": "x"}}}}}},
			}},
			wantTypes: []events.EventType{
				events.EventTypeToolCallStarted,
				events.EventTypeToolCallRequested,
			},
			check: func(t *testing.T, state *geminiStreamState, got []events.Event) {
				t.Helper()
				if len(state.pendingCalls) != 1 || state.pendingCalls[0].name != "lookup" {
					t.Fatalf("pending calls = %#v, want one lookup", state.pendingCalls)
				}
				requested, ok := got[1].(*events.EventToolCallRequested)
				if !ok {
					t.Fatalf("event[1] = %#v, want tool call requested", got[1])
				}
				if requested.ToolName != "lookup" || requested.Input != `{"q":"x"}` {
					t.Fatalf("requested = (%q,%q), want lookup JSON args", requested.ToolName, requested.Input)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newGeminiStreamState(providerCorr)
			var got []events.Event
			for _, chunk := range tt.chunks {
				got = append(got, reduceGeminiStreamResponse(metadata, state, chunk)...)
			}
			assertGeminiEventTypes(t, got, tt.wantTypes)
			for i, ev := range got {
				if err := events.ValidateCanonicalEvent(ev); err != nil {
					t.Fatalf("event[%d] %s failed canonical validation: %v", i, ev.Type(), err)
				}
			}
			if tt.check != nil {
				tt.check(t, state, got)
			}
		})
	}
}

func TestCompleteGeminiStreamTerminalErrorClosesActiveText(t *testing.T) {
	metadata := events.EventMetadata{SessionID: "session-1", InferenceID: "inference-1", TurnID: "turn-1"}
	providerCorr := geminiProviderCallCorrelation(metadata, metadata.InferenceID, "gemini-test", 0)
	state := newGeminiStreamState(providerCorr)

	chunkEvents := reduceGeminiStreamResponse(metadata, state, geminiTextResponse("partial"))
	assertGeminiEventTypes(t, chunkEvents, []events.EventType{
		events.EventTypeTextSegmentStarted,
		events.EventTypeTextDelta,
	})

	turn := &turns.Turn{ID: "turn-1"}
	terminalErr := errors.New("stream exploded")
	result, completionEvents := completeGeminiStream(turn, &metadata, state, time.Now(), terminalErr)

	assertGeminiEventTypes(t, completionEvents, []events.EventType{
		events.EventTypeTextSegmentFinished,
		events.EventTypeError,
		events.EventTypeProviderCallFinished,
	})
	if result.FinishClass != engine.InferenceFinishClassError {
		t.Fatalf("finish class = %q, want error", result.FinishClass)
	}
	if metadata.StopReason == nil || *metadata.StopReason != "error" {
		t.Fatalf("stop reason = %#v, want error", metadata.StopReason)
	}
	finished, ok := completionEvents[0].(*events.EventTextSegmentFinished)
	if !ok || finished.Text != "partial" || finished.FinishReason != "error" {
		t.Fatalf("finished event = %#v, want partial text with error reason", completionEvents[0])
	}
	providerFinished, ok := completionEvents[2].(*events.EventProviderCallFinished)
	if !ok || providerFinished.FinishClass != string(engine.InferenceFinishClassError) || providerFinished.StopReason != "error" {
		t.Fatalf("provider finished = %#v, want error/error", completionEvents[2])
	}
	if len(turn.Blocks) != 1 || turn.Blocks[0].Kind != turns.BlockKindLLMText {
		t.Fatalf("turn blocks = %#v, want one assistant text block", turn.Blocks)
	}
}

func geminiTextResponse(text string) *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []genai.Part{genai.Text(text)}}}}}
}

func assertGeminiEventTypes(t *testing.T, got []events.Event, want []events.EventType) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("event count = %d, want %d; got=%#v", len(got), len(want), got)
	}
	for i, wantType := range want {
		if got[i].Type() != wantType {
			t.Fatalf("event[%d] type = %s, want %s", i, got[i].Type(), wantType)
		}
	}
}

func TestGeminiEngineDoesNotCallLegacyEventConstructors(t *testing.T) {
	b, err := os.ReadFile("engine_gemini.go")
	if err != nil {
		t.Fatalf("read engine_gemini.go: %v", err)
	}
	src := string(b)
	for _, forbidden := range []string{
		"New" + "StartEvent(",
		"New" + "PartialCompletionEvent(",
		"New" + "FinalEvent(",
		"New" + "ThinkingPartialEvent(",
		"New" + "ToolCallEvent(",
	} {
		if strings.Contains(src, forbidden) {
			t.Fatalf("Gemini engine still calls legacy event constructor %s", forbidden)
		}
	}
}

func assertGeminiClientOptions(t *testing.T, opts []option.ClientOption, wantAPIKey, wantHTTPClient, wantEndpoint bool) {
	t.Helper()

	var hasAPIKey, hasHTTPClient, hasEndpoint bool
	for _, opt := range opts {
		switch reflect.TypeOf(opt).String() {
		case "option.withAPIKey":
			hasAPIKey = true
		case "option.withHTTPClient":
			hasHTTPClient = true
		case "option.withEndpoint":
			hasEndpoint = true
		}
	}

	if hasAPIKey != wantAPIKey {
		t.Fatalf("withAPIKey mismatch: got=%v want=%v", hasAPIKey, wantAPIKey)
	}
	if hasHTTPClient != wantHTTPClient {
		t.Fatalf("withHTTPClient mismatch: got=%v want=%v", hasHTTPClient, wantHTTPClient)
	}
	if hasEndpoint != wantEndpoint {
		t.Fatalf("withEndpoint mismatch: got=%v want=%v", hasEndpoint, wantEndpoint)
	}
}

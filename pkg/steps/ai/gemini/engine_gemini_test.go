package gemini

import (
	"net/http"
	"reflect"
	"testing"

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

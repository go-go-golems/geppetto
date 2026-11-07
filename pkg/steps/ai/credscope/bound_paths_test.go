package credscope_test

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/decision/systemone"
	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/rerank"
	cohere "github.com/go-go-golems/geppetto/pkg/rerank/cohere"
	"github.com/go-go-golems/geppetto/pkg/rerank/llamacpp"
	"github.com/go-go-golems/geppetto/pkg/rerank/llamacpp/qwen3"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	claudeapi "github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai_responses"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/providers/geminisdk"
	aitransport "github.com/go-go-golems/geppetto/pkg/steps/ai/transport"
)

func boundPath(kind string) func(*testing.T, *http.Client) error {
	return func(t *testing.T, client *http.Client) error {
		routes := connection.Routes{"embed": connection.Relative("embeddings")}
		switch kind {
		case "cohere embed":
			routes = embeddings.CohereRoutes()
		case "ollama embed":
			routes = embeddings.OllamaRoutes()
		case "cohere rerank":
			routes = cohere.Routes()
		case "llama rerank":
			routes = llamacpp.Routes()
		case "qwen rerank":
			routes = qwen3.Routes()
		case "systemone":
			routes = systemone.Routes()
		case "claude count", "claude messages":
			routes = claudeapi.Routes()
		case "responses count":
			routes = openai_responses.Routes()
		case "whisper":
			routes = openai.Routes()
		}
		auth := connection.StaticBearer(secret)
		if kind == "claude count" || kind == "claude messages" {
			auth = connection.StaticToken(secret, connection.TokenHeaders{"x-api-key": ""})
		}
		if kind == "gemini cache" {
			auth = connection.StaticToken(secret, connection.TokenHeaders{"x-goog-api-key": ""})
		}
		c, err := connection.New(connection.Config{Provider: "matrix", BaseURL: gw, Routes: routes, Auth: auth, HTTPClient: client})
		if err != nil {
			return err
		}
		switch kind {
		case "openai embed", "cohere embed", "ollama embed":
			var p embeddings.Provider
			switch kind {
			case "openai embed":
				p, err = embeddings.NewOpenAIProvider(c, "text-embedding-3-small", 2)
			case "cohere embed":
				p, err = embeddings.NewCohereProvider(c, "embedding", 2)
			case "ollama embed":
				p, err = embeddings.NewOllamaProvider(c, "embedding", 2)
			}
			if err == nil {
				_, err = p.GenerateEmbedding(t.Context(), "test")
			}
		case "cohere rerank", "llama rerank", "qwen rerank":
			var p rerank.Provider
			switch kind {
			case "cohere rerank":
				p, err = cohere.New(cohere.Options{Connection: c, Model: "rerank"})
			case "llama rerank":
				p, err = llamacpp.New(llamacpp.Options{Connection: c, Model: "rerank"})
			case "qwen rerank":
				p, err = qwen3.New(qwen3.Options{Connection: c, Model: "rerank"})
			}
			if err == nil {
				_, err = p.Rerank(t.Context(), rerank.Request{Query: "test", Documents: []rerank.Document{{ID: "1", Text: "test"}}, TopN: 1})
			}
		case "systemone":
			p, e := systemone.New(systemone.Options{Connection: c, Model: "djev"})
			err = e
			if err == nil {
				_, err = p.Decide(t.Context(), decision.Request{State: "test", Questions: map[string]decision.Question{"q": {Type: decision.Noul, Instructions: "?"}}})
			}
		case "claude messages":
			_, err = claudeapi.NewClient(c).SendMessage(t.Context(), &claudeapi.MessageRequest{Model: "test"})
		case "claude count":
			counter, e := claude.NewTokenCounter(c, newSettings(t, client))
			err = e
			if err == nil {
				_, err = counter.CountTurn(t.Context(), userTurn())
			}
		case "responses count":
			counter, e := openai_responses.NewTokenCounter(c, newSettings(t, client))
			err = e
			if err == nil {
				_, err = counter.CountTurn(t.Context(), userTurn())
			}
		case "whisper":
			p, e := openai.NewTranscriptionClient(c)
			err = e
			if err != nil {
				return err
			}
			file := filepath.Join(t.TempDir(), "test.mp3")
			if err = os.WriteFile(file, []byte("ID3dummy"), 0o600); err != nil {
				return err
			}
			result, e := p.TranscribeFile(t.Context(), file)
			err = e
			if err == nil && result != nil {
				err = result.Error
			}
		case "gemini cache":
			p, e := geminisdk.NewClient(t.Context(), c, "v1beta")
			err = e
			if err == nil {
				_, err = p.Caches.Get(t.Context(), "cachedContents/test", nil)
			}
		}
		return err
	}
}

func additionalBoundRows() []row {
	var rows []row
	for _, kind := range []string{"openai embed", "cohere embed", "ollama embed", "cohere rerank", "llama rerank", "qwen rerank", "systemone", "claude messages", "claude count", "responses count", "whisper", "gemini cache"} {
		rows = append(rows, row{name: "bound " + kind, run: boundPath(kind), allowed: []string{"gateway.example"}})
		rows = append(rows, row{name: "bound " + kind + " + redirect", run: boundPath(kind), allowed: []string{"gateway.example"}, redirectFrom: "gateway.example"})
	}
	rows = append(rows, row{name: "responses route cannot change credential origin", run: func(t *testing.T, client *http.Client) error {
		s := newSettings(t, client)
		s.API.APIKeys["open-responses-api-key"] = secret
		s.API.BaseUrls["open-responses-base-url"] = gw
		c, err := connection.New(connection.Config{Provider: "matrix", BaseURL: gw, Routes: connection.Routes{"responses": foreignRoute{}}, Auth: connection.StaticBearer(secret), HTTPClient: client})
		if err == nil {
			_, err = c.Resolve("responses")
		}
		return err
	}})
	rows = append(rows, row{name: "embedding-local key cannot inherit parent URL", run: func(t *testing.T, client *http.Client) error {
		s := newSettings(t, client)
		s.API.APIKeys["openai-api-key"] = secret
		s.API.BaseUrls["openai-base-url"] = gw
		s.Embeddings.APIKeys["openai-api-key"] = secret
		_, err := embeddings.NewSettingsFactoryFromInferenceSettings(s).NewProvider()
		return err
	}})
	return rows
}

type foreignRoute struct{}

func (foreignRoute) Resolve(aitransport.RouteRequest) (*url.URL, error) {
	return url.Parse("https://other.example/responses")
}

// Keep a source-level interface assertion for the trusted route extension.
var _ aitransport.RouteResolver = foreignRoute{}

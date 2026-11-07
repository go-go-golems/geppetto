package connection_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/types"
)

func ExampleConnection() {
	c, err := connection.New(connection.Config{
		Provider: "gateway", BaseURL: "https://gateway.example/v1",
		Routes: connection.Routes{"models": connection.Relative("models")},
		Auth:   connection.StaticBearer("example-token"), Headers: http.Header{"X-Team": {"example"}},
		// An application supplies its timeout/proxy/TLS client here. This example
		// records the request in process instead of contacting a real provider.
		HTTPClient: &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: http.NoBody, Request: req}, nil
		})},
	})
	if err != nil {
		panic(err)
	}
	req, err := c.NewRequest(context.Background(), http.MethodGet, "models", nil)
	if err != nil {
		panic(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer func() { _ = resp.Body.Close() }()
	fmt.Println(resp.StatusCode)
	// Output: 200
}

func ExampleConnection_sharing() {
	c, err := connection.New(connection.Config{
		Provider: "openai", BaseURL: "https://gateway.example/v1",
		Routes: openai.Routes(), Auth: connection.StaticBearer("example-token"),
	})
	if err != nil {
		panic(err)
	}
	s, err := settings.NewInferenceSettings()
	if err != nil {
		panic(err)
	}
	model := "example-chat"
	protocol := types.ApiTypeOpenAI
	s.Chat.Engine, s.Chat.ApiType = &model, &protocol
	chat, err := openai.NewOpenAIEngine(c, s)
	if err != nil {
		panic(err)
	}
	embed, err := embeddings.NewOpenAIProvider(c, "text-embedding-3-small", 1536)
	if err != nil {
		panic(err)
	}
	audio, err := openai.NewTranscriptionClient(c)
	if err != nil {
		panic(err)
	}
	// Constructors share the authority; no request is made until an operation runs.
	fmt.Println(chat != nil, embed != nil, audio != nil)
	// Output: true true true
}

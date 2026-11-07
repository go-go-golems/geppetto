// Package geminisdk constructs modern Gemini SDK clients with one bound HTTP
// client for every service, including Models, Caches, Files and token counting.
package geminisdk

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/pkg/errors"
	"google.golang.org/genai"
)

func NewClient(ctx context.Context, c *connection.Connection, apiVersion string) (*genai.Client, error) {
	if c == nil {
		return nil, errors.New("Gemini SDK requires a bound API connection")
	}
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: connection.SDKAPIKey, Backend: genai.BackendGeminiAPI,
		HTTPClient: c.SDKClient(), HTTPOptions: genai.HTTPOptions{BaseURL: c.BaseURL(), APIVersion: apiVersion},
	})
}

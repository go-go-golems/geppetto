// Package umans provides the verified Umans Anthropic-Messages authentication
// binding. It is API-key based, not renewable OAuth.
package umans

import (
	"errors"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
)

// ClaudeOptions returns Go-only Claude engine options for an Umans API key.
// The key is sent in the verified dual-auth form by the Claude API client and
// never becomes an inference setting or JavaScript value.
func ClaudeOptions(apiKey string) ([]claude.EngineOption, error) {
	source, err := credentials.NewStaticBearerTokenSource(apiKey)
	if err != nil {
		return nil, errors.New("umans API key is required")
	}
	return []claude.EngineOption{claude.WithBearerTokenSource(source)}, nil
}

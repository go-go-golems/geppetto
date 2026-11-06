// Package openaisdk constructs OpenAI SDK clients exclusively from an already
// bound API connection. It is shared by embeddings and audio transcription.
package openaisdk

import (
	"github.com/go-go-golems/geppetto/pkg/steps/ai/connection"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
)

func NewClient(c *connection.Connection) (*openai.Client, error) {
	if c == nil {
		return nil, errors.New("OpenAI SDK requires a bound API connection")
	}
	cfg := openai.DefaultConfig("")
	cfg.BaseURL = c.BaseURL()
	cfg.HTTPClient = c.SDKClient()
	return openai.NewClientWithConfig(cfg), nil
}

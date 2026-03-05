package tokencount

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

type Source string

const (
	SourceProviderAPI Source = "provider_api"
	SourceEstimate    Source = "estimate"
)

type Result struct {
	Provider    string `json:"provider" yaml:"provider"`
	Model       string `json:"model" yaml:"model"`
	InputTokens int    `json:"input_tokens" yaml:"input_tokens"`
	Source      Source `json:"source" yaml:"source"`
	Endpoint    string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	RequestKind string `json:"request_kind,omitempty" yaml:"request_kind,omitempty"`
}

type Counter interface {
	CountTurn(ctx context.Context, t *turns.Turn) (*Result, error)
}

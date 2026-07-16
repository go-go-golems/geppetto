package credentials

import (
	"context"
	"errors"
	"strings"
)

// StaticBearerTokenSource adapts an explicitly supplied provider key to the
// request-time source interface. It retains no provider metadata and is useful
// for static-key gateways such as Umans. The caller remains responsible for
// keeping the value out of settings, logs, and JavaScript.
type StaticBearerTokenSource struct {
	value string
}

// NewStaticBearerTokenSource creates a source for one explicit key.
func NewStaticBearerTokenSource(value string) (*StaticBearerTokenSource, error) {
	if strings.TrimSpace(value) == "" {
		return nil, errors.New("static bearer credential is required")
	}
	return &StaticBearerTokenSource{value: value}, nil
}

func (s *StaticBearerTokenSource) BearerToken(context.Context, Request) (string, error) {
	if s == nil || strings.TrimSpace(s.value) == "" {
		return "", &ErrUnavailable{Operation: "static credential lookup"}
	}
	return s.value, nil
}

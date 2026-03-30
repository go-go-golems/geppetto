package parsehelpers

import (
	"bytes"
	"errors"
	"strings"
	"time"

	yamlsanitize "github.com/go-go-golems/sanitize/pkg/yaml"
	"gopkg.in/yaml.v3"
)

var (
	errEmptyBody       = errors.New("empty")
	errPayloadTooLarge = errors.New("payload too large")
)

// StripCodeFenceBytes detects ```lang\n ... \n``` blocks and returns (lang, body).
// If no fence is detected, returns ("", originalBytes).
func StripCodeFenceBytes(b []byte) (string, []byte) {
	s := string(b)
	idx := strings.Index(s, "```")
	if idx < 0 {
		return "", b
	}
	rest := s[idx+3:]
	nl := strings.IndexByte(rest, '\n')
	if nl < 0 {
		return "", b
	}
	header := strings.TrimSpace(rest[:nl])
	body := rest[nl+1:]
	end := strings.LastIndex(body, "```")
	if end >= 0 {
		body = body[:end]
	}
	return strings.ToLower(header), []byte(body)
}

type DebounceConfig struct {
	SnapshotEveryBytes int
	SnapshotOnNewline  bool
	ParseTimeout       time.Duration
	MaxBytes           int
	SanitizeYAML       *bool `json:"sanitize_yaml,omitempty" yaml:"sanitize_yaml,omitempty"`
}

func (c DebounceConfig) withDefaults() DebounceConfig {
	if c.SanitizeYAML != nil {
		return c
	}
	return c.WithSanitizeYAML(true)
}

func (c DebounceConfig) SanitizeEnabled() bool {
	c = c.withDefaults()
	return c.SanitizeYAML != nil && *c.SanitizeYAML
}

func (c DebounceConfig) WithSanitizeYAML(v bool) DebounceConfig {
	ret := c
	ret.SanitizeYAML = new(bool)
	*ret.SanitizeYAML = v
	return ret
}

// YAMLController incrementally accumulates bytes and attempts to parse typed YAML
// on a cadence (newline or every N bytes). T should be a struct type with yaml tags.
type YAMLController[T any] struct {
	cfg              DebounceConfig
	buf              bytes.Buffer
	sinceLastAttempt int
}

func NewDebouncedYAML[T any](cfg DebounceConfig) *YAMLController[T] {
	return &YAMLController[T]{cfg: cfg.withDefaults()}
}

// FeedBytes appends chunk; if cadence triggers, attempts to parse and returns a snapshot.
// On parse error, returns (nil, err). Callers may ignore errors until a future success.
func (c *YAMLController[T]) FeedBytes(chunk []byte) (*T, error) {
	if len(chunk) == 0 {
		return nil, nil
	}
	c.buf.Write(chunk)
	c.sinceLastAttempt += len(chunk)

	shouldAttempt := c.cfg.SnapshotOnNewline && bytes.Contains(chunk, []byte{'\n'})
	if !shouldAttempt && c.cfg.SnapshotEveryBytes > 0 && c.sinceLastAttempt >= c.cfg.SnapshotEveryBytes {
		shouldAttempt = true
	}
	if !shouldAttempt {
		return nil, nil
	}
	c.sinceLastAttempt = 0
	return c.tryParse()
}

// FinalBytes attempts a final parse using the provided raw (if non-empty) or the internal buffer.
func (c *YAMLController[T]) FinalBytes(raw []byte) (*T, error) {
	if len(raw) > 0 {
		// don't mutate internal buffer on final
		return c.parseYAML(raw)
	}
	return c.tryParse()
}

func (c *YAMLController[T]) tryParse() (*T, error) {
	return c.parseYAML(c.buf.Bytes())
}

func (c *YAMLController[T]) parseYAML(raw []byte) (*T, error) {
	body, err := c.normalizedYAML(raw)
	if err != nil {
		return nil, err
	}
	if c.cfg.ParseTimeout > 0 {
		type result struct {
			v   T
			err error
		}
		ch := make(chan result, 1)
		go func(b []byte) {
			var tmp T
			err := yaml.Unmarshal(b, &tmp)
			ch <- result{v: tmp, err: err}
		}(append([]byte(nil), body...))
		select {
		case r := <-ch:
			if r.err != nil {
				return nil, r.err
			}
			out := r.v
			return &out, nil
		case <-time.After(c.cfg.ParseTimeout):
			return nil, errors.New("parse timeout")
		}
	}
	var out T
	if err := yaml.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *YAMLController[T]) normalizedYAML(raw []byte) ([]byte, error) {
	_, body := StripCodeFenceBytes(raw)
	if c.cfg.MaxBytes > 0 && len(body) > c.cfg.MaxBytes {
		return nil, errPayloadTooLarge
	}
	src := strings.TrimSpace(string(body))
	if src == "" {
		return nil, errEmptyBody
	}
	if c.cfg.SanitizeEnabled() {
		result := yamlsanitize.Sanitize(src)
		if trimmed := strings.TrimSpace(result.Sanitized); trimmed != "" {
			src = trimmed
		}
	}
	return []byte(src), nil
}

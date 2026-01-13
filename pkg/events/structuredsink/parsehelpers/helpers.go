package parsehelpers

import (
	"bytes"
	"errors"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
}

// YAMLController incrementally accumulates bytes and attempts to parse typed YAML
// on a cadence (newline or every N bytes). T should be a struct type with yaml tags.
type YAMLController[T any] struct {
	cfg              DebounceConfig
	buf              bytes.Buffer
	sinceLastAttempt int
}

func NewDebouncedYAML[T any](cfg DebounceConfig) *YAMLController[T] {
	return &YAMLController[T]{cfg: cfg}
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
		lang, body := StripCodeFenceBytes(raw)
		_ = lang
		if c.cfg.MaxBytes > 0 && len(body) > c.cfg.MaxBytes {
			return nil, errors.New("payload too large")
		}
		var out T
		if err := yaml.Unmarshal(body, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	return c.tryParse()
}

func (c *YAMLController[T]) tryParse() (*T, error) {
	lang, body := StripCodeFenceBytes(c.buf.Bytes())
	_ = lang
	if c.cfg.MaxBytes > 0 && len(body) > c.cfg.MaxBytes {
		return nil, errors.New("payload too large")
	}
	// empty body -> treat as no-op
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, errors.New("empty")
	}
	var out T
	// Optional timeout window
	if c.cfg.ParseTimeout > 0 {
		type result struct {
			v   *T
			err error
		}
		ch := make(chan result, 1)
		go func(b []byte) {
			var tmp T
			err := yaml.Unmarshal(b, &tmp)
			if err == nil {
				out = tmp
			}
			ch <- result{v: &out, err: err}
		}(append([]byte(nil), body...))
		select {
		case r := <-ch:
			return r.v, r.err
		case <-time.After(c.cfg.ParseTimeout):
			return nil, errors.New("parse timeout")
		}
	}
	if err := yaml.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

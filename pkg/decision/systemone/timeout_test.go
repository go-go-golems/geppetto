package systemone

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type timeoutTransport func(*http.Request) (*http.Response, error)

var _ http.RoundTripper = timeoutTransport(nil)

func (f timeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDirectClientTimeoutPolicy(t *testing.T) {
	for _, tc := range []struct {
		name     string
		injected bool
		timeout  time.Duration
	}{
		{name: "default", timeout: 60 * time.Second},
		{name: "injected", injected: true, timeout: 7 * time.Second},
		{name: "injected_without_timeout", injected: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			transport := timeoutTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				deadline, hasDeadline := req.Context().Deadline()
				if tc.timeout == 0 {
					if hasDeadline {
						t.Fatal("injected zero timeout must remain unset")
					}
				} else {
					if !hasDeadline {
						t.Fatal("request has no client deadline")
					}
					remaining := time.Until(deadline)
					if remaining <= tc.timeout-time.Second || remaining > tc.timeout {
						t.Fatalf("remaining deadline %v, want approximately %v", remaining, tc.timeout)
					}
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{},
					Body: io.NopCloser(strings.NewReader(okBody)), Request: req}, nil
			})
			options := Options{Model: "djev", APIKey: "dummy-key"}
			var injected *http.Client
			if tc.injected {
				injected = &http.Client{Transport: transport, Timeout: tc.timeout}
				options.HTTPClient = injected
			} else {
				previous := http.DefaultTransport
				http.DefaultTransport = transport
				t.Cleanup(func() { http.DefaultTransport = previous })
			}
			p, err := New(options)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.Decide(t.Context(), request()); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("network calls = %d, want 1", calls)
			}
			if injected != nil && (injected.Timeout != tc.timeout || injected.CheckRedirect != nil) {
				t.Fatal("constructor mutated the injected client")
			}
		})
	}
}

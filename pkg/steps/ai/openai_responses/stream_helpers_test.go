package openai_responses

import "testing"

func TestMissingProviderSuffix(t *testing.T) {
	tests := []struct {
		name    string
		current string
		full    string
		want    string
	}{
		{name: "empty full", current: "abc", full: "", want: ""},
		{name: "already complete", current: "hello", full: "hello", want: ""},
		{name: "current has full suffix", current: "say hello", full: "hello", want: ""},
		{name: "simple append", current: "hel", full: "hello", want: "lo"},
		{name: "no overlap", current: "abc", full: "xyz", want: "xyz"},
		{name: "partial overlap", current: "thinking har", full: "hard.", want: "d."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := missingProviderSuffix(tt.current, tt.full); got != tt.want {
				t.Fatalf("missingProviderSuffix(%q, %q) = %q, want %q", tt.current, tt.full, got, tt.want)
			}
		})
	}
}

func TestResponsesChunkFromValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "nil", in: nil, want: ""},
		{name: "string", in: "hello", want: "hello"},
		{name: "object", in: map[string]any{"answer": "yes"}, want: `{"answer":"yes"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := responsesChunkFromValue(tt.in); got != tt.want {
				t.Fatalf("responsesChunkFromValue(%#v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

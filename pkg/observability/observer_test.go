package observability

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type captureObserver struct {
	records []Record
}

func (o *captureObserver) OnGeppettoRecord(_ context.Context, rec Record) {
	o.records = append(o.records, rec)
}

type panicObserver struct{}

func (panicObserver) OnGeppettoRecord(context.Context, Record) {
	panic("observer failed")
}

func TestParseTraceLevel(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want TraceLevel
	}{
		{"", TraceOff},
		{"off", TraceOff},
		{"events", TraceEvents},
		{"provider", TraceProvider},
		{" PROVIDER ", TraceProvider},
	} {
		got, err := ParseTraceLevel(tc.in)
		if err != nil {
			t.Fatalf("ParseTraceLevel(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseTraceLevel(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	if _, err := ParseTraceLevel("raw"); err == nil {
		t.Fatalf("expected raw to be rejected until raw stream capture is implemented")
	}
	if _, err := ParseTraceLevel("verbose"); err == nil {
		t.Fatalf("expected invalid level error")
	}
}

func TestNotifyPanicSafe(t *testing.T) {
	Notify(context.Background(), panicObserver{}, Record{Stage: StageProviderRoutedEvent})
}

func TestNotifySetsTimestamp(t *testing.T) {
	obs := &captureObserver{}
	Notify(context.Background(), obs, Record{Stage: StageProviderRoutedEvent})
	if len(obs.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(obs.records))
	}
	if obs.records[0].Timestamp.IsZero() {
		t.Fatalf("expected timestamp to be set")
	}
}

func TestMarshalEvidenceJSONRedactsAndCaps(t *testing.T) {
	payload := map[string]any{
		"type":              "response.test",
		"authorization":     "Bearer secret-token",
		"encrypted_content": "abcdef",
		"nested": map[string]any{
			"text": strings.Repeat("x", 20),
		},
	}
	b := MarshalEvidenceJSON(payload, Config{Level: TraceProvider, MaxPayloadBytes: 8, RedactProviderData: true})

	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("expected valid JSON, got %s: %v", string(b), err)
	}
	if decoded["authorization"] == "Bearer secret-token" {
		t.Fatalf("authorization was not redacted: %s", string(b))
	}
	if decoded["encrypted_content"] == "abcdef" {
		t.Fatalf("encrypted content was not redacted: %s", string(b))
	}
	nested, ok := decoded["nested"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested object: %s", string(b))
	}
	text, _ := nested["text"].(string)
	if !strings.Contains(text, "<truncated:") {
		t.Fatalf("expected capped string marker, got %q in %s", text, string(b))
	}
}

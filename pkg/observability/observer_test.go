package observability

import (
	"context"
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

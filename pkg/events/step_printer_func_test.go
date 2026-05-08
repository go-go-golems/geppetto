package events

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
)

func TestStepPrinterSuppressesFinalReasoningSummaryInfo(t *testing.T) {
	metadata := EventMetadata{}
	payload, err := json.Marshal(NewInfoEvent(metadata, "reasoning-summary", map[string]interface{}{"text": "already streamed"}))
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var out bytes.Buffer
	printer := StepPrinterFunc("", &out)
	msg := message.NewMessage("test", payload)
	if err := printer(msg); err != nil {
		t.Fatalf("printer: %v", err)
	}
	if got := out.String(); got != "" {
		t.Fatalf("expected final reasoning-summary info to be suppressed, got %q", got)
	}
}

func TestStepPrinterStillPrintsReasoningSummaryBoundaries(t *testing.T) {
	metadata := EventMetadata{}
	var out bytes.Buffer
	printer := StepPrinterFunc("", &out)

	for _, name := range []string{"reasoning-summary-started", "reasoning-summary-ended"} {
		payload, err := json.Marshal(NewInfoEvent(metadata, name, nil))
		if err != nil {
			t.Fatalf("marshal event: %v", err)
		}
		if err := printer(message.NewMessage(name, payload)); err != nil {
			t.Fatalf("printer %s: %v", name, err)
		}
	}

	got := out.String()
	if !strings.Contains(got, "[i] reasoning-summary-started") || !strings.Contains(got, "[i] reasoning-summary-ended") {
		t.Fatalf("expected summary boundary markers to remain visible, got %q", got)
	}
}

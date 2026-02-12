package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGetRunHandlerMissingRunReturnsNotFound(t *testing.T) {
	h := &APIHandler{BaseDir: t.TempDir()}

	req := httptest.NewRequest(http.MethodGet, "/api/runs/does-not-exist", nil)
	w := httptest.NewRecorder()
	h.GetRunHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestParseEventsStreamsAndParsesNDJSON(t *testing.T) {
	baseDir := t.TempDir()
	eventsPath := filepath.Join(baseDir, "events.ndjson")
	content := `{"type":"step","ts":1000,"event":{"name":"a","meta":{"k":"v"}}}
{"type":"step","ts":2000,"event":{"name":"b"}}
not-json
`
	if err := os.WriteFile(eventsPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write events file: %v", err)
	}

	h := &APIHandler{}
	events, err := h.parseEvents(baseDir, eventsPath)
	if err != nil {
		t.Fatalf("parseEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 parsed events, got %d", len(events))
	}
	if events[0].Meta["k"] != "v" {
		t.Fatalf("expected first event meta k=v, got %#v", events[0].Meta)
	}
}

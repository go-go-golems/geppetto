package fixtures

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// diskTap implements engine.DebugTap to persist raw provider breadcrumbs under out/raw.
type diskTap struct {
	rawDir    string
	turnIndex int
	turnID    string

	mu      sync.Mutex
	sseFile *os.File
	perName map[string]int
	seq     int // global sequence across provider object types
}

func NewDiskTap(outDir string, turnIndex int, turnID string) *diskTap {
	dt := &diskTap{rawDir: filepath.Join(outDir, "raw"), turnIndex: turnIndex, turnID: turnID, perName: map[string]int{}}
	_ = os.MkdirAll(dt.rawDir, 0755)
	return dt
}

func (d *diskTap) path(name string) string {
	return filepath.Join(d.rawDir, name)
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0644)
}

func (d *diskTap) OnHTTP(req *http.Request, body []byte) {
	p := d.path(fmt.Sprintf("turn-%d-http-request.json", d.turnIndex))
	env := map[string]any{
		"turn_index": d.turnIndex,
		"turn_id":    d.turnID,
		"method":     req.Method,
		"url":        req.URL.String(),
		"headers":    headerMap(req.Header),
		"body":       string(body),
	}
	writeJSON(p, env)
}

func (d *diskTap) OnHTTPResponse(resp *http.Response, body []byte) {
	p := d.path(fmt.Sprintf("turn-%d-http-response.json", d.turnIndex))
	env := map[string]any{
		"turn_index": d.turnIndex,
		"turn_id":    d.turnID,
		"status":     resp.StatusCode,
		"headers":    headerMap(resp.Header),
		"body":       jsonRawOrString(body),
	}
	writeJSON(p, env)
}

func (d *diskTap) OnSSE(event string, data []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sseFile == nil {
		f, err := os.Create(d.path(fmt.Sprintf("turn-%d-sse.log", d.turnIndex)))
		if err != nil {
			return
		}
		d.sseFile = f
		// header line to include turn context
		_, _ = fmt.Fprintf(d.sseFile, "# turn_index=%d turn_id=%s\n\n", d.turnIndex, d.turnID)
	}
	// Write minimal framing: event name and data JSON/string
	if event != "" {
		_, _ = d.sseFile.WriteString("event: " + event + "\n")
	}
	_, _ = d.sseFile.WriteString(string(data) + "\n\n")
}

func (d *diskTap) OnProviderObject(name string, v any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seq++
	c := d.perName[name] + 1
	d.perName[name] = c
	safeName := strings.NewReplacer("/", "-", "\\", "-", " ", "_").Replace(name)
	p := d.path(fmt.Sprintf("turn-%d-provider-%06d-%s.json", d.turnIndex, d.seq, safeName))
	wrapped := map[string]any{
		"seq":        d.seq,
		"type":       name,
		"turn_index": d.turnIndex,
		"turn_id":    d.turnID,
		"object":     v,
	}
	writeJSON(p, wrapped)
}

func (d *diskTap) OnTurnBeforeConversion(turnYAML []byte) {
	p := d.path(fmt.Sprintf("turn-%d-input.yaml", d.turnIndex))
	_ = os.WriteFile(p, turnYAML, 0644)
}

func (d *diskTap) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sseFile != nil {
		_ = d.sseFile.Close()
		d.sseFile = nil
	}
}

func headerMap(h http.Header) map[string][]string {
	m := make(map[string][]string, len(h))
	for k, v := range h {
		m[k] = v
	}
	return m
}

func jsonRawOrString(b []byte) any {
	var v any
	if json.Unmarshal(b, &v) == nil {
		return v
	}
	return string(b)
}

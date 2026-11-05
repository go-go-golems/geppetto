package systemone

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/security"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
)

const okBody = `{"answers":{
 "route":{"choice":"semantic","confidence":0.907,"probabilities":{"keyword":0.021,"none":0.0000125,"semantic":0.979},"type":"choice"},
 "needs":{"noul":0.994,"type":"noul"},
 "complexity":{"confidence":0.959,"legend":{"0":"a","1":"b","2":"c"},"probabilities":{"0":0.001,"1":0.003,"2":0.996},"score":1.99,"type":"score"}},
 "model":"djev","usage":{"input_tokens":228,"output_tokens":16},"latency_ms":31,"future_field":{"x":1}}`

func request() decision.Request {
	return decision.Request{
		State: map[string]any{"query": "how does the rebalance law preserve placements?"},
		Questions: map[string]decision.Question{
			"route": {Type: decision.Choice, Instructions: "Which strategy?", Options: []decision.Option{
				{Name: "semantic", Description: "concepts"}, {Name: "keyword", Description: "identifiers"}, {Name: "none"}}},
			"needs":      {Type: decision.Noul, Instructions: "Needs documents?", True: "corpus facts"},
			"complexity": {Type: decision.Score, Instructions: "How complex?", Levels: []string{"a", "b", "c"}},
		},
	}
}

type server struct {
	*httptest.Server
	hits    atomic.Int32
	lastReq atomic.Value // []byte
	lastHdr atomic.Value // http.Header
}

func newServer(t *testing.T, h func(w http.ResponseWriter, r *http.Request)) *server {
	t.Helper()
	s := &server{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		b, _ := io.ReadAll(r.Body)
		s.lastReq.Store(b)
		s.lastHdr.Store(r.Header.Clone())
		if r.URL.Path != "/v1/systemone" || r.Method != http.MethodPost {
			http.Error(w, "wrong route "+r.Method+" "+r.URL.Path, http.StatusTeapot)
			return
		}
		h(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func reply(status int, body string, hdr ...string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i+1 < len(hdr); i += 2 {
			w.Header().Set(hdr[i], hdr[i+1])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func local() security.OutboundURLOptions {
	return security.OutboundURLOptions{AllowHTTP: true, AllowLocalNetworks: true}
}

func provider(t *testing.T, s *server, mod ...func(*Options)) *Provider {
	t.Helper()
	o := Options{BaseURL: s.URL + "/v1", APIKey: "test-key", Model: "djev", OutboundURL: local()}
	for _, m := range mod {
		m(&o)
	}
	p, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDecide(t *testing.T) {
	s := newServer(t, reply(200, okBody, "X-Request-Id", "req-123"))
	rate := 0.042
	p := provider(t, s, func(o *Options) { o.CostPerMInputTokens = &rate })
	r, err := p.Decide(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if r.Provider != "systemone" || r.Model != "djev" || r.RequestID != "req-123" || r.DurationMs == nil {
		t.Fatalf("response meta %+v", r)
	}
	if r.Usage == nil || r.Usage.InputTokens != 228 || r.Usage.OutputTokens != 16 || r.Cost == nil || math.Abs(*r.Cost-228*0.042/1e6) > 1e-15 {
		t.Fatalf("usage/cost %+v %v", r.Usage, r.Cost)
	}
	if a := r.Answers["route"]; a.Choice != "semantic" || a.Confidence != 0.907 || a.Probabilities["keyword"] != 0.021 {
		t.Fatalf("route %+v", a)
	}
	if a := r.Answers["needs"]; a.Type != decision.Noul || a.Probability != 0.994 {
		t.Fatalf("needs %+v", a)
	}
	if a := r.Answers["complexity"]; a.Score != 1.99 || len(a.LevelProbabilities) != 3 || a.LevelProbabilities[2] != 0.996 {
		t.Fatalf("complexity %+v", a)
	}
	hdr := s.lastHdr.Load().(http.Header)
	if hdr.Get("Authorization") != "Bearer test-key" || hdr.Get("Content-Type") != "application/json" {
		t.Fatalf("headers %v", hdr)
	}
	var sent map[string]any
	body := s.lastReq.Load().([]byte)
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatal(err)
	}
	if len(sent) != 3 || sent["model"] != "djev" {
		t.Fatalf("top-level fields %v", sent)
	}
	qs := sent["questions"].(map[string]any)
	if c := qs["needs"].(map[string]any)["criteria"].(map[string]any); c["true"] != "corpus facts" || c["false"] != nil {
		t.Fatalf("noul criteria %v", c)
	}
	if c := qs["complexity"].(map[string]any)["criteria"].([]any); len(c) != 3 || c[0] != "a" {
		t.Fatalf("score criteria %v", c)
	}
}

// Options reach the server in slice order, not sorted.
func TestDecideKeepsOptionOrder(t *testing.T) {
	s := newServer(t, reply(200, `{"model":"djev","answers":{"q":{"type":"choice","choice":"z","confidence":1,"probabilities":{"z":1,"a":0,"m":0}}}}`))
	p := provider(t, s)
	_, err := p.Decide(context.Background(), decision.Request{State: "s", Questions: map[string]decision.Question{
		"q": {Type: decision.Choice, Instructions: "pick", Options: []decision.Option{{Name: "z"}, {Name: "a", Description: `say "hi"`}, {Name: "m"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := string(s.lastReq.Load().([]byte))
	if !strings.Contains(body, `"criteria":{"z":"","a":"say \"hi\"","m":""}`) {
		t.Fatalf("options reordered or misencoded: %s", body)
	}
}

func TestDecideErrors(t *testing.T) {
	envelope := func(code, msg string) string {
		return `{"error":{"code":"` + code + `","message":"` + msg + `","request_id":"rid"}}`
	}
	cases := []struct {
		name   string
		status int
		body   string
		hdr    []string
		kind   error
		code   string
		msg    string
		retry  time.Duration
	}{
		{"400 keeps message", 400, envelope("systemone_invalid_request", `unknown parameter \"options\"`), nil, decision.ErrInvalidRequest, "systemone_invalid_request", `unknown parameter "options"`, 0},
		{"401 drops message", 401, envelope("unauthorized", "bad key Bearer test-key"), nil, decision.ErrNotGranted, "unauthorized", "", 0},
		{"402", 402, envelope("insufficient_credits", "x"), nil, decision.ErrNotGranted, "insufficient_credits", "", 0},
		{"403", 403, envelope("systemone_policy_disabled", "x"), nil, decision.ErrNotGranted, "systemone_policy_disabled", "", 0},
		{"404", 404, envelope("model_not_found", "the requested model was not found"), nil, decision.ErrModelNotFound, "model_not_found", "the requested model was not found", 0},
		{"413", 413, envelope("payload_too_large", "x"), nil, decision.ErrRequestTooLarge, "payload_too_large", "", 0},
		{"429 retry-after", 429, envelope("systemone_gate_busy", "x"), []string{"Retry-After", "3"}, decision.ErrRateLimited, "systemone_gate_busy", "", 3 * time.Second},
		{"502", 502, envelope("systemone_backend_error", "x"), nil, decision.ErrUnavailable, "systemone_backend_error", "", 0},
		{"503 html body", 503, "<html>down</html>", nil, decision.ErrUnavailable, "", "", 0},
		{"odd code dropped", 400, envelope("Bearer sk-12345", "m"), nil, decision.ErrInvalidRequest, "", "m", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newServer(t, reply(c.status, c.body, c.hdr...))
			_, err := provider(t, s).Decide(context.Background(), request())
			var pe *decision.ProviderError
			if !errors.Is(err, c.kind) || !errors.As(err, &pe) {
				t.Fatalf("want %v ProviderError, got %v", c.kind, err)
			}
			if pe.Code != c.code || pe.Message != c.msg || pe.RetryAfter != c.retry || pe.Status != c.status {
				t.Fatalf("provider error %+v", pe)
			}
			if strings.Contains(err.Error(), "test-key") || strings.Contains(err.Error(), "sk-12345") || strings.Contains(err.Error(), "rebalance") {
				t.Fatalf("error leaks: %v", err)
			}
		})
	}
}

func TestDecideInvalidResponses(t *testing.T) {
	cases := map[string]string{
		"missing answer":      `{"model":"djev","answers":{"needs":{"type":"noul","noul":0.5}}}`,
		"noul without noul":   strings.Replace(okBody, `"noul":0.994,`, ``, 1),
		"choice outside":      strings.Replace(okBody, `"choice":"semantic"`, `"choice":"vector"`, 1),
		"level out of range":  strings.Replace(okBody, `"2":0.996`, `"3":0.996`, 1),
		"padded level key":    strings.Replace(okBody, `"2":0.996`, `"02":0.996`, 1),
		"score above top":     strings.Replace(okBody, `"score":1.99`, `"score":2.5`, 1),
		"trailing data":       okBody + `{}`,
		"not json":            `answers: yes`,
		"unasked answer":      strings.Replace(okBody, `"needs":{`, `"surprise":{"noul":0.1,"type":"noul"},"needs":{`, 1),
		"probability above 1": strings.Replace(okBody, `"noul":0.994`, `"noul":1.5`, 1),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			s := newServer(t, reply(200, body))
			_, err := provider(t, s).Decide(context.Background(), request())
			if !errors.Is(err, decision.ErrInvalidResponse) {
				t.Fatalf("want ErrInvalidResponse, got %v", err)
			}
		})
	}
}

func TestDecideLimits(t *testing.T) {
	s := newServer(t, reply(200, okBody))
	p := provider(t, s, func(o *Options) { o.MaxRequestBytes = 64 })
	if _, err := p.Decide(context.Background(), request()); !errors.Is(err, decision.ErrRequestTooLarge) || s.hits.Load() != 0 {
		t.Fatalf("request limit: %v, hits %d", err, s.hits.Load())
	}
	p = provider(t, s, func(o *Options) { o.MaxResponseBytes = 100 })
	if _, err := p.Decide(context.Background(), request()); !errors.Is(err, decision.ErrResponseTooLarge) {
		t.Fatalf("response limit: %v", err)
	}
	// Local validation fails before any request.
	before := s.hits.Load()
	if _, err := p.Decide(context.Background(), decision.Request{State: "s"}); !errors.Is(err, decision.ErrInvalidRequest) || s.hits.Load() != before {
		t.Fatalf("validation: %v", err)
	}
}

func TestDecideRejectsRedirects(t *testing.T) {
	var followed atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { followed.Store(true) }))
	defer target.Close()
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/v1/systemone", http.StatusFound)
	})
	_, err := provider(t, s).Decide(context.Background(), request())
	if !errors.Is(err, decision.ErrUnavailable) || followed.Load() {
		t.Fatalf("redirect: %v followed=%v", err, followed.Load())
	}
	if strings.Contains(err.Error(), target.URL) {
		t.Fatalf("error leaks URL: %v", err)
	}
}

func TestDecideCancel(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := provider(t, s).Decide(ctx, request()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
}

func TestNewValidation(t *testing.T) {
	cases := map[string]Options{
		"no model":         {APIKey: "k"},
		"no credential":    {Model: "djev"},
		"both":             {Model: "djev", APIKey: "k", TokenSource: staticSource("t")},
		"userinfo":         {Model: "djev", APIKey: "k", BaseURL: "https://u:p@gw.lunaroute.com/v1"},
		"query":            {Model: "djev", APIKey: "k", BaseURL: "https://gw.lunaroute.com/v1?x=1"},
		"dot segment":      {Model: "djev", APIKey: "k", BaseURL: "https://gw.lunaroute.com/v1/../x"},
		"ftp":              {Model: "djev", APIKey: "k", BaseURL: "ftp://gw.lunaroute.com/v1"},
		"http by default":  {Model: "djev", APIKey: "k", BaseURL: "http://gw.lunaroute.com/v1"},
		"local by default": {Model: "djev", APIKey: "k", BaseURL: "https://127.0.0.1/v1"},
	}
	for name, o := range cases {
		if _, err := New(o); !errors.Is(err, decision.ErrInvalidRequest) {
			t.Fatalf("%s: want ErrInvalidRequest, got %v", name, err)
		}
	}
	p, err := New(Options{Model: "djev", APIKey: "k"})
	if err != nil || p.endpoint != "https://gw.lunaroute.com/v1/systemone" {
		t.Fatalf("default endpoint: %v %v", p, err)
	}
	if m := p.Model(); m.Provider != "systemone" || m.Name != "djev" {
		t.Fatalf("model %+v", m)
	}
}

type staticSource string

type credentialsRequest = credentials.Request

func (s staticSource) BearerToken(context.Context, credentialsRequest) (string, error) {
	return string(s), nil
}

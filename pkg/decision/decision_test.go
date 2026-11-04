package decision

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func opts(names ...string) []Option {
	out := make([]Option, len(names))
	for i, n := range names {
		out[i] = Option{Name: n, Description: "about " + n}
	}
	return out
}

func validRequest() Request {
	return Request{
		State: map[string]any{"query": "how does rebalance work?"},
		Questions: map[string]Question{
			"route":     {Type: Choice, Instructions: "Which index?", Options: opts("keyword", "semantic")},
			"needs":     {Type: Noul, Instructions: "Needs documents?", True: "yes", False: "no"},
			"hardness":  {Type: Score, Instructions: "How hard?", Levels: []string{"easy", "medium", "hard"}},
			"plain-nou": {Type: Noul, Instructions: "Is it a question?"},
		},
	}
}

func TestValidateRequest(t *testing.T) {
	m := Model{Provider: "p", Name: "djev"}
	if err := ValidateRequest(validRequest(), m); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	many := make([]Option, MaxOptions+1)
	for i := range many {
		many[i] = Option{Name: strings.Repeat("o", i+1)}
	}
	cases := map[string]func(r *Request){
		"model drift":        func(r *Request) { r.Model = "other" },
		"nil state":          func(r *Request) { r.State = nil },
		"empty state":        func(r *Request) { r.State = "  " },
		"empty object state": func(r *Request) { r.State = map[string]any{} },
		"number state":       func(r *Request) { r.State = 42 },
		"no questions":       func(r *Request) { r.Questions = nil },
		"remote image":       func(r *Request) { r.Images = []string{"https://x/y.png"} },
		"gif image":          func(r *Request) { r.Images = []string{"data:image/gif;base64,AAAA"} },
		"id with space":      func(r *Request) { r.Questions["a b"] = Question{Type: Noul, Instructions: "x"} },
		"long id":            func(r *Request) { r.Questions[strings.Repeat("a", 65)] = Question{Type: Noul, Instructions: "x"} },
		"unknown type":       func(r *Request) { r.Questions["x"] = Question{Type: "multi", Instructions: "x"} },
		"empty instructions": func(r *Request) { r.Questions["x"] = Question{Type: Noul, Instructions: ""} },
		"noul with options":  func(r *Request) { r.Questions["x"] = Question{Type: Noul, Instructions: "x", Options: opts("a")} },
		"choice no options":  func(r *Request) { r.Questions["x"] = Question{Type: Choice, Instructions: "x"} },
		"choice with levels": func(r *Request) {
			r.Questions["x"] = Question{Type: Choice, Instructions: "x", Options: opts("a"), Levels: []string{"l"}}
		},
		"choice with true": func(r *Request) {
			r.Questions["x"] = Question{Type: Choice, Instructions: "x", Options: opts("a"), True: "t"}
		},
		"256 options": func(r *Request) { r.Questions["x"] = Question{Type: Choice, Instructions: "x", Options: many} },
		"duplicate options": func(r *Request) {
			r.Questions["x"] = Question{Type: Choice, Instructions: "x", Options: opts("a", "a")}
		},
		"unnamed option": func(r *Request) {
			r.Questions["x"] = Question{Type: Choice, Instructions: "x", Options: []Option{{Description: "d"}}}
		},
		"score no levels": func(r *Request) { r.Questions["x"] = Question{Type: Score, Instructions: "x"} },
		"11 levels": func(r *Request) {
			r.Questions["x"] = Question{Type: Score, Instructions: "x", Levels: strings.Split("a b c d e f g h i j k", " ")}
		},
		"empty level": func(r *Request) {
			r.Questions["x"] = Question{Type: Score, Instructions: "x", Levels: []string{"a", ""}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := validRequest()
			mutate(&r)
			err := ValidateRequest(r, m)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("want ErrInvalidRequest, got %v", err)
			}
			if strings.Contains(err.Error(), "rebalance") {
				t.Fatalf("error leaks state: %v", err)
			}
		})
	}
	// Structured instructions and states are accepted.
	r := validRequest()
	r.Questions["structured"] = Question{Type: Noul, Instructions: map[string]any{"passage": "p", "question": "relevant?"}}
	r.State = []any{"a", "b"}
	r.Images = []string{"data:image/png;base64,iVBORw0KGgo="}
	if err := ValidateRequest(r, m); err != nil {
		t.Fatalf("structured request rejected: %v", err)
	}
}

func validAnswers() map[string]Answer {
	return map[string]Answer{
		"route":     {Type: Choice, Choice: "semantic", Confidence: 0.9, Probabilities: map[string]float64{"keyword": 0.02, "semantic": 0.98}},
		"needs":     {Type: Noul, Probability: 0.99},
		"hardness":  {Type: Score, Score: 1.2, Confidence: 0.8, LevelProbabilities: []float64{0.1, 0.6, 0.3}},
		"plain-nou": {Type: Noul, Probability: 0},
	}
}

func TestValidateResponse(t *testing.T) {
	req := validRequest()
	if err := ValidateResponse(req, validAnswers()); err != nil {
		t.Fatalf("valid answers rejected: %v", err)
	}
	cases := map[string]func(a map[string]Answer){
		"missing answer":       func(a map[string]Answer) { delete(a, "needs") },
		"extra answer":         func(a map[string]Answer) { a["surprise"] = Answer{Type: Noul, Probability: 0.5} },
		"type mismatch":        func(a map[string]Answer) { a["needs"] = Answer{Type: Choice, Choice: "x"} },
		"probability 1.5":      func(a map[string]Answer) { a["needs"] = Answer{Type: Noul, Probability: 1.5} },
		"probability NaN":      func(a map[string]Answer) { a["needs"] = Answer{Type: Noul, Probability: math.NaN()} },
		"choice not an option": func(a map[string]Answer) { x := a["route"]; x.Choice = "vector"; a["route"] = x },
		"undeclared option probability": func(a map[string]Answer) {
			x := a["route"]
			x.Probabilities = map[string]float64{"vector": 0.5}
			a["route"] = x
		},
		"score above top":  func(a map[string]Answer) { x := a["hardness"]; x.Score = 2.5; a["hardness"] = x },
		"level count":      func(a map[string]Answer) { x := a["hardness"]; x.LevelProbabilities = []float64{1}; a["hardness"] = x },
		"confidence above": func(a map[string]Answer) { x := a["route"]; x.Confidence = 2; a["route"] = x },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			a := validAnswers()
			mutate(a)
			if err := ValidateResponse(req, a); !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("want ErrInvalidResponse, got %v", err)
			}
		})
	}
	// Rounding just past the bounds is tolerated.
	a := validAnswers()
	a["needs"] = Answer{Type: Noul, Probability: 1 + 1e-9}
	if err := ValidateResponse(req, a); err != nil {
		t.Fatalf("rounding rejected: %v", err)
	}
}

func TestProviderError(t *testing.T) {
	e := NewProviderError("systemone", 400, "systemone_invalid_request", `unknown parameter "options"`, "req-1", 0)
	if !errors.Is(e, ErrInvalidRequest) || !strings.Contains(e.Error(), `unknown parameter "options"`) {
		t.Fatalf("400: %v", e)
	}
	e = NewProviderError("systemone", 429, "systemone_gate_busy", "busy, retry with token sk-secret", "", 2*time.Second)
	if !errors.Is(e, ErrRateLimited) || e.Message != "" || strings.Contains(e.Error(), "sk-secret") || e.RetryAfter != 2*time.Second {
		t.Fatalf("429: %+v %v", e, e)
	}
	for status, want := range map[int]error{401: ErrNotGranted, 402: ErrNotGranted, 403: ErrNotGranted, 404: ErrModelNotFound, 408: ErrUnavailable, 413: ErrRequestTooLarge, 415: ErrInvalidRequest, 502: ErrUnavailable, 504: ErrUnavailable} {
		if got := KindForStatus(status); got != want {
			t.Fatalf("%d: %v", status, got)
		}
	}
	long := NewProviderError("p", 400, "c", strings.Repeat("x", 500), "", 0)
	if len(long.Message) > MaxProviderMessage+len("…") {
		t.Fatalf("message not capped: %d", len(long.Message))
	}
}

func TestGate(t *testing.T) {
	for _, c := range []struct {
		p    float64
		want GateAction
	}{{0.95, Act}, {0.7, Act}, {0.6, Confirm}, {0.5, Confirm}, {0.49, Reject}} {
		if got := Gate(c.p, 0.7, 0.5); got != c.want {
			t.Fatalf("Gate(%v) = %s, want %s", c.p, got, c.want)
		}
	}
}

func TestResponseJSONOmitsUnset(t *testing.T) {
	b, err := json.Marshal(Response{Provider: "systemone", Model: "djev", Answers: map[string]Answer{"a": {Type: Noul, Probability: 0.25}}})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"provider":"systemone","model":"djev","answers":{"a":{"type":"noul","probability":0.25}}}` {
		t.Fatalf("json: %s", b)
	}
	// Zero values of the answer's own type survive; other types' fields do not appear.
	cases := []struct {
		a    Answer
		want string
	}{
		{Answer{Type: Noul}, `{"type":"noul","probability":0}`},
		{Answer{Type: Choice, Choice: "x"}, `{"type":"choice","choice":"x","confidence":0}`},
		{Answer{Type: Score, LevelProbabilities: []float64{1, 0}}, `{"type":"score","score":0,"confidence":0,"level_probabilities":[1,0]}`},
	}
	for _, c := range cases {
		b, _ := json.Marshal(c.a)
		if string(b) != c.want {
			t.Fatalf("%+v: %s, want %s", c.a, b, c.want)
		}
		var back Answer
		if err := json.Unmarshal(b, &back); err != nil || back.Type != c.a.Type {
			t.Fatalf("round trip: %v %+v", err, back)
		}
	}
}

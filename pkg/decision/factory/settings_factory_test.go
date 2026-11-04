package factory

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/decision"
	"github.com/go-go-golems/geppetto/pkg/decision/config"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"gopkg.in/yaml.v3"
)

func TestNewProviderRejects(t *testing.T) {
	cases := map[string]struct {
		f    *SettingsFactory
		want string
	}{
		"nil config":  {NewSettingsFactory(nil, nil, nil, nil), "no decision configuration"},
		"no type":     {NewSettingsFactory(&config.DecisionConfig{Engine: "djev"}, nil, nil, nil), "no decision type"},
		"no engine":   {NewSettingsFactory(&config.DecisionConfig{Type: "systemone"}, nil, nil, nil), "no decision model"},
		"unsupported": {NewSettingsFactory(&config.DecisionConfig{Type: "jev", Engine: "m"}, nil, nil, nil), "systemone"},
		"no api":      {NewSettingsFactory(&config.DecisionConfig{Type: "systemone", Engine: "djev"}, nil, nil, nil), "systemone-api-key"},
		"no key":      {NewSettingsFactory(&config.DecisionConfig{Type: "systemone", Engine: "djev"}, settings.NewAPISettings(), nil, nil), "systemone-api-key"},
	}
	for name, c := range cases {
		_, err := c.f.NewProvider()
		if !errors.Is(err, decision.ErrInvalidRequest) || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestValidateInferenceSettingsForDecision(t *testing.T) {
	if err := ValidateInferenceSettingsForDecision(nil); !errors.Is(err, decision.ErrInvalidRequest) {
		t.Fatal(err)
	}
	s, err := settings.NewInferenceSettings()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateInferenceSettingsForDecision(s); err == nil || !strings.Contains(err.Error(), "missing inference_settings.decision") {
		t.Fatalf("no decision: %v", err)
	}
	s.Decision = &config.DecisionConfig{Type: "systemone", Engine: "djev"}
	if err := ValidateInferenceSettingsForDecision(s); err == nil || !strings.Contains(err.Error(), "systemone-api-key") {
		t.Fatalf("no key: %v", err)
	}
	s.API.APIKeys[APIKeyName] = "k"
	if err := ValidateInferenceSettingsForDecision(s); err != nil {
		t.Fatal(err)
	}
}

// A profile in YAML builds a working provider: base URL, key, policy and
// limits all come from InferenceSettings.
func TestProfileYAMLBuildsProvider(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = io.ReadAll(r.Body)
		if r.URL.Path != "/v1/systemone" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"model":"djev","answers":{"q":{"type":"noul","noul":0.75}},"usage":{"input_tokens":10,"output_tokens":1}}`)
	}))
	defer srv.Close()
	doc := `
decision:
  type: systemone
  engine: djev
api:
  api_keys:
    systemone-api-key: test-key
  base_urls:
    systemone-base-url: ` + srv.URL + `/v1
  allow_http:
    decision: true
  allow_local_networks:
    decision: true
model_info:
  cost:
    input: 0.042
`
	s, err := settings.NewInferenceSettings()
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal([]byte(doc), s); err != nil {
		t.Fatal(err)
	}
	f, err := NewSettingsFactoryFromInferenceSettings(s)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.NewProvider()
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Decide(context.Background(), decision.Request{State: "s", Questions: map[string]decision.Question{"q": {Type: decision.Noul, Instructions: "?"}}})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer test-key" || r.Answers["q"].Probability != 0.75 || r.Cost == nil {
		t.Fatalf("auth %q answers %+v cost %v", gotAuth, r.Answers, r.Cost)
	}
	// The clone keeps the decision section; metadata names it.
	c := s.Clone()
	if c.Decision == nil || c.Decision == s.Decision || c.Decision.Engine != "djev" {
		t.Fatalf("clone: %+v", c.Decision)
	}
	if md := s.GetMetadata(); md["decision-engine"] != "djev" || md["decision-type"] != "systemone" {
		t.Fatalf("metadata: %v", md)
	}
}

// Without allow_http, a plain-HTTP base URL is refused by the outbound policy.
func TestOutboundPolicyApplies(t *testing.T) {
	api := settings.NewAPISettings()
	api.APIKeys[APIKeyName] = "k"
	api.BaseUrls[BaseURLName] = "http://127.0.0.1:1/v1"
	_, err := NewSettingsFactory(&config.DecisionConfig{Type: "systemone", Engine: "djev"}, api, nil, nil).NewProvider()
	if !errors.Is(err, decision.ErrInvalidRequest) || !strings.Contains(err.Error(), "outbound URL policy") {
		t.Fatalf("policy: %v", err)
	}
}

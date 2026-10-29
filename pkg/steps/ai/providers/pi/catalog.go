package pi

import (
	"sort"
	"strings"
)

// Wire APIs, as Pi names them in a model's "api" field.
const (
	APIOpenAICompletions    = "openai-completions"
	APIOpenAIResponses      = "openai-responses"
	APIOpenAICodexResponses = "openai-codex-responses"
	APIAnthropicMessages    = "anthropic-messages"
	APIGoogleGenerativeAI   = "google-generative-ai"
)

// supportedAPIs are the wire APIs with a Geppetto engine.
var supportedAPIs = map[string]bool{
	APIOpenAICompletions:    true,
	APIOpenAIResponses:      true,
	APIOpenAICodexResponses: true,
	APIAnthropicMessages:    true,
	APIGoogleGenerativeAI:   true,
}

// Compat is the part of Pi's per-model compatibility block this package uses.
type Compat struct {
	SupportsReasoningEffort *bool  `json:"supportsReasoningEffort,omitempty"`
	MaxTokensField          string `json:"maxTokensField,omitempty"` // max_tokens | max_completion_tokens
	ThinkingFormat          string `json:"thinkingFormat,omitempty"` // openai, zai, deepseek, …
}

// Model is one catalog entry.
type Model struct {
	Provider      string
	ID            string
	Name          string
	API           string
	BaseURL       string
	Reasoning     bool
	Input         []string // "text", "image"
	ContextWindow int
	MaxTokens     int
	Compat        Compat
	// ThinkingLevelMap maps a Pi thinking level to the provider's value; a
	// present key with a nil value means the level is not offered.
	ThinkingLevelMap map[string]*string
	// Headers are non-secret request headers from models.json; values may be
	// "$ENV" or "!command" references, resolved when the model is used.
	Headers map[string]string
	// Supported reports whether a Geppetto engine serves API.
	Supported bool
}

// Ref is "provider/id".
func (m Model) Ref() string { return m.Provider + "/" + m.ID }

// Images reports whether the model accepts image input.
func (m Model) Images() bool {
	for _, i := range m.Input {
		if i == "image" {
			return true
		}
	}
	return false
}

// ThinkingLevels lists the Pi thinking levels the model offers, in Pi's order.
func (m Model) ThinkingLevels() []string {
	if !m.Reasoning {
		return nil
	}
	var out []string
	for _, l := range []string{"off", "minimal", "low", "medium", "high", "xhigh"} {
		if v, ok := m.ThinkingLevelMap[l]; ok && v == nil && l != "off" {
			continue
		}
		out = append(out, l)
	}
	return out
}

// Catalog is the merged model list: models-store.json, then models.json over it.
type Catalog struct {
	models []Model
	byRef  map[string]int
}

// Models returns every model, sorted by provider and id.
func (c *Catalog) Models() []Model { return append([]Model(nil), c.models...) }

// Lookup finds "provider/id".
func (c *Catalog) Lookup(ref string) (Model, bool) {
	i, ok := c.byRef[strings.TrimSpace(ref)]
	if !ok {
		return Model{}, false
	}
	return c.models[i], true
}

// modelDef is a model as both files store it.
type modelDef struct {
	ID               string             `json:"id"`
	Name             string             `json:"name,omitempty"`
	API              string             `json:"api,omitempty"`
	Provider         string             `json:"provider,omitempty"`
	BaseURL          string             `json:"baseUrl,omitempty"`
	Reasoning        *bool              `json:"reasoning,omitempty"`
	Input            []string           `json:"input,omitempty"`
	ContextWindow    int                `json:"contextWindow,omitempty"`
	MaxTokens        int                `json:"maxTokens,omitempty"`
	MaxOutputTokens  int                `json:"maxOutputTokens,omitempty"`
	Headers          map[string]string  `json:"headers,omitempty"`
	Compat           *Compat            `json:"compat,omitempty"`
	ThinkingLevelMap map[string]*string `json:"thinkingLevelMap,omitempty"`
}

// providerConfig is one provider of models.json.
type providerConfig struct {
	BaseURL string            `json:"baseUrl,omitempty"`
	APIKey  string            `json:"apiKey,omitempty"`
	API     string            `json:"api,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Compat  *Compat           `json:"compat,omitempty"`
	Models  []modelDef        `json:"models,omitempty"`
}

func buildCatalog(store map[string][]modelDef, config map[string]providerConfig) *Catalog {
	merged := map[string]Model{}
	add := func(provider string, d modelDef, pc *providerConfig) {
		if d.ID == "" {
			return
		}
		ref := provider + "/" + d.ID
		m, ok := merged[ref]
		if !ok {
			m = Model{Provider: provider, ID: d.ID}
		}
		if pc != nil {
			m.API = first(m.API, pc.API)
			m.BaseURL = first(pc.BaseURL, m.BaseURL)
			m.Headers = mergeHeaders(m.Headers, pc.Headers)
			if pc.Compat != nil {
				m.Compat = mergeCompat(m.Compat, *pc.Compat)
			}
		}
		m.Name = first(d.Name, m.Name, d.ID)
		m.API = first(d.API, m.API)
		m.BaseURL = first(d.BaseURL, m.BaseURL)
		if d.Reasoning != nil {
			m.Reasoning = *d.Reasoning
		}
		if len(d.Input) > 0 {
			m.Input = append([]string(nil), d.Input...)
		}
		if d.ContextWindow > 0 {
			m.ContextWindow = d.ContextWindow
		}
		if d.MaxTokens > 0 {
			m.MaxTokens = d.MaxTokens
		} else if d.MaxOutputTokens > 0 {
			m.MaxTokens = d.MaxOutputTokens
		}
		m.Headers = mergeHeaders(m.Headers, d.Headers)
		if d.Compat != nil {
			m.Compat = mergeCompat(m.Compat, *d.Compat)
		}
		if d.ThinkingLevelMap != nil {
			m.ThinkingLevelMap = d.ThinkingLevelMap
		}
		merged[ref] = m
	}
	for provider, defs := range store {
		for _, d := range defs {
			add(first(d.Provider, provider), d, nil)
		}
	}
	for provider, pc := range config {
		pc := pc
		for _, d := range pc.Models {
			add(provider, d, &pc)
		}
		// Provider-level settings also apply to the provider's stored models.
		for ref, m := range merged {
			if m.Provider == provider {
				if pc.BaseURL != "" {
					m.BaseURL = pc.BaseURL
				}
				m.API = first(m.API, pc.API)
				m.Headers = mergeHeaders(pc.Headers, m.Headers)
				merged[ref] = m
			}
		}
	}
	c := &Catalog{byRef: map[string]int{}}
	for _, m := range merged {
		if len(m.Input) == 0 {
			m.Input = []string{"text"}
		}
		m.Supported = supportedAPIs[m.API]
		c.models = append(c.models, m)
	}
	sort.Slice(c.models, func(i, j int) bool { return c.models[i].Ref() < c.models[j].Ref() })
	for i, m := range c.models {
		c.byRef[m.Ref()] = i
	}
	return c
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func mergeHeaders(base, over map[string]string) map[string]string {
	if len(base) == 0 && len(over) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func mergeCompat(base, over Compat) Compat {
	if over.SupportsReasoningEffort != nil {
		base.SupportsReasoningEffort = over.SupportsReasoningEffort
	}
	base.MaxTokensField = first(over.MaxTokensField, base.MaxTokensField)
	base.ThinkingFormat = first(over.ThinkingFormat, base.ThinkingFormat)
	return base
}

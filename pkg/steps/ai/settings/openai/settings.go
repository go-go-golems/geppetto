package openai

import (
	_ "embed"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/huandu/go-clone"
)

type Settings struct {
	// How many choice to create for each prompt
	N *int `yaml:"n" glazed:"openai-n"`
	// PresencePenalty to use
	PresencePenalty *float64 `yaml:"presence_penalty,omitempty" glazed:"openai-presence-penalty"`
	// FrequencyPenalty to use
	FrequencyPenalty *float64 `yaml:"frequency_penalty,omitempty" glazed:"openai-frequency-penalty"`
	// LogitBias to use
	// TODO(manuel, 2023-03-28) Properly load logit bias
	// See https://github.com/go-go-golems/geppetto/issues/48
	LogitBias map[string]string `yaml:"logit_bias,omitempty" glazed:"openai-logit-bias"`
	// ReasoningEffort for Responses API (low|medium|high)
	ReasoningEffort *string `yaml:"reasoning_effort,omitempty" glazed:"openai-reasoning-effort"`
	// ParallelToolCalls is a hint for tool parallelization in Responses
	ParallelToolCalls *bool `yaml:"parallel_tool_calls,omitempty" glazed:"openai-parallel-tool-calls"`
	// ReasoningSummary requests a public reasoning summary ("auto" to enable)
	ReasoningSummary *string `yaml:"reasoning_summary,omitempty" glazed:"openai-reasoning-summary"`
	// IncludeReasoningEncrypted requests encrypted reasoning content for reuse across turns
	IncludeReasoningEncrypted *bool `yaml:"include_reasoning_encrypted,omitempty" glazed:"openai-include-reasoning-encrypted"`
	// StreamIncludeUsage requests usage in streaming events (when supported)
	StreamIncludeUsage *bool `yaml:"stream-include-usage,omitempty" glazed:"openai-stream-include-usage"`
}

func NewSettings() (*Settings, error) {
	s := &Settings{
		N:                         nil,
		PresencePenalty:           nil,
		FrequencyPenalty:          nil,
		LogitBias:                 map[string]string{},
		ReasoningEffort:           nil,
		ParallelToolCalls:         nil,
		ReasoningSummary:          nil,
		IncludeReasoningEncrypted: nil,
		StreamIncludeUsage:        nil,
	}

	p, err := NewValueSection()
	if err != nil {
		return nil, err
	}

	err = p.InitializeStructFromFieldDefaults(s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Settings) Clone() *Settings {
	return clone.Clone(s).(*Settings)
}

//go:embed "chat.yaml"
var settingsYAML []byte

type ValueSection struct {
	*schema.SectionImpl `yaml:",inline"`
}

const OpenAiChatSlug = "openai-chat"

func NewValueSection(options ...schema.SectionOption) (*ValueSection, error) {
	ret, err := schema.NewSectionFromYAML(settingsYAML, options...)
	if err != nil {
		return nil, err
	}

	return &ValueSection{
		SectionImpl: ret,
	}, nil
}

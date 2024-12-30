package openai

import (
	"github.com/mitchellh/mapstructure"
	"github.com/sashabaranov/go-openai"
)

// ChatCompletionMetadata contains metadata extracted from OpenAI chat completion stream responses
type ChatCompletionMetadata struct {
	Model             string              `json:"model" mapstructure:"model"`
	SystemFingerprint string              `json:"system_fingerprint" mapstructure:"system_fingerprint"`
	PromptAnnotations []PromptAnnotation  `json:"prompt_annotations,omitempty" mapstructure:"prompt_annotations,omitempty"`
	Usage             *Usage              `json:"usage,omitempty" mapstructure:"usage,omitempty"`
	ContentFilters    []ContentFilterData `json:"content_filters,omitempty" mapstructure:"content_filters,omitempty"`
}

// PromptAnnotation represents an annotation for a prompt
type PromptAnnotation struct {
	PromptIndex          int                  `json:"prompt_index" mapstructure:"prompt_index"`
	ContentFilterResults ContentFilterResults `json:"content_filter_results" mapstructure:"content_filter_results"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens      int                     `json:"prompt_tokens" mapstructure:"prompt_tokens"`
	CompletionTokens  int                     `json:"completion_tokens" mapstructure:"completion_tokens"`
	TotalTokens       int                     `json:"total_tokens" mapstructure:"total_tokens"`
	PromptDetails     *PromptTokenDetails     `json:"prompt_details,omitempty" mapstructure:"prompt_details,omitempty"`
	CompletionDetails *CompletionTokenDetails `json:"completion_details,omitempty" mapstructure:"completion_details,omitempty"`
}

// PromptTokenDetails contains detailed information about prompt tokens
type PromptTokenDetails struct {
	AudioTokens  int `json:"audio_tokens" mapstructure:"audio_tokens"`
	CachedTokens int `json:"cached_tokens" mapstructure:"cached_tokens"`
}

// CompletionTokenDetails contains detailed information about completion tokens
type CompletionTokenDetails struct {
	AudioTokens     int `json:"audio_tokens" mapstructure:"audio_tokens"`
	ReasoningTokens int `json:"reasoning_tokens" mapstructure:"reasoning_tokens"`
}

// ContentFilterData contains content filter results
type ContentFilterData struct {
	Index                int                  `json:"index" mapstructure:"index"`
	ContentFilterResults ContentFilterResults `json:"content_filter_results" mapstructure:"content_filter_results"`
}

// ContentFilterResults contains individual content filter results
type ContentFilterResults struct {
	Hate      *FilterResult `json:"hate,omitempty" mapstructure:"hate,omitempty"`
	SelfHarm  *FilterResult `json:"self_harm,omitempty" mapstructure:"self_harm,omitempty"`
	Sexual    *FilterResult `json:"sexual,omitempty" mapstructure:"sexual,omitempty"`
	Violence  *FilterResult `json:"violence,omitempty" mapstructure:"violence,omitempty"`
	JailBreak *FilterResult `json:"jailbreak,omitempty" mapstructure:"jailbreak,omitempty"`
	Profanity *FilterResult `json:"profanity,omitempty" mapstructure:"profanity,omitempty"`
}

// FilterResult represents a single content filter result
type FilterResult struct {
	Filtered bool   `json:"filtered" mapstructure:"filtered"`
	Severity string `json:"severity,omitempty" mapstructure:"severity,omitempty"`
	Detected bool   `json:"detected,omitempty" mapstructure:"detected,omitempty"`
}

// ExtractChatCompletionMetadata extracts metadata from an OpenAI chat completion stream response
func ExtractChatCompletionMetadata(response *openai.ChatCompletionStreamResponse) (map[string]interface{}, error) {
	if response == nil {
		return nil, nil
	}

	// Convert OpenAI types to our internal types
	metadata := &ChatCompletionMetadata{
		Model:             response.Model,
		SystemFingerprint: response.SystemFingerprint,
	}

	// Convert prompt annotations
	if len(response.PromptAnnotations) > 0 {
		metadata.PromptAnnotations = make([]PromptAnnotation, len(response.PromptAnnotations))
		for i, pa := range response.PromptAnnotations {
			metadata.PromptAnnotations[i] = PromptAnnotation{
				PromptIndex:          pa.PromptIndex,
				ContentFilterResults: convertContentFilterResults(pa.ContentFilterResults),
			}
		}
	}

	// Convert usage if present
	if response.Usage != nil {
		metadata.Usage = &Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		}

		// Add prompt details if available
		if response.Usage.PromptTokensDetails != nil {
			metadata.Usage.PromptDetails = &PromptTokenDetails{
				AudioTokens:  response.Usage.PromptTokensDetails.AudioTokens,
				CachedTokens: response.Usage.PromptTokensDetails.CachedTokens,
			}
		}

		// Add completion details if available
		if response.Usage.CompletionTokensDetails != nil {
			metadata.Usage.CompletionDetails = &CompletionTokenDetails{
				AudioTokens:     response.Usage.CompletionTokensDetails.AudioTokens,
				ReasoningTokens: response.Usage.CompletionTokensDetails.ReasoningTokens,
			}
		}
	}

	// Convert content filter results if present
	if len(response.PromptFilterResults) > 0 {
		metadata.ContentFilters = make([]ContentFilterData, len(response.PromptFilterResults))
		for i, filter := range response.PromptFilterResults {
			metadata.ContentFilters[i] = ContentFilterData{
				Index:                filter.Index,
				ContentFilterResults: convertContentFilterResults(filter.ContentFilterResults),
			}
		}
	}

	// Convert to map using mapstructure
	var result map[string]interface{}
	err := mapstructure.Decode(metadata, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// convertContentFilterResults converts OpenAI content filter results to our internal type
func convertContentFilterResults(cfr openai.ContentFilterResults) ContentFilterResults {
	return ContentFilterResults{
		Hate: &FilterResult{
			Filtered: cfr.Hate.Filtered,
			Severity: cfr.Hate.Severity,
		},
		SelfHarm: &FilterResult{
			Filtered: cfr.SelfHarm.Filtered,
			Severity: cfr.SelfHarm.Severity,
		},
		Sexual: &FilterResult{
			Filtered: cfr.Sexual.Filtered,
			Severity: cfr.Sexual.Severity,
		},
		Violence: &FilterResult{
			Filtered: cfr.Violence.Filtered,
			Severity: cfr.Violence.Severity,
		},
		JailBreak: &FilterResult{
			Filtered: cfr.JailBreak.Filtered,
			Detected: cfr.JailBreak.Detected,
		},
		Profanity: &FilterResult{
			Filtered: cfr.Profanity.Filtered,
			Detected: cfr.Profanity.Detected,
		},
	}
}

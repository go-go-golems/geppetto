package tools

import (
	"encoding/json"
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/inference/engine"
)

// ToolAdapter converts our generic ToolDefinition to provider-specific formats
type ToolAdapter interface {
	ConvertToProviderFormat(tool ToolDefinition) (interface{}, error)
	ConvertFromProviderResponse(response interface{}) ([]ToolCall, error)
	ValidateToolDefinition(tool ToolDefinition) error
	GetProviderLimits() engine.ProviderLimits
}

// OpenAIToolAdapter converts to go_openai.Tool format
type OpenAIToolAdapter struct{}

// NewOpenAIToolAdapter creates a new OpenAI tool adapter
func NewOpenAIToolAdapter() *OpenAIToolAdapter {
	return &OpenAIToolAdapter{}
}

func (a *OpenAIToolAdapter) ConvertToProviderFormat(tool ToolDefinition) (interface{}, error) {
	if err := a.ValidateToolDefinition(tool); err != nil {
		return nil, fmt.Errorf("tool validation failed: %w", err)
	}

	// Convert to OpenAI tool format
	openaiTool := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
		},
	}

	if tool.Parameters != nil {
		// Convert jsonschema.Schema to map for OpenAI
		parametersBytes, err := json.Marshal(tool.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}

		var parametersMap map[string]interface{}
		if err := json.Unmarshal(parametersBytes, &parametersMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		openaiTool["function"].(map[string]interface{})["parameters"] = parametersMap
	}

	return openaiTool, nil
}

func (a *OpenAIToolAdapter) ConvertFromProviderResponse(response interface{}) ([]ToolCall, error) {
	// This would convert OpenAI API response to our ToolCall format
	// Implementation depends on the specific OpenAI response structure
	return nil, fmt.Errorf("not implemented")
}

func (a *OpenAIToolAdapter) ValidateToolDefinition(tool ToolDefinition) error {
	limits := a.GetProviderLimits()

	if len(tool.Name) > limits.MaxToolNameLength {
		return fmt.Errorf("tool name too long: %d > %d", len(tool.Name), limits.MaxToolNameLength)
	}

	// Additional OpenAI-specific validations
	return nil
}

func (a *OpenAIToolAdapter) GetProviderLimits() engine.ProviderLimits {
	return engine.ProviderLimits{
		MaxToolsPerRequest:      64,
		MaxToolNameLength:       64,
		SupportedParameterTypes: []string{"string", "number", "integer", "boolean", "object", "array"},
	}
}

// ClaudeToolAdapter converts to Claude's tool format
type ClaudeToolAdapter struct{}

// NewClaudeToolAdapter creates a new Claude tool adapter
func NewClaudeToolAdapter() *ClaudeToolAdapter {
	return &ClaudeToolAdapter{}
}

func (a *ClaudeToolAdapter) ConvertToProviderFormat(tool ToolDefinition) (interface{}, error) {
	if err := a.ValidateToolDefinition(tool); err != nil {
		return nil, fmt.Errorf("tool validation failed: %w", err)
	}

	// Convert to Claude tool format
	claudeTool := map[string]interface{}{
		"name":        tool.Name,
		"description": tool.Description,
	}

	if tool.Parameters != nil {
		// Convert jsonschema.Schema to Claude's input_schema format
		parametersBytes, err := json.Marshal(tool.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}

		var inputSchema map[string]interface{}
		if err := json.Unmarshal(parametersBytes, &inputSchema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		claudeTool["input_schema"] = inputSchema
	}

	return claudeTool, nil
}

func (a *ClaudeToolAdapter) ConvertFromProviderResponse(response interface{}) ([]ToolCall, error) {
	// This would convert Claude API response to our ToolCall format
	return nil, fmt.Errorf("not implemented")
}

func (a *ClaudeToolAdapter) ValidateToolDefinition(tool ToolDefinition) error {
	limits := a.GetProviderLimits()

	if len(tool.Name) > limits.MaxToolNameLength {
		return fmt.Errorf("tool name too long: %d > %d", len(tool.Name), limits.MaxToolNameLength)
	}

	// Additional Claude-specific validations
	return nil
}

func (a *ClaudeToolAdapter) GetProviderLimits() engine.ProviderLimits {
	return engine.ProviderLimits{
		MaxToolsPerRequest:      20,
		MaxTotalSizeBytes:       51200, // 50KB total
		SupportedParameterTypes: []string{"string", "number", "integer", "boolean", "object", "array"},
	}
}

// GeminiToolAdapter converts to Gemini's function calling format
type GeminiToolAdapter struct{}

// NewGeminiToolAdapter creates a new Gemini tool adapter
func NewGeminiToolAdapter() *GeminiToolAdapter {
	return &GeminiToolAdapter{}
}

func (a *GeminiToolAdapter) ConvertToProviderFormat(tool ToolDefinition) (interface{}, error) {
	if err := a.ValidateToolDefinition(tool); err != nil {
		return nil, fmt.Errorf("tool validation failed: %w", err)
	}

	// Convert to Gemini function declaration format
	geminiTool := map[string]interface{}{
		"name":        tool.Name,
		"description": tool.Description,
	}

	if tool.Parameters != nil {
		// Convert jsonschema.Schema to Gemini's parameter format
		parametersBytes, err := json.Marshal(tool.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}

		var parameters map[string]interface{}
		if err := json.Unmarshal(parametersBytes, &parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		geminiTool["parameters"] = parameters
	}

	return geminiTool, nil
}

func (a *GeminiToolAdapter) ConvertFromProviderResponse(response interface{}) ([]ToolCall, error) {
	// This would convert Gemini API response to our ToolCall format
	return nil, fmt.Errorf("not implemented")
}

func (a *GeminiToolAdapter) ValidateToolDefinition(tool ToolDefinition) error {
	limits := a.GetProviderLimits()

	if len(tool.Name) > limits.MaxToolNameLength {
		return fmt.Errorf("tool name too long: %d > %d", len(tool.Name), limits.MaxToolNameLength)
	}

	// Additional Gemini-specific validations
	return nil
}

func (a *GeminiToolAdapter) GetProviderLimits() engine.ProviderLimits {
	return engine.ProviderLimits{
		MaxToolsPerRequest:      50, // Placeholder value
		MaxToolNameLength:       100,
		SupportedParameterTypes: []string{"string", "number", "integer", "boolean", "object", "array"},
	}
}

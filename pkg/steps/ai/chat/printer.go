package chat

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Package chat provides functionality for printing AI chat events in various formats.
//
// The structured printer supports three output formats:
//   - Text: Similar to the original format, with optional metadata display
//   - JSON: Structured output in JSON format
//   - YAML: Structured output in YAML format
//
// Example usage:
//
//	// Text format with metadata
//	printer := NewStructuredPrinter(os.Stdout, PrinterOptions{
//	    Format: FormatText,
//	    Name: "Assistant",
//	    IncludeMetadata: true,
//	    IncludeStepMetadata: true,
//	})
//
//	// JSON format
//	printer := NewStructuredPrinter(os.Stdout, PrinterOptions{
//	    Format: FormatJSON,
//	    IncludeMetadata: true,
//	})
//
//	// YAML format
//	printer := NewStructuredPrinter(os.Stdout, PrinterOptions{
//	    Format: FormatYAML,
//	    IncludeMetadata: true,
//	    IncludeStepMetadata: true,
//	})
//
// The structured output includes:
//   - Event type
//   - Content (varies by event type)
//   - Optional metadata
//   - Optional step metadata
//
// This enables better debugging and monitoring of the AI conversation process
// by providing detailed information about each event in a configurable format.

type PrinterFormat string

const (
	FormatText PrinterFormat = "text"
	FormatJSON PrinterFormat = "json"
	FormatYAML PrinterFormat = "yaml"
)

type PrinterOptions struct {
	// Format determines the output format (text, json, yaml)
	Format PrinterFormat
	// Name is the prefix to use for text output
	Name string
	// IncludeMetadata controls whether to include Event.Metadata() in output
	IncludeMetadata bool
	// Full controls whether to print all available metadata (overrides IncludeMetadata and IncludeStepMetadata)
	Full bool
}

type structuredOutput struct {
	Type         EventType   `json:"type,omitempty" yaml:"type,omitempty"`
	Content      interface{} `json:"content,omitempty" yaml:"content,omitempty"`
	Metadata     interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	StepMetadata interface{} `json:"step_metadata,omitempty" yaml:"step_metadata,omitempty"`
}

type ClaudeUsage struct {
	InputTokens  float64 `mapstructure:"input_tokens"`
	OutputTokens float64 `mapstructure:"output_tokens"`
}

type AISettings struct {
	Temperature float64 `mapstructure:"ai-temperature"`
	MaxTokens   float64 `mapstructure:"ai-max-response-tokens"`
	Engine      string  `mapstructure:"ai-engine"`
	ApiType     string  `mapstructure:"ai-api-type"`
	TopP        float64 `mapstructure:"ai-top-p"`
}

type StepMetadataContent struct {
	ClaudeUsage ClaudeUsage `mapstructure:"claude_usage"`
	Settings    AISettings  `mapstructure:"settings"`
	StopReason  string      `mapstructure:"claude_stop_reason"`
}

func NewStructuredPrinter(w io.Writer, options PrinterOptions) func(msg *message.Message) error {
	isFirst := true

	return func(msg *message.Message) error {
		defer msg.Ack()

		e, err := NewEventFromJson(msg.Payload)
		if err != nil {
			return err
		}

		switch options.Format {
		case FormatText:
			return handleTextFormat(w, e, options, &isFirst)
		case FormatJSON:
			return handleStructuredFormat(w, e, options, json.Marshal)
		case FormatYAML:
			return handleStructuredFormat(w, e, options, yaml.Marshal)
		default:
			return fmt.Errorf("unknown format: %s", options.Format)
		}
	}
}

func handleTextFormat(w io.Writer, e Event, options PrinterOptions, isFirst *bool) error {
	switch p := e.(type) {
	case *EventError:
		return fmt.Errorf("error event: %v", p.Error_)
	case *EventPartialCompletion:
		if *isFirst && options.Name != "" {
			*isFirst = false
			if _, err := w.Write([]byte(fmt.Sprintf("\n%s: \n", options.Name))); err != nil {
				return err
			}
		}
		_, err := w.Write([]byte(p.Delta))
		return err
	case *EventFinal, *EventText:
		// Add metadata if requested
		if options.IncludeMetadata {
			metaBytes, err := yaml.Marshal(e.Metadata())
			if err != nil {
				return err
			}
			if _, err := w.Write([]byte(fmt.Sprintf("\nMetadata:\n%s\n", metaBytes))); err != nil {
				return err
			}
			if e.StepMetadata() != nil {
				stepMetaBytes, err := yaml.Marshal(e.StepMetadata())
				if err != nil {
					return err
				}
				if _, err := w.Write([]byte(fmt.Sprintf("\nStep Metadata:\n%s\n", stepMetaBytes))); err != nil {
					return err
				}
			}
		}
		return nil
	case *EventToolCall:
		toolCallBytes, err := yaml.Marshal(p.ToolCall)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(fmt.Sprintf("%s\n", toolCallBytes)))
		return err
	case *EventToolResult:
		toolResultBytes, err := yaml.Marshal(p.ToolResult)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(fmt.Sprintf("%s\n", toolResultBytes)))
		return err
	}
	return nil
}

func handleStructuredFormat(w io.Writer, e Event, options PrinterOptions, marshal func(interface{}) ([]byte, error)) error {
	output := structuredOutput{
		Type: e.Type(),
	}

	switch p := e.(type) {
	case *EventPartialCompletion:
		output.Content = p.Delta
	case *EventFinal:
		output.Content = p.Text
	case *EventText:
		output.Content = p.Text
	case *EventToolCall:
		output.Content = p.ToolCall
	case *EventToolResult:
		output.Content = p.ToolResult
	case *EventError:
		output.Content = p.Error_
	}

	if options.Full {
		output.Metadata = e.Metadata()
		output.StepMetadata = e.StepMetadata()
	} else if options.IncludeMetadata {
		output.Metadata = e.Metadata()
		importantMeta := extractImportantMetadata(e)
		if importantMeta != nil {
			if e.Type() == EventTypeStart {
				output = structuredOutput{
					Type:    e.Type(),
					Content: importantMeta,
				}
			} else if e.Type() == EventTypeFinal {
				output.StepMetadata = importantMeta
			}
		}
	}

	bytes, err := marshal(output)
	if err != nil {
		return err
	}

	_, err = w.Write(append(bytes, '\n'))
	return err
}

func extractImportantMetadata(e Event) map[string]interface{} {
	if e.StepMetadata() == nil {
		return nil
	}

	metadata := e.Metadata()
	stepMetadata := e.StepMetadata()

	//nolint:exhaustive
	switch e.Type() {
	case EventTypeStart:
		result := map[string]interface{}{
			"type": stepMetadata.Type,
		}

		if metadata.Engine != "" {
			result["engine"] = metadata.Engine
		}
		if metadata.Temperature != 0 {
			result["temp"] = metadata.Temperature
		}
		if metadata.MaxTokens != 0 {
			result["max_tokens"] = metadata.MaxTokens
		}
		if metadata.TopP != 0 {
			result["top_p"] = metadata.TopP
		}
		if metadata.Usage.InputTokens != 0 {
			result["input_tokens"] = metadata.Usage.InputTokens
		}

		return result

	case EventTypeFinal:
		result := map[string]interface{}{
			"type": stepMetadata.Type,
		}

		if metadata.Usage.InputTokens != 0 || metadata.Usage.OutputTokens != 0 {
			result["tokens"] = map[string]interface{}{
				"in":  metadata.Usage.InputTokens,
				"out": metadata.Usage.OutputTokens,
			}
		}

		if metadata.Engine != "" {
			result["engine"] = metadata.Engine
		}
		if metadata.TopP != 0 {
			result["top_p"] = metadata.TopP
		}
		if metadata.StopReason != "" {
			result["stop_reason"] = metadata.StopReason
		}
		if metadata.Temperature != 0 {
			result["temp"] = metadata.Temperature
		}

		return result

	default:
	}
	return nil
}

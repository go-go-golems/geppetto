package events

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
	case *EventTextDelta:
		if *isFirst && options.Name != "" {
			*isFirst = false
			if _, err := fmt.Fprintf(w, "\n%s: \n", options.Name); err != nil {
				return err
			}
		}
		_, err := w.Write([]byte(p.Delta))
		return err
	case *EventReasoningDelta:
		_, err := w.Write([]byte(p.Delta))
		return err
	case *EventTextSegmentFinished:
		// Add metadata if requested
		if options.IncludeMetadata {
			metaBytes, err := yaml.Marshal(e.Metadata())
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "\nMetadata:\n%s\n", metaBytes); err != nil {
				return err
			}
			// Step metadata removed
		}
		return nil
	case *EventToolCallRequested:
		toolCallBytes, err := yaml.Marshal(map[string]any{"id": p.ToolCallID, "name": p.ToolName, "input": p.Input})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", toolCallBytes)
		return err
	case *EventToolResultReady:
		toolResultBytes, err := yaml.Marshal(map[string]any{"id": p.ToolCallID, "name": p.ToolName, "result": p.Result, "status": p.Status})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", toolResultBytes)
		return err
	case *EventLog:
		level := p.Level
		if level == "" {
			level = "info"
		}
		if _, err := fmt.Fprintf(w, "\n[%s] %s\n", level, p.Message); err != nil {
			return err
		}
		if len(p.Fields) > 0 {
			fieldsBytes, err := yaml.Marshal(p.Fields)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "%s\n", fieldsBytes); err != nil {
				return err
			}
		}
		return nil
	case *EventInfo:
		if _, err := fmt.Fprintf(w, "\n[i] %s\n", p.Message); err != nil {
			return err
		}
		if len(p.Data) > 0 {
			dataBytes, err := yaml.Marshal(p.Data)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "%s\n", dataBytes); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func handleStructuredFormat(w io.Writer, e Event, options PrinterOptions, marshal func(interface{}) ([]byte, error)) error {
	output := structuredOutput{
		Type: e.Type(),
	}

	switch p := e.(type) {
	case *EventTextDelta:
		output.Content = p.Delta
	case *EventTextSegmentFinished:
		output.Content = p.Text
	case *EventReasoningDelta:
		output.Content = p.Delta
	case *EventReasoningSegmentFinished:
		output.Content = p.Text
	case *EventToolCallRequested:
		output.Content = map[string]any{"id": p.ToolCallID, "name": p.ToolName, "input": p.Input}
	case *EventToolResultReady:
		output.Content = map[string]any{"id": p.ToolCallID, "name": p.ToolName, "result": p.Result, "status": p.Status}
	case *EventError:
		output.Content = p.Error_
	case *EventLog:
		output.Content = map[string]interface{}{
			"level":   p.Level,
			"message": p.Message,
			"fields":  p.Fields,
		}
	case *EventInfo:
		output.Content = map[string]interface{}{
			"message": p.Message,
			"data":    p.Data,
		}
	}

	if options.Full {
		output.Metadata = e.Metadata()
	} else if options.IncludeMetadata {
		output.Metadata = e.Metadata()
		importantMeta := extractImportantMetadata(e.Metadata())
		if importantMeta != nil && e.Type() == EventTypeTextSegmentFinished {
			// include on content side when the text segment finishes
			output.Content = map[string]interface{}{"final": output.Content, "meta": importantMeta}
		}
	}

	bytes, err := marshal(output)
	if err != nil {
		return err
	}

	_, err = w.Write(append(bytes, '\n'))
	return err
}

func extractImportantMetadata(metadata EventMetadata) map[string]interface{} {

	//nolint:exhaustive
	// provide a compact subset for lifecycle summary output
	{
		result := map[string]interface{}{
			"model": metadata.Model,
		}
		if metadata.Usage != nil {
			result["input_tokens"] = metadata.Usage.InputTokens
			result["output_tokens"] = metadata.Usage.OutputTokens
		}
		if metadata.StopReason != nil {
			result["stop_reason"] = *metadata.StopReason
		}
		if metadata.DurationMs != nil {
			result["duration_ms"] = *metadata.DurationMs
		}

		return result
	}
}

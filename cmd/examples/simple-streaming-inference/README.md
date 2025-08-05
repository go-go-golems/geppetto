# Simple Streaming Inference Example

This example demonstrates how to implement streaming inference using the Engine-first architecture with event routing and sinks.

## Features

- **Streaming Events**: Real-time streaming of inference events using Watermill message bus
- **Multiple Output Formats**: Support for text, JSON, and YAML output formats
- **Event Router**: Coordinated event routing with proper cleanup and error handling
- **Configurable Sinks**: Flexible sink system for event distribution
- **Structured Output**: Rich metadata and formatting options
- **Debugging Support**: Verbose logging and debug modes

## Architecture

The implementation follows the streaming inference pattern described in the documentation:

1. **Event Router**: Creates and manages the Watermill message bus
2. **Watermill Sink**: Publishes events to specific topics
3. **Engine with Sink**: Inference engine configured with event sinks
4. **Printer Handlers**: Convert events to human-readable output
5. **Coordinated Execution**: Parallel execution with proper cleanup

## Usage

### Basic Streaming

```bash
./simple-streaming-inference simple-streaming-inference "Your prompt here" --ai-stream=true
```

### JSON Output with Metadata

```bash
./simple-streaming-inference simple-streaming-inference "Your prompt" \
  --ai-stream=true \
  --output-format=json \
  --with-metadata=true
```

### YAML Output with Full Details

```bash
./simple-streaming-inference simple-streaming-inference "Your prompt" \
  --ai-stream=true \
  --output-format=yaml \
  --full-output=true \
  --with-metadata=true
```

### Verbose Logging

```bash
./simple-streaming-inference simple-streaming-inference "Your prompt" \
  --ai-stream=true \
  --verbose=true \
  --with-logging=true
```

## Command Line Options

### Streaming Options

- `--output-format`: Output format (text, json, yaml) - default: "text"
- `--with-metadata`: Include metadata in output - default: false
- `--full-output`: Include full output details - default: false
- `--verbose`: Verbose event router logging - default: false

### Debug Options

- `--debug`: Debug mode - show parsed layers - default: false
- `--with-logging`: Enable logging middleware - default: false

### AI Configuration

- `--ai-stream`: Whether to stream responses - default: false
- `--ai-engine`: The model to use for chat
- `--ai-temperature`: Temperature for response generation
- `--ai-max-response-tokens`: Maximum number of tokens in the response

## Event Flow

1. **Engine** runs inference and generates events
2. **EventSink** (WatermillSink) publishes events to watermill topic
3. **EventRouter** routes events to registered handlers
4. **Handlers** (StepPrinterFunc/StructuredPrinter) process and display events

## Event Types

- `EventPartialCompletion` - Streaming text chunks
- `EventFinal` - Final completion
- `EventError` - Error events
- `EventToolCall` - Tool call events
- `EventToolResult` - Tool result events

## Implementation Details

The example demonstrates:

- Proper resource cleanup with defer statements
- Context cancellation for coordinated shutdown
- Error group usage for parallel execution
- Event sink configuration and management
- Structured printer configuration
- Middleware integration for logging

## Comparison with Simple Inference

This streaming version extends the basic simple inference example by adding:

- Event routing infrastructure
- Streaming event handling
- Multiple output format support
- Rich metadata and debugging options
- Coordinated parallel execution

The core inference logic remains the same, but now supports real-time streaming with proper event distribution and handling. 
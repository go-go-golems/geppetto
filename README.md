# Geppetto - Go LLM Application Framework

![Retro cybernetic puppetmaster controlling a pinocchio puppet that is working on a computer, retro mainframe aesthetic](geppetto.jpg)

Geppetto is a Go framework for building LLM (Large Language Model) applications. It provides a comprehensive set of tools for conversation management, embedding generation, step-based processing, and AI integration with multiple providers.

## Features

- **Multi-Provider AI Support**: OpenAI, Claude, Gemini, and Ollama integration
- **Conversation Management**: Tree-based conversation structure with branching support
- **Step-Based Processing**: Composable steps for building complex LLM workflows
- **Embedding Generation**: With caching support for improved performance
- **Event System**: Comprehensive event handling for monitoring and debugging
- **JavaScript API**: JavaScript bindings for embedding runtime
- **Streaming Support**: Real-time response handling

## Architecture

Geppetto is built around several core packages:

- [`pkg/conversation`](pkg/conversation): Tree-based conversation management
- [`pkg/steps`](pkg/steps): Step-based processing framework with AI providers
- [`pkg/embeddings`](pkg/embeddings): Embedding generation with caching
- [`pkg/events`](pkg/events): Event system for monitoring and debugging
- [`pkg/js`](pkg/js): JavaScript runtime integration

## Installation

To use Geppetto as a library in your Go project:

```bash
go get github.com/go-go-golems/geppetto
```

## Requirements

- Go 1.24.2 or later
- API keys for your chosen AI providers (OpenAI, Claude, Gemini, etc.)

## Basic Usage

### Creating a Simple Chat Application

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/go-go-golems/geppetto/pkg/conversation"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
)

func main() {
    ctx := context.Background()
    
    // Create conversation manager
    manager := conversation.NewManager()
    
    // Create OpenAI chat step
    settings := openai.NewSettings(
        openai.WithAPIKey("your-api-key"),
        openai.WithModel("gpt-4"),
    )
    
    chatStep := openai.NewChatStep(settings)
    
    // Add user message
    userMsg := conversation.NewChatMessage(conversation.RoleUser, "Hello, how are you?")
    manager.AppendMessages(userMsg)
    
    // Execute AI step
    result := chatStep.Start(ctx, manager.GetConversation())
    
    // Handle response
    for res := range result.GetChannel() {
        if res.Error != nil {
            log.Fatal(res.Error)
        }
        fmt.Printf("Assistant: %s\n", res.Value)
    }
}
```

### Working with Embeddings

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/go-go-golems/geppetto/pkg/embeddings/config"
)

func main() {
    ctx := context.Background()
    
    // Create embedding settings
    settings := config.EmbeddingSettings{
        Engine: config.EmbeddingEngineOpenAI,
        OpenAI: &config.OpenAISettings{
            APIKey: "your-api-key",
            Model:  "text-embedding-3-small",
        },
    }
    
    // Create embedder with caching
    embedder := embeddings.NewCachedEmbedder(
        embeddings.NewOpenAIEmbedder(settings.OpenAI),
        embeddings.NewDiskCache("./cache"),
    )
    
    // Generate embeddings
    texts := []string{"Hello world", "How are you?"}
    vectors, err := embedder.EmbedTexts(ctx, texts)
    if err != nil {
        log.Fatal(err)
    }
    
    for i, vector := range vectors {
        fmt.Printf("Text: %s, Embedding length: %d\n", texts[i], len(vector))
    }
}
```

## Building and Testing

```bash
# Build the project
make build

# Run tests
make test

# Run linting
make lint

# Run a specific test
go test ./pkg/embeddings -run TestBasicCacheOperations
```

## Documentation

Comprehensive documentation is available in the [`pkg/doc/topics`](pkg/doc/topics) directory:

- [Conversation Package Guide](pkg/doc/topics/05-conversation.md)
- [Embeddings Package Guide](pkg/doc/topics/06-embeddings.md)
- [Steps Framework Guide](pkg/doc/topics/07-steps.md)
- [Events System Guide](pkg/doc/topics/04-events.md)
- [JavaScript API Guide](pkg/doc/topics/08-javascript-api.md)

## Examples

The [`examples`](examples) directory contains practical examples showing how to use various features of the framework.

## Contributing

This is a GO GO GOLEMS project. The structure and API may change as the project evolves, but the core concepts of conversation management and step-based processing will remain stable.

## License

See [LICENSE](LICENSE) for details.

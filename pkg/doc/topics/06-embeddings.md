---
Title: Understanding and Using the Embeddings Package
Slug: geppetto-embeddings-package
Short: A guide to the embeddings package in Geppetto, covering embedding generation, caching, and integrating with applications.
Topics:
- geppetto
- embeddings
- vector
- ai
- tutorial
- semantic search
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Understanding and Using the Embeddings Package

This tutorial provides a comprehensive guide to using the `embeddings` package in Geppetto, which enables applications to generate and work with vector embeddings for text. The package provides a robust set of interfaces, implementations, and utilities for working with different embedding providers, caching strategies, and application integration.

## Core Concepts

The embeddings package is built around several key types:

- `Provider`: Interface for generating embeddings from text
- `EmbeddingModel`: Contains metadata about the embedding model being used
- `CachedProvider`: In-memory LRU cache wrapper for embedding providers
- `DiskCacheProvider`: Persistent disk-based cache for embedding providers
- `SettingsFactory`: Creates embedding providers based on configuration

Embeddings are numerical vector representations of text that capture semantic meaning, enabling operations like similarity comparison. These vectors typically contain hundreds or thousands of floating-point values.

### Provider Interface

The core interface that all embedding providers implement:

```go
type Provider interface {
    // GenerateEmbedding creates an embedding vector for the given text
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
    
    // GetModel returns information about the embedding model being used
    GetModel() EmbeddingModel
}
```

### Embedding Model

The metadata structure for embedding models:

```go
type EmbeddingModel struct {
    Name       string  // The name of the model
    Dimensions int     // The number of dimensions in the embedding vector
}
```

## Working with Embedding Providers

### Creating a Basic Provider

The package includes implementations for popular embedding providers:

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/sashabaranov/go-openai"
)

func main() {
    // Create an OpenAI embedding provider
    apiKey := os.Getenv("OPENAI_API_KEY")
    provider := embeddings.NewOpenAIProvider(
        apiKey,                        // OpenAI API key
        openai.SmallEmbedding3,        // Model to use
        1536,                          // Vector dimensions
    )
    
    // Generate an embedding
    ctx := context.Background()
    embedding, err := provider.GenerateEmbedding(ctx, "Hello, world!")
    if err != nil {
        fmt.Printf("Error generating embedding: %v\n", err)
        return
    }
    
    // Print the first few dimensions
    fmt.Printf("Generated %d-dimensional embedding. First 5 values: %v\n", 
        len(embedding), embedding[:5])
}
```

### Using Ollama for Local Embeddings

For local or self-hosted scenarios, you can use Ollama:

```go
// Create an Ollama embedding provider
ollamaProvider := embeddings.NewOllamaProvider(
    "http://localhost:11434",   // Base URL for Ollama API
    "all-minilm",               // Model to use
    384,                        // Vector dimensions for this model
)

// Generate an embedding
embedding, err := ollamaProvider.GenerateEmbedding(ctx, "Hello, world!")
if err != nil {
    fmt.Printf("Error generating embedding: %v\n", err)
    return
}
```

## Implementing Caching Strategies

API calls to embedding providers can be expensive and slow. The embeddings package provides caching mechanisms to improve performance.

### In-Memory Caching

For short-lived applications, in-memory caching is efficient:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/sashabaranov/go-openai"
)

func main() {
    // Create a base provider
    provider := embeddings.NewOpenAIProvider(
        "your-api-key",
        openai.SmallEmbedding3,
        1536,
    )
    
    // Wrap with a cached provider (storing up to 1000 embeddings)
    cachedProvider := embeddings.NewCachedProvider(provider, 1000)
    
    // First call - will hit the API
    ctx := context.Background()
    embedding1, _ := cachedProvider.GenerateEmbedding(ctx, "Hello, world!")
    
    // Second call with the same text - will use cache
    embedding2, _ := cachedProvider.GenerateEmbedding(ctx, "Hello, world!")
    
    // Clear the cache if needed
    cachedProvider.ClearCache()
    
    // Get current cache stats
    size := cachedProvider.Size()
    maxSize := cachedProvider.MaxSize()
    fmt.Printf("Cache contains %d/%d entries\n", size, maxSize)
}
```

### Persistent Disk Caching

For long-running applications or CLI tools, disk caching can persist embeddings across runs:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

func main() {
    // Create a base provider
    provider := createBaseProvider() // Implementation omitted for brevity
    
    // Create a disk cache provider with options
    diskProvider, err := embeddings.NewDiskCacheProvider(
        provider,
        embeddings.WithDirectory("./cache/embeddings"),  // Custom directory
        embeddings.WithMaxSize(1<<30),                  // 1GB max size
        embeddings.WithMaxEntries(10000),               // 10,000 entries max
    )
    if err != nil {
        fmt.Printf("Error creating disk cache: %v\n", err)
        return
    }
    
    // Generate an embedding (will be cached to disk)
    ctx := context.Background()
    text := "This is a test of persistent caching"
    embedding, _ := diskProvider.GenerateEmbedding(ctx, text)
    
    // On subsequent runs, this will be loaded from disk
    
    // Clear the cache if needed
    _ = diskProvider.ClearCache()
}
```

## Working with Settings and Configuration

The embeddings package integrates with Geppetto's configuration system for easy setup.

### Using the Settings Factory

The `SettingsFactory` creates providers based on application configuration:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/go-go-golems/geppetto/pkg/embeddings/config"
)

func main() {
    // Create a configuration
    embeddingsConfig := &config.EmbeddingsConfig{
        Type:          "openai",
        Engine:        "text-embedding-3-small",
        Dimensions:    1536,
        CacheType:     "memory",
        CacheMaxEntries: 1000,
        APIKeys: map[string]string{
            "openai-api-key": "your-api-key",
        },
    }
    
    // Create a factory
    factory := embeddings.NewSettingsFactory(embeddingsConfig)
    
    // Create a provider using the factory
    provider, err := factory.NewProvider()
    if err != nil {
        fmt.Printf("Error creating provider: %v\n", err)
        return
    }
    
    // Use the provider
    ctx := context.Background()
    embedding, _ := provider.GenerateEmbedding(ctx, "Hello, world!")
    
    // Override settings for a specific use case
    customProvider, _ := factory.NewProvider(
        embeddings.WithType("ollama"),
        embeddings.WithEngine("all-minilm"),
        embeddings.WithDimensions(384),
        embeddings.WithBaseURL("http://localhost:11434"),
    )
    
    // Use the custom provider
    customEmbedding, _ := customProvider.GenerateEmbedding(ctx, "Hello, world!")
    
    fmt.Printf("Original dimensions: %d, Custom dimensions: %d\n", 
        len(embedding), len(customEmbedding))
}
```

### Loading Settings from Parsed Layers

For CLI applications using Glazed's parameter system:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
)

func createProviderFromLayers(parsedLayers *layers.ParsedLayers) (embeddings.Provider, error) {
    // Create a factory from parsed layers
    factory, err := embeddings.NewSettingsFactoryFromParsedLayers(parsedLayers)
    if err != nil {
        return nil, fmt.Errorf("failed to create factory: %w", err)
    }
    
    // Create a provider using the factory
    provider, err := factory.NewProvider()
    if err != nil {
        return nil, fmt.Errorf("failed to create provider: %w", err)
    }
    
    return provider, nil
}
```

### Creating Provider from StepSettings

If you already have a populated `settings.StepSettings` object (perhaps loaded via layers or other means), you can also create an embeddings provider directly from it. This is useful when integrating embedding functionality into components that primarily deal with AI step configurations.

```go
package main

import (
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/embeddings"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

func createProviderFromStepSettings(stepSettings *settings.StepSettings) (embeddings.Provider, error) {
	// Create a factory directly from StepSettings
	embeddingFactory := embeddings.NewSettingsFactoryFromStepSettings(stepSettings)
	
	// Create the provider using the derived factory
	provider, err := embeddingFactory.NewProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create provider from step settings: %w", err)
	}
	
	return provider, nil
}
```

## Practical Applications

### Computing Text Similarity

One common use case for embeddings is to compute similarity between texts:

```go
package main

import (
    "context"
    "fmt"
    "math"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

// computeCosineSimilarity calculates cosine similarity between two vectors
func computeCosineSimilarity(a, b []float32) float64 {
    var dotProduct float64
    var normA float64
    var normB float64
    
    for i := 0; i < len(a); i++ {
        dotProduct += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func main() {
    // Create a provider (implementation omitted)
    provider := createProvider()
    
    ctx := context.Background()
    
    // Generate embeddings for two texts
    text1 := "Golang is a programming language designed at Google."
    text2 := "Go is a statically typed compiled language."
    
    embedding1, _ := provider.GenerateEmbedding(ctx, text1)
    embedding2, _ := provider.GenerateEmbedding(ctx, text2)
    
    // Compute similarity
    similarity := computeCosineSimilarity(embedding1, embedding2)
    
    fmt.Printf("Similarity between texts: %.4f\n", similarity)
}
```

### Building a Semantic Search System

Embeddings enable semantic search capabilities:

```go
package main

import (
    "context"
    "fmt"
    "sort"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

type Document struct {
    ID       string
    Text     string
    Embedding []float32
}

type SearchResult struct {
    Document  Document
    Similarity float64
}

func main() {
    // Create a provider
    provider := createProvider()
    
    // Create a document collection
    documents := []Document{
        {ID: "doc1", Text: "Golang is a statically typed language."},
        {ID: "doc2", Text: "Python is a dynamically typed language."},
        {ID: "doc3", Text: "JavaScript runs in the browser."},
        {ID: "doc4", Text: "TypeScript adds types to JavaScript."},
    }
    
    // Precompute embeddings for all documents
    ctx := context.Background()
    for i := range documents {
        embedding, _ := provider.GenerateEmbedding(ctx, documents[i].Text)
        documents[i].Embedding = embedding
    }
    
    // Search query
    query := "Which programming languages have static typing?"
    queryEmbedding, _ := provider.GenerateEmbedding(ctx, query)
    
    // Search for similar documents
    var results []SearchResult
    for _, doc := range documents {
        similarity := computeCosineSimilarity(queryEmbedding, doc.Embedding)
        results = append(results, SearchResult{
            Document:   doc,
            Similarity: similarity,
        })
    }
    
    // Sort by similarity (descending)
    sort.Slice(results, func(i, j int) bool {
        return results[i].Similarity > results[j].Similarity
    })
    
    // Print top results
    fmt.Println("Search results:")
    for i, result := range results {
        fmt.Printf("%d. %s (similarity: %.4f)\n",
            i+1, result.Document.Text, result.Similarity)
    }
}
```

## Integrating with CLI Applications

### Creating Parameter Layers

To integrate embeddings with a CLI application:

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/go-go-golems/geppetto/pkg/embeddings/config"
    "github.com/go-go-golems/glazed/pkg/cli"
    "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
    "github.com/go-go-golems/glazed/pkg/cmds/middlewares"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
    "github.com/spf13/cobra"
)

// GetEmbeddingsLayers returns parameter layers for embeddings
func GetEmbeddingsLayers() ([]layers.ParameterLayer, error) {
    // Create embeddings parameter layer
    embeddingsLayer, err := config.NewEmbeddingsParameterLayer()
    if err != nil {
        return nil, err
    }
    
    // Create API key parameter layer
    embeddingsApiKey, err := config.NewEmbeddingsApiKeyParameter()
    if err != nil {
        return nil, err
    }
    
    return []layers.ParameterLayer{
        embeddingsLayer,
        embeddingsApiKey,
    }, nil
}

// EmbeddingsCommand implements a command that works with embeddings
type EmbeddingsCommand struct {
    *cmds.CommandDescription
}

// NewEmbeddingsCommand creates a new command for generating embeddings
func NewEmbeddingsCommand() (*EmbeddingsCommand, error) {
    parametersLayers, err := GetEmbeddingsLayers()
    if err != nil {
        return nil, err
    }
    
    command := &EmbeddingsCommand{
        CommandDescription: cmds.NewCommandDescription(
            "embeddings",
            cmds.WithShortDescription("Generate embeddings for text"),
            cmds.WithLongDescription("Generate vector embeddings for text using various providers"),
            cmds.WithLayersList(parametersLayers),
            cmds.WithArguments(
                parameters.NewParameterDefinition(
                    "text",
                    parameters.ParameterTypeString,
                    parameters.WithHelp("Text to generate embeddings for"),
                    parameters.WithRequired(true),
                ),
            ),
        ),
    }
    
    return command, nil
}

// RunIntoGlazeProcessor runs the embeddings command
func (c *EmbeddingsCommand) RunIntoGlazeProcessor(
    ctx context.Context,
    parsedLayers *layers.ParsedLayers,
    gp middlewares.GlazeProcessor,
) error {
    // Get argument
    text, err := parsedLayers.GetString("text")
    if err != nil {
        return err
    }
    
    // Create factory from parsed layers
    factory, err := embeddings.NewSettingsFactoryFromParsedLayers(parsedLayers)
    if err != nil {
        return err
    }
    
    // Create provider
    provider, err := factory.NewProvider()
    if err != nil {
        return err
    }
    
    // Generate embedding
    embedding, err := provider.GenerateEmbedding(ctx, text)
    if err != nil {
        return err
    }
    
    // Output result
    err = gp.WriteRow(map[string]interface{}{
        "text":       text,
        "dimensions": len(embedding),
        "model":      provider.GetModel().Name,
        "embedding":  embedding,
    })
    
    return err
}

func main() {
    // Create root command
    rootCmd := &cobra.Command{
        Use:   "embeddings-tool",
        Short: "Tool for working with embeddings",
    }
    
    // Create embeddings command
    embeddingsCmd, err := NewEmbeddingsCommand()
    if err != nil {
        fmt.Printf("Error creating command: %v\n", err)
        os.Exit(1)
    }
    
    // Convert to cobra command
    cobraCmd, err := cli.BuildCobraCommandFromCommand(embeddingsCmd)
    if err != nil {
        fmt.Printf("Error building cobra command: %v\n", err)
        os.Exit(1)
    }
    
    // Add to root
    rootCmd.AddCommand(cobraCmd)
    
    // Execute
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

### Complete Application Example

Here's an example of a complete application that implements text similarity comparison using the embeddings package:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    
    clay "github.com/go-go-golems/clay/pkg"
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/go-go-golems/geppetto/pkg/embeddings/config"
    "github.com/go-go-golems/glazed/pkg/cli"
    "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/cmds/layers"
    "github.com/go-go-golems/glazed/pkg/cmds/middlewares"
    "github.com/go-go-golems/glazed/pkg/cmds/parameters"
    "github.com/spf13/cobra"
)

type CompareCommand struct {
    *cmds.CommandDescription
}

func NewCompareCommand() (*CompareCommand, error) {
    parametersLayers, err := GetEmbeddingsLayers()
    if err != nil {
        return nil, err
    }
    
    return &CompareCommand{
        CommandDescription: cmds.NewCommandDescription(
            "compare",
            cmds.WithShortDescription("Compare similarity between texts"),
            cmds.WithLongDescription("Generate embeddings and compute cosine similarity between texts"),
            cmds.WithLayersList(parametersLayers),
            cmds.WithArguments(
                parameters.NewParameterDefinition(
                    "text1",
                    parameters.ParameterTypeString,
                    parameters.WithHelp("First text to compare"),
                    parameters.WithRequired(true),
                ),
                parameters.NewParameterDefinition(
                    "text2",
                    parameters.ParameterTypeString,
                    parameters.WithHelp("Second text to compare"),
                    parameters.WithRequired(true),
                ),
            ),
        ),
    }
}

func (c *CompareCommand) RunIntoGlazeProcessor(
    ctx context.Context,
    parsedLayers *layers.ParsedLayers,
    gp middlewares.GlazeProcessor,
) error {
    // Get arguments
    text1, err := parsedLayers.GetString("text1")
    if err != nil {
        return err
    }
    
    text2, err := parsedLayers.GetString("text2")
    if err != nil {
        return err
    }
    
    // Create factory from parsed layers
    factory, err := embeddings.NewSettingsFactoryFromParsedLayers(parsedLayers)
    if err != nil {
        return err
    }
    
    // Create provider
    provider, err := factory.NewProvider()
    if err != nil {
        return err
    }
    
    // Generate embeddings
    embedding1, err := provider.GenerateEmbedding(ctx, text1)
    if err != nil {
        return err
    }
    
    embedding2, err := provider.GenerateEmbedding(ctx, text2)
    if err != nil {
        return err
    }
    
    // Calculate similarity
    similarity := computeCosineSimilarity(embedding1, embedding2)
    
    // Output result
    return gp.WriteRow(map[string]interface{}{
        "text1":      text1,
        "text2":      text2,
        "similarity": similarity,
        "model":      provider.GetModel().Name,
    })
}

func main() {
    // Create root command
    rootCmd := &cobra.Command{
        Use:   "text-compare",
        Short: "Compare text similarity using embeddings",
    }
    
    // Initialize Viper for config file support
    err := clay.InitViper("text-compare", rootCmd)
    if err != nil {
        fmt.Printf("Failed to initialize Viper: %v\n", err)
        os.Exit(1)
    }
    
    // Create compare command
    compareCmd, err := NewCompareCommand()
    if err != nil {
        fmt.Printf("Error creating command: %v\n", err)
        os.Exit(1)
    }
    
    // Convert to cobra command and register
    cobraCmd, err := cli.BuildCobraCommandFromCommand(compareCmd)
    if err != nil {
        fmt.Printf("Error building cobra command: %v\n", err)
        os.Exit(1)
    }
    rootCmd.AddCommand(cobraCmd)
    
    // Create context with signal handling
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
    
    // Execute with context
    if err := rootCmd.ExecuteContext(ctx); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
```

## Using Embeddings with LLM Agents

The embeddings package integrates with Geppetto's LLM agents:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/go-go-golems/geppetto/pkg/llm"
)

// LLMWithEmbeddings implements both LLM and embedding functionality
type LLMWithEmbeddings struct {
    llmProvider      llm.LLM
    embeddingProvider embeddings.Provider
}

// Ensure it implements both interfaces
var _ llm.LLM = &LLMWithEmbeddings{}
var _ embeddings.Provider = &LLMWithEmbeddings{}

// Generate delegates to the LLM provider
func (l *LLMWithEmbeddings) Generate(ctx context.Context, messages []*conversation.Message) (string, error) {
    return l.llmProvider.Generate(ctx, messages)
}

// GenerateWithStream delegates to the LLM provider
func (l *LLMWithEmbeddings) GenerateWithStream(ctx context.Context, messages []*conversation.Message) (<-chan string, error) {
    return l.llmProvider.GenerateWithStream(ctx, messages)
}

// GenerateEmbedding delegates to the embedding provider
func (l *LLMWithEmbeddings) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
    return l.embeddingProvider.GenerateEmbedding(ctx, text)
}

// GetModel returns the embedding model information
func (l *LLMWithEmbeddings) GetModel() embeddings.EmbeddingModel {
    return l.embeddingProvider.GetModel()
}

func main() {
    // Create providers
    llmProvider := createLLMProvider()
    embeddingProvider := createEmbeddingProvider()
    
    // Create combined provider
    combined := &LLMWithEmbeddings{
        llmProvider:      llmProvider,
        embeddingProvider: embeddingProvider,
    }
    
    // Now you can use the combined provider with agents that need both capabilities
    useWithAgent(combined)
}
```

## Best Practices

When working with the embeddings package, keep these best practices in mind:

- **Vector Dimensions**: Be aware of the embedding dimensions for your chosen model:
  - OpenAI text-embedding-3-small: 1536 dimensions
  - Ollama all-minilm: 384 dimensions
- **Configuration**: Use the parameter system to allow flexible configuration of embedding providers.
- **Storage**: For large-scale applications, consider using vector databases instead of in-memory storage.

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
- emrichen
- templates
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Understanding and Using the Embeddings Package

## What Are Embeddings?

Embeddings are numerical vector representations of text that capture semantic meaning. Two texts with similar meanings will have vectors that are "close" in the embedding space, even if they use different words.

**Use cases:**
- **Semantic search** — find documents by meaning, not just keywords
- **Clustering** — group similar texts together
- **Classification** — categorize text based on content
- **Recommendations** — find similar items

Example: "The cat sat on the mat" and "A feline rested on the rug" would have similar embedding vectors because they mean similar things.

## Quick Start

```go
resolved, err := profilebootstrap.ResolveCLIEngineSettings(ctx, parsedValues)
if err != nil { panic(err) }
defer resolved.Close()

if err := embeddings.ValidateInferenceSettingsForEmbeddings(resolved.FinalInferenceSettings); err != nil {
    panic(err)
}

factory := embeddings.NewSettingsFactoryFromInferenceSettings(resolved.FinalInferenceSettings)
provider, err := factory.NewProvider()
if err != nil { panic(err) }

vector, err := provider.GenerateEmbedding(ctx, "Hello, world!")
if err != nil { panic(err) }

fmt.Printf("Generated %d-dimensional vector\n", len(vector))
```

Use `--profile-registries ~/.config/pinocchio/profiles.yaml` and select an embedding-capable profile such as `openai-embedding-small` or `ollama-nomic-embedding`.

## Core Concepts

### Provider Interface

All embedding providers implement:

```go
type Provider interface {
    // Single text → single vector
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

    // Multiple texts → multiple vectors (more efficient)
    GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
    
    // Model metadata
    GetModel() EmbeddingModel
}

type EmbeddingModel struct {
    Name       string // e.g., "text-embedding-3-small"
    Dimensions int    // e.g., 1536
}
```

### Choosing a Provider

| Provider | Model | Dimensions | Use Case |
|----------|-------|------------|----------|
| **OpenAI** | `text-embedding-3-small` | 1536 | Best quality, requires API key |
| **OpenAI** | `text-embedding-3-large` | 3072 | Higher quality, higher cost |
| **Ollama** | `all-minilm` | 384 | Local, no API key needed |
| **Ollama** | `nomic-embed-text` | 768 | Local, higher quality |

## Working with Embedding Providers

### Creating an OpenAI Provider

```go
package main

import (
    "context"
    "fmt"

    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

func main() {
    // Resolve a profile that stacks the OpenAI base credentials from
    // ~/.config/pinocchio/profiles.yaml, then build the provider from the
    // final merged InferenceSettings.
    resolved := resolveEmbeddingProfile("openai-embedding-small") // Application-specific bootstrap.
    if err := embeddings.ValidateInferenceSettingsForEmbeddings(resolved.FinalInferenceSettings); err != nil {
        panic(err)
    }
    provider, err := embeddings.NewSettingsFactoryFromInferenceSettings(resolved.FinalInferenceSettings).NewProvider()
    if err != nil {
        panic(err)
    }
    
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

## Caching Strategies

Embedding API calls are **expensive** (cost money) and **slow** (network latency). Caching dramatically improves both.

| Cache Type | Best For | Persists? | Trade-offs |
|------------|----------|-----------|------------|
| **None** | Always-fresh embeddings | — | Most expensive, slowest |
| **Memory** | Scripts, tests, short-lived processes | No | Fast, lost on restart |
| **File** | CLI tools, long-running apps | Yes | Persists, uses disk space |

### In-Memory Caching

For short-lived processes where same texts are embedded multiple times:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

func main() {
    // Create a base provider from a resolved embedding profile. The profile stack
    // supplies provider credentials and model settings.
    resolved := resolveEmbeddingProfile("openai-embedding-small")
    provider, err := embeddings.NewSettingsFactoryFromInferenceSettings(resolved.FinalInferenceSettings).NewProvider()
    if err != nil {
        panic(err)
    }

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

### Persistent File Caching

For CLI tools or applications that restart frequently:

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
    
    // Create a file-backed cache provider with options
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

Default cache location: `~/.geppetto/cache/embeddings/<model>/`

## Batch Embeddings

For multiple texts, batch calls are more efficient:

```go
package main

import (
    "context"
    "fmt"

    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

func main() {
    provider := createProvider() // Implementation omitted for brevity

    texts := []string{"one", "two", "three"}
    vectors, err := provider.GenerateBatchEmbeddings(context.Background(), texts)
    if err != nil {
        fmt.Printf("batch embeddings error: %v\n", err)
        return
    }
    fmt.Printf("generated %d vectors\n", len(vectors))
}
```

If you only have `GenerateEmbedding`, use the helper:

```go
vectors, err := embeddings.ParallelGenerateBatchEmbeddings(ctx, provider, texts, 4)
```

## Configuration via Settings

### Using EmbeddingsConfig

```go
import "github.com/go-go-golems/geppetto/pkg/embeddings/config"

resolved := resolveEmbeddingProfile("openai-embedding-small") // Application-specific bootstrap.
embeddingsConfig := &config.EmbeddingsConfig{
    Type:            "openai",
    Engine:          "text-embedding-3-small",
    Dimensions:      1536,
    CacheType:       "file",  // "none", "memory", or "file"
    CacheMaxEntries: 10000,
    // Prefer NewSettingsFactoryFromInferenceSettings with a resolved profile.
    // If constructing EmbeddingsConfig directly, pass keys from the already
    // resolved profile settings rather than reading provider environment variables.
    APIKeys: resolved.FinalInferenceSettings.API.APIKeys,
}

factory := embeddings.NewSettingsFactory(embeddingsConfig)
provider, err := factory.NewProvider()
```

### CLI Flags

When using Glazed-based CLIs:

```bash
mycommand --embeddings-type=openai \
          --embeddings-engine=text-embedding-3-small \
          --embeddings-cache-type=file \
          --embeddings-cache-max-entries=10000
```

Available flags:
- `--embeddings-type` — `openai` or `ollama`
- `--embeddings-engine` — model name
- `--embeddings-dimensions` — vector size (required for Ollama)
- `--embeddings-cache-type` — `none`, `memory`, or `file`
- `--embeddings-cache-max-size` — max bytes for file cache
- `--embeddings-cache-max-entries` — max entries
- `--embeddings-cache-directory` — custom cache path

### Using Embeddings from Engine Profiles

Applications that already expose Geppetto engine profile flags should prefer profile-backed embedding configuration over application-specific API-key flags. The application selects and resolves a profile, validates that the final merged settings are embedding-capable, and then asks the embeddings package to construct the provider.

A reusable OpenAI embedding profile should stack a provider base profile that owns the key and add only embedding-specific settings:

```yaml
profiles:
  openai-base:
    inference_settings:
      api:
        api_keys:
          openai-api-key: <configured in ~/.config/pinocchio/profiles.yaml>

  openai-embedding-small:
    stack:
      - profile_slug: openai-base
    display_name: OpenAI text-embedding-3-small
    description: Embedding profile for semantic search and vector indexing.
    inference_settings:
      embeddings:
        type: openai
        engine: text-embedding-3-small
        dimensions: 1536
        cache_type: file
        cache_directory: ./.geppetto/embeddings-cache/openai-text-embedding-3-small
```

A local Ollama profile can carry the base URL and embedding model in the same profile, or stack a shared Ollama base profile:

```yaml
profiles:
  ollama-nomic-embedding:
    display_name: Ollama nomic-embed-text embeddings
    description: Local embedding profile for private smoke tests.
    inference_settings:
      api:
        base_urls:
          ollama-base-url: http://localhost:11434
      embeddings:
        type: ollama
        engine: nomic-embed-text
        dimensions: 768
        cache_type: file
        cache_directory: ./.geppetto/embeddings-cache/ollama-nomic-embed-text
```

The canonical profile cache keys are `cache_type`, `cache_directory`,
`cache_max_size`, and `cache_max_entries`. `cache_type: file` constructs a
disk-backed provider when the profile is resolved through
`NewSettingsFactoryFromInferenceSettings`; repeated embedding requests can then
reuse entries across process restarts. Use an explicit `cache_directory` for
reproducible CLI and experiment storage.

Consumer code should validate the resolved final settings before provider construction:

```go
resolved, err := profilebootstrap.ResolveCLIEngineSettings(ctx, parsedValues)
if err != nil {
    return err
}
defer resolved.Close()

if err := embeddings.ValidateInferenceSettingsForEmbeddings(resolved.FinalInferenceSettings); err != nil {
    return err
}

factory := embeddings.NewSettingsFactoryFromInferenceSettings(resolved.FinalInferenceSettings)
provider, err := factory.NewProvider()
if err != nil {
    return err
}
```

A chat profile is not automatically an embedding profile. For example, a profile that sets `inference_settings.chat.engine: gpt-5` can be valid for chat while still missing `inference_settings.embeddings`. Vector-search commands should report that shape problem and ask the user to choose or create an embedding profile, rather than asking for an application-specific `--openai-api-key`.

### Loading Settings from Parsed Values

For CLI applications using Glazed's parameter system:

```go
package main

import (
    "fmt"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/go-go-golems/glazed/pkg/cmds/values"
)

func createProviderFromValues(parsedValues *values.Values) (embeddings.Provider, error) {
    // Create a factory from parsed values
    factory, err := embeddings.NewSettingsFactoryFromParsedValues(parsedValues)
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

If you already have a populated `settings.StepSettings` object:

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

## Using `!Embeddings` in Emrichen Templates

[Emrichen](https://github.com/con2/emrichen) is a YAML templating language. Geppetto adds an `!Embeddings` tag for generating embeddings inline.

### Basic Usage

```yaml
# Uses default provider from EmbeddingsConfig
embedding: !Embeddings
  text: "Text to embed"
```

### With Configuration Overrides

```yaml
embedding: !Embeddings
  text: "Text to embed"
  config:
    type: openai
    engine: text-embedding-3-small
    dimensions: 1536
```

### Ollama Example

```yaml
embedding: !Embeddings
  text: "Local embeddings"
  config:
    type: ollama
    engine: all-minilm
    dimensions: 384
    base_url: "http://localhost:11434"
```

### OpenAI-Compatible Endpoint from a Profile

For OpenAI-compatible endpoints, put the base URL and key in a profile registry layer and resolve that profile before creating the embeddings factory. Keep provider credentials out of templates.

```yaml
profiles:
  openai-compatible-embedding:
    inference_settings:
      api:
        api_keys:
          openai-api-key: <configured in ~/.config/pinocchio/profiles.yaml>
        base_urls:
          openai-base-url: https://api.mycompany.com/v1
      embeddings:
        type: openai
        engine: text-embedding-3-small
        dimensions: 1536
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

### Creating Sections

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
    "github.com/go-go-golems/glazed/pkg/cmds/fields"
    "github.com/go-go-golems/glazed/pkg/middlewares"
    "github.com/go-go-golems/glazed/pkg/cmds/schema"
    "github.com/go-go-golems/glazed/pkg/cmds/values"
    "github.com/spf13/cobra"
)

// GetEmbeddingsLayers returns sections for embeddings
func GetEmbeddingsLayers() ([]schema.Section, error) {
    // Create embeddings parameter layer
    embeddingsLayer, err := config.NewEmbeddingsValueSection()
    if err != nil {
        return nil, err
    }
    
    // Create API key parameter layer
    embeddingsApiKey, err := config.NewEmbeddingsApiKeyParameter()
    if err != nil {
        return nil, err
    }
    
    return []schema.Section{
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
            cmds.WithSections(parametersLayers),
            cmds.WithArguments(
                fields.New(
                    "text",
                    fields.TypeString,
                    fields.WithHelp("Text to generate embeddings for"),
                    fields.WithRequired(true),
                ),
            ),
        ),
    }
    
    return command, nil
}

// RunIntoGlazeProcessor runs the embeddings command
func (c *EmbeddingsCommand) RunIntoGlazeProcessor(
    ctx context.Context,
    parsedValues *values.Values,
    gp middlewares.GlazeProcessor,
) error {
    // Decode arguments from the default section
    args := struct {
        Text string `glazed:"text"`
    }{}
    if err := parsedValues.DecodeSectionInto(values.DefaultSlug, &args); err != nil {
        return err
    }
    text := args.Text
    
    // Create factory from parsed values
    factory, err := embeddings.NewSettingsFactoryFromParsedValues(parsedValues)
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
    "github.com/go-go-golems/glazed/pkg/cmds/values"
    "github.com/go-go-golems/glazed/pkg/middlewares"
    "github.com/go-go-golems/glazed/pkg/cmds/fields"
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
            cmds.WithSections(parametersLayers),
            cmds.WithArguments(
                fields.New(
                    "text1",
                    fields.TypeString,
                    fields.WithHelp("First text to compare"),
                    fields.WithRequired(true),
                ),
                fields.New(
                    "text2",
                    fields.TypeString,
                    fields.WithHelp("Second text to compare"),
                    fields.WithRequired(true),
                ),
            ),
        ),
    }
}

func (c *CompareCommand) RunIntoGlazeProcessor(
    ctx context.Context,
    parsedValues *values.Values,
    gp middlewares.GlazeProcessor,
) error {
    // Decode arguments from the default section
    args := struct {
        Text1 string `glazed:"text1"`
        Text2 string `glazed:"text2"`
    }{}
    if err := parsedValues.DecodeSectionInto(values.DefaultSlug, &args); err != nil {
        return err
    }
    text1 := args.Text1
    text2 := args.Text2
    
    // Create factory from parsed values
    factory, err := embeddings.NewSettingsFactoryFromParsedValues(parsedValues)
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
    
    // Initialize Glazed logging flags on the root command.
    // Configure env/config loading explicitly through CobraParserConfig
    // or direct source middlewares when building commands.
    err := clay.InitGlazed("text-compare", rootCmd)
    if err != nil {
        fmt.Printf("Failed to initialize Glazed: %v\n", err)
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

## Best Practices

When working with the embeddings package, keep these best practices in mind:

| Practice | Why |
|----------|-----|
| **Cache aggressively** | Embedding API calls are expensive |
| **Use batch endpoints** | More efficient than individual calls |
| **Store embeddings** | Don't regenerate for static content |
| **Match dimensions** | All vectors in a collection must have same dimensions |
| **Normalize for comparison** | Use cosine similarity, not Euclidean distance |
| **Use sections** | Allow flexible configuration via CLI flags |
| **Consider vector databases** | For large-scale applications, use specialized storage |

## Common Dimensions

| Provider | Model | Dimensions |
|----------|-------|------------|
| OpenAI | text-embedding-3-small | 1536 |
| OpenAI | text-embedding-3-large | 3072 |
| OpenAI | text-embedding-ada-002 | 1536 |
| Ollama | all-minilm | 384 |
| Ollama | nomic-embed-text | 768 |
| Ollama | mxbai-embed-large | 1024 |

## Packages

```go
import (
    "github.com/go-go-golems/geppetto/pkg/embeddings"        // Core providers, cache
    "github.com/go-go-golems/geppetto/pkg/embeddings/config" // Settings, sections
)
```

## See Also

- [Inference Engines](06-inference-engines.md) — Using embeddings with inference
- Example: `geppetto/cmd/examples/` (check for embedding examples)

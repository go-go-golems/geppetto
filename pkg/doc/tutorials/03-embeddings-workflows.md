---
Title: Embeddings Workflows
Slug: geppetto-tutorial-embeddings-workflows
Short: Build applications using embeddings for semantic search, with single/batch generation and caching strategies.
Topics:
- geppetto
- tutorial
- embeddings
- semantic-search
- caching
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Embeddings Workflows

This tutorial teaches you how to use Geppetto's embeddings package to build semantic search and similarity applications. You'll learn single and batch embedding generation, caching strategies, and how to compute similarity between texts.

## What You'll Build

A semantic document search application that:
- Generates embeddings for a document collection
- Caches embeddings to avoid redundant API calls
- Finds similar documents based on query similarity
- Uses batch processing for efficiency

## Prerequisites

- Basic Go knowledge
- API key for OpenAI or a running Ollama instance
- Understanding of what embeddings are (see [Embeddings](../topics/06-embeddings.md))

## Learning Objectives

- Create and configure embedding providers
- Choose between memory and file caching
- Generate single and batch embeddings
- Compute cosine similarity for search
- Use the Emrichen `!Embeddings` tag in templates

## Step 1: Create an Embedding Provider

Choose between OpenAI (cloud) or Ollama (local):

### OpenAI Provider

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
    // Create OpenAI provider
    provider := embeddings.NewOpenAIProvider(
        os.Getenv("OPENAI_API_KEY"),  // API key
        openai.SmallEmbedding3,        // Model: text-embedding-3-small
        1536,                          // Dimensions
    )

    ctx := context.Background()
    vector, err := provider.GenerateEmbedding(ctx, "Hello, world!")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Generated %d-dimensional vector\n", len(vector))
    fmt.Printf("First 5 values: %v\n", vector[:5])
}
```

### Ollama Provider (Local)

```go
    // Create Ollama provider (runs locally)
    provider := embeddings.NewOllamaProvider(
        "http://localhost:11434",  // Ollama server URL
        "all-minilm",              // Model name
        384,                       // Dimensions (model-specific)
    )
```

**Common models and dimensions:**

| Provider | Model | Dimensions |
|----------|-------|------------|
| OpenAI | text-embedding-3-small | 1536 |
| OpenAI | text-embedding-3-large | 3072 |
| Ollama | all-minilm | 384 |
| Ollama | nomic-embed-text | 768 |

## Step 2: Add In-Memory Caching

Wrap your provider with a cache to avoid redundant API calls:

```go
    // Wrap with memory cache (stores up to 1000 embeddings)
    cachedProvider := embeddings.NewCachedProvider(provider, 1000)

    // First call - hits the API
    vec1, _ := cachedProvider.GenerateEmbedding(ctx, "Hello, world!")

    // Second call - served from cache (instant, free)
    vec2, _ := cachedProvider.GenerateEmbedding(ctx, "Hello, world!")

    // Check cache stats
    fmt.Printf("Cache size: %d/%d\n", cachedProvider.Size(), cachedProvider.MaxSize())
```

**When to use memory cache:**
- Short-lived scripts
- Tests
- Same texts embedded multiple times in one run

## Step 3: Add Persistent File Caching

For long-running applications or CLI tools that restart:

```go
    // Wrap with disk cache
    diskProvider, err := embeddings.NewDiskCacheProvider(
        provider,
        embeddings.WithDirectory("./cache/embeddings"),  // Custom directory
        embeddings.WithMaxSize(1<<30),                   // 1GB max
        embeddings.WithMaxEntries(10000),                // 10,000 entries max
    )
    if err != nil {
        panic(err)
    }

    // Embeddings persist across program restarts
    vec, _ := diskProvider.GenerateEmbedding(ctx, "Hello, world!")
```

**Default location:** `~/.geppetto/cache/embeddings/<model>/`

**When to use file cache:**
- CLI tools run repeatedly
- Static document collections
- Development/testing iterations

## Step 4: Batch Embedding Generation

For multiple texts, batch calls are more efficient:

```go
    texts := []string{
        "Go is a statically typed language.",
        "Python is dynamically typed.",
        "Rust focuses on memory safety.",
        "JavaScript runs in browsers.",
    }

    // Generate all embeddings in one API call
    vectors, err := provider.GenerateBatchEmbeddings(ctx, texts)
    if err != nil {
        panic(err)
    }

    for i, vec := range vectors {
        fmt.Printf("%s → %d dimensions\n", texts[i][:20], len(vec))
    }
```

**If your provider doesn't support batch natively:**

```go
    // Parallel generation with concurrency limit
    vectors, err := embeddings.ParallelGenerateBatchEmbeddings(
        ctx, 
        provider, 
        texts, 
        4,  // Max 4 concurrent requests
    )
```

## Step 5: Compute Cosine Similarity

Find how similar two texts are:

```go
import "math"

// Cosine similarity: 1.0 = identical, 0.0 = orthogonal, -1.0 = opposite
func cosineSimilarity(a, b []float32) float64 {
    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func main() {
    ctx := context.Background()
    provider := createProvider()

    // Compare two texts
    vec1, _ := provider.GenerateEmbedding(ctx, "Golang is a programming language.")
    vec2, _ := provider.GenerateEmbedding(ctx, "Go is a compiled language.")
    vec3, _ := provider.GenerateEmbedding(ctx, "The weather is nice today.")

    fmt.Printf("Go vs Go: %.4f\n", cosineSimilarity(vec1, vec2))   // High similarity
    fmt.Printf("Go vs Weather: %.4f\n", cosineSimilarity(vec1, vec3)) // Low similarity
}
```

**Output:**
```
Go vs Go: 0.9234
Go vs Weather: 0.1876
```

## Step 6: Build a Semantic Search System

Put it all together:

```go
package main

import (
    "context"
    "fmt"
    "math"
    "sort"

    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

type Document struct {
    ID        string
    Text      string
    Embedding []float32
}

type SearchResult struct {
    Document   Document
    Similarity float64
}

func main() {
    ctx := context.Background()

    // Create cached provider
    provider := embeddings.NewOpenAIProvider(apiKey, model, 1536)
    cachedProvider := embeddings.NewCachedProvider(provider, 1000)

    // Document collection
    documents := []Document{
        {ID: "1", Text: "Go is a statically typed, compiled language designed at Google."},
        {ID: "2", Text: "Python is an interpreted, high-level programming language."},
        {ID: "3", Text: "Rust is a systems programming language focused on safety."},
        {ID: "4", Text: "JavaScript is the language of the web browser."},
        {ID: "5", Text: "TypeScript adds static typing to JavaScript."},
    }

    // Pre-compute embeddings for all documents
    fmt.Println("Indexing documents...")
    for i := range documents {
        embedding, err := cachedProvider.GenerateEmbedding(ctx, documents[i].Text)
        if err != nil {
            panic(err)
        }
        documents[i].Embedding = embedding
    }

    // Search query
    query := "Which languages have static typing?"
    fmt.Printf("\nQuery: %s\n\n", query)

    queryEmbedding, _ := cachedProvider.GenerateEmbedding(ctx, query)

    // Compute similarities
    var results []SearchResult
    for _, doc := range documents {
        sim := cosineSimilarity(queryEmbedding, doc.Embedding)
        results = append(results, SearchResult{Document: doc, Similarity: sim})
    }

    // Sort by similarity (descending)
    sort.Slice(results, func(i, j int) bool {
        return results[i].Similarity > results[j].Similarity
    })

    // Print results
    fmt.Println("Results:")
    for i, r := range results {
        fmt.Printf("%d. [%.4f] %s\n", i+1, r.Similarity, r.Document.Text)
    }
}

func cosineSimilarity(a, b []float32) float64 {
    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

**Output:**
```
Indexing documents...

Query: Which languages have static typing?

Results:
1. [0.8912] Go is a statically typed, compiled language designed at Google.
2. [0.8654] TypeScript adds static typing to JavaScript.
3. [0.7823] Rust is a systems programming language focused on safety.
4. [0.6234] Python is an interpreted, high-level programming language.
5. [0.5987] JavaScript is the language of the web browser.
```

## Step 7: Use Configuration-Based Setup

For CLI applications, use the settings factory:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/embeddings"
    "github.com/go-go-golems/geppetto/pkg/embeddings/config"
)

func createProviderFromConfig() (embeddings.Provider, error) {
    cfg := &config.EmbeddingsConfig{
        Type:            "openai",
        Engine:          "text-embedding-3-small",
        Dimensions:      1536,
        CacheType:       "file",  // "none", "memory", or "file"
        CacheMaxEntries: 10000,
        APIKeys: map[string]string{
            "openai-api-key": os.Getenv("OPENAI_API_KEY"),
        },
    }

    factory := embeddings.NewSettingsFactory(cfg)
    return factory.NewProvider()
}
```

## Step 8: Use Emrichen Templates

Generate embeddings inline in YAML templates:

```yaml
# template.yaml
documents:
  - text: "Introduction to Go"
    embedding: !Embeddings
      text: "Introduction to Go"
      config:
        type: openai
        engine: text-embedding-3-small

  - text: "Python Tutorial"
    embedding: !Embeddings
      text: "Python Tutorial"
```

Process with Emrichen:
```go
import "github.com/go-go-golems/geppetto/pkg/js"

// Register the !Embeddings tag
emrichen.RegisterTag("Embeddings", js.GetEmbeddingTagFunc(embeddingsConfig))

// Process template
result, err := emrichen.ProcessFile("template.yaml")
```

## Cache Decision Guide

| Scenario | Cache Type | Why |
|----------|-----------|-----|
| One-off script | `none` | Not worth caching |
| Unit tests | `memory` | Fast, isolated |
| CLI tool (repeated runs) | `file` | Persists across runs |
| Long-running server | `memory` | In-process performance |
| Large static corpus | `file` | Don't re-embed on restart |
| Dynamic content | `none` or `memory` | Content changes frequently |

## Performance Tips

1. **Batch when possible** — 100 texts in one call is faster than 100 individual calls
2. **Cache aggressively** — Embedding API calls are expensive
3. **Pre-compute static content** — Index documents once, store embeddings
4. **Use smaller models for prototyping** — `all-minilm` (384d) is faster than `text-embedding-3-large` (3072d)
5. **Consider vector databases** — For large collections, use Pinecone, Weaviate, or pgvector

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Dimension mismatch | Comparing vectors from different models | Use same model for all vectors |
| API rate limits | Too many concurrent requests | Use `ParallelGenerateBatchEmbeddings` with low concurrency |
| Cache not working | Wrong cache type | Check `CacheType` in config |
| Slow first run | Cache empty | Expected; subsequent runs use cache |
| Ollama connection refused | Server not running | Start with `ollama serve` |

## See Also

- [Embeddings](../topics/06-embeddings.md) — Full embeddings reference
- Example: `geppetto/cmd/examples/` (check for embedding examples)


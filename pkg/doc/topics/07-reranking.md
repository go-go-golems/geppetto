---
Title: Using the Reranking Package
Slug: geppetto-reranking-package
Short: A guide to the reranking package in Geppetto, for search-related applications.
Topics:
- geppetto
- reranking
- search
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

# Using the Reranking Package

This tutorial provides a guide to using the reranking functionality in Geppetto. Reranking is a powerful technique for improving search results by scoring and reordering documents based on their relevance to a specific query.

## Core Concepts

Reranking works differently from embedding-based search:

- **Embeddings**: Convert text to vectors and compare similarity mathematically
- **Reranking**: Directly compares a query against a list of documents using contextual understanding

The reranking package provides a simple interface:

```go
type Reranker interface {
    // Rerank reorders the provided documents based on their relevance to the query
    Rerank(ctx context.Context, query string, documents []string, options ...RerankOption) ([]RankResult, error)
    
    // GetModel returns information about the reranking model being used
    GetModel() RerankerModel
}
```

Each result contains:

```go
type RankResult struct {
    // Index is the position of the document in the original input list
    Index int
    
    // Document is the original document text
    Document string
    
    // Score is the relevance score between 0 and 1 (higher = more relevant)
    Score float64
}
```

## Using Cohere for Reranking

Geppetto implements reranking using Cohere's Rerank API, which is specifically designed for this purpose:

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

func main() {
    // Create a Cohere reranker
    apiKey := os.Getenv("COHERE_API_KEY")
    reranker := embeddings.NewCohereReranker(
        apiKey,           // Cohere API key
        "rerank-v3.5",    // Model to use
    )
    
    // Define a query and documents to rank
    query := "What is the capital of the United States?"
    documents := []string{
        "Carson City is the capital city of the American state of Nevada.",
        "The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean. Its capital is Saipan.",
        "Capitalization in English grammar is the use of a capital letter at the start of a word.",
        "Washington, D.C. is the capital of the United States.",
        "Capital punishment has existed in the United States since before the United States was a country.",
    }
    
    // Rerank the documents
    ctx := context.Background()
    results, err := reranker.Rerank(ctx, query, documents, 
        embeddings.WithTopN(3), // Optional: limit to top 3 results
    )
    if err != nil {
        fmt.Printf("Error reranking documents: %v\n", err)
        return
    }
    
    // Print the ranked results
    fmt.Println("Reranked documents by relevance:")
    for i, result := range results {
        fmt.Printf("%d. (Score: %.4f) %s\n", i+1, result.Score, result.Document)
    }
}
```

## Configuration Options

The reranker supports several configuration options:

```go
// Limit the number of results returned
results, err := reranker.Rerank(ctx, query, documents, 
    embeddings.WithTopN(5),
)

// Limit the number of tokens per document to process
results, err := reranker.Rerank(ctx, query, documents, 
    embeddings.WithMaxTokensPerDoc(1000),
)

// Combine multiple options
results, err := reranker.Rerank(ctx, query, documents,
    embeddings.WithTopN(5),
    embeddings.WithMaxTokensPerDoc(1000),
)
```

## Practical Applications

### Building a Better Search Engine

Combining embeddings with reranking creates a powerful search solution:

```go
package main

import (
    "context"
    "fmt"
    "sort"
    
    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

type Document struct {
    ID      string
    Title   string
    Content string
}

func main() {
    // Create providers
    embeddingProvider := createEmbeddingProvider() // Implementation omitted for brevity
    reranker := createReranker()                   // Implementation omitted for brevity
    
    // Sample document collection
    documents := []Document{
        {ID: "doc1", Title: "Nevada", Content: "Carson City is the capital of Nevada, a western US state."},
        {ID: "doc2", Title: "US Government", Content: "Washington, D.C. is the capital of the United States."},
        {ID: "doc3", Title: "Grammar Rules", Content: "Capitalization is the writing of a word with its first letter in uppercase."},
        {ID: "doc4", Title: "US History", Content: "The United States declared independence in 1776 and established its capital in Washington."},
        {ID: "doc5", Title: "Mariana Islands", Content: "The Northern Mariana Islands have Saipan as their capital."},
    }
    
    // Create embeddings for all documents
    ctx := context.Background()
    contents := make([]string, len(documents))
    for i, doc := range documents {
        contents[i] = doc.Content
    }
    
    // User query
    query := "What is the US capital city?"
    
    // Step 1: Generate query embedding
    queryEmbedding, _ := embeddingProvider.GenerateEmbedding(ctx, query)
    
    // Step 2: Generate embeddings for all documents
    docEmbeddings, _ := embeddingProvider.GenerateBatchEmbeddings(ctx, contents)
    
    // Step 3: Calculate similarity scores
    type ScoredDoc struct {
        Document Document
        Similarity float64
    }
    
    var scoredDocs []ScoredDoc
    for i, embedding := range docEmbeddings {
        similarity := computeCosineSimilarity(queryEmbedding, embedding) // Implementation omitted
        scoredDocs = append(scoredDocs, ScoredDoc{
            Document:   documents[i],
            Similarity: similarity,
        })
    }
    
    // Step 4: Sort by similarity (descending)
    sort.Slice(scoredDocs, func(i, j int) bool {
        return scoredDocs[i].Similarity > scoredDocs[j].Similarity
    })
    
    // Step 5: Take top candidates for reranking (10 in this example)
    candidateCount := 10
    if len(scoredDocs) < candidateCount {
        candidateCount = len(scoredDocs)
    }
    
    candidates := make([]string, candidateCount)
    for i := 0; i < candidateCount; i++ {
        candidates[i] = scoredDocs[i].Document.Content
    }
    
    // Step 6: Rerank the top candidates
    rerankedResults, _ := reranker.Rerank(ctx, query, candidates)
    
    // Print final results
    fmt.Println("Search results:")
    for i, result := range rerankedResults {
        originalIndex := scoredDocs[result.Index].Document.ID
        fmt.Printf("%d. [%s] %s (Score: %.4f)\n", 
            i+1, 
            originalIndex,
            result.Document,
            result.Score,
        )
    }
}
```

## Best Practices

When working with reranking:

1. **Pre-filter with embeddings**: Use embeddings to quickly filter a large document set before reranking
2. **Document size**: Keep documents under 1,000 per batch for optimal performance
3. **Document length**: Consider truncating very long documents using `WithMaxTokensPerDoc`
4. **Result limiting**: Use `WithTopN` to limit results when you only need the most relevant items
5. **Hybrid approaches**: Combine keyword search, embeddings, and reranking for best results

## Performance Considerations

Reranking is more computationally intensive than embedding-based similarity:

- **Latency**: Reranking typically has higher latency than embedding comparison
- **Batch size**: Process fewer documents at once (hundreds rather than thousands)
- **Cost**: API costs may be higher for reranking operations

A common approach is to use cheaper methods (keyword search, embeddings) for initial filtering, then apply reranking to a smaller set of candidates.
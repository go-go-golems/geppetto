# JavaScript Embeddings API

## Overview

The Embeddings API provides JavaScript bindings for generating vector embeddings from text using various embedding models. It supports different providers (like OpenAI, Ollama) and offers synchronous, Promise-based, and callback-based APIs for text-to-vector conversion.

## Core Concepts

### What are Embeddings?

Embeddings are vector representations of text that capture semantic meaning in a high-dimensional space. They're useful for:
- Semantic search
- Text similarity comparison
- Document clustering
- Information retrieval
- Machine learning features

### Model Information

Each embeddings provider exposes information about its model:
- Name: Identifier of the embedding model
- Dimensions: Number of dimensions in the output vectors

## API Reference

The Embeddings API exposes the following functions:

### Synchronous API

#### generateEmbedding

Converts text into a vector embedding synchronously.

```javascript
const text = "Hello, world!";
const embedding = embeddings.generateEmbedding(text);
// Returns: Float32Array of dimensions length
```

#### getModel

Returns information about the current embedding model.

```javascript
const model = embeddings.getModel();
// Returns: { name: string, dimensions: number }
```

### Asynchronous API

#### generateEmbeddingAsync

Promise-based API for generating embeddings.

```javascript
const text = "Hello, world!";
try {
    const embedding = await embeddings.generateEmbeddingAsync(text);
    console.log("Embedding dimensions:", embedding.length);
} catch (err) {
    console.error("Failed to generate embedding:", err);
}
```

#### generateEmbeddingWithCallbacks

Callback-based API for generating embeddings with cancellation support.

```javascript
const text = "Hello, world!";
const cancel = embeddings.generateEmbeddingWithCallbacks(text, {
    onSuccess: (embedding) => {
        console.log("Embedding generated:", embedding);
    },
    onError: (err) => {
        console.error("Error:", err);
    }
});

// Cancel the operation if needed
cancel();
```

## Usage Examples

### Basic Text Embedding

```javascript
// Synchronous API
const text = "The quick brown fox jumps over the lazy dog";
try {
    const embedding = embeddings.generateEmbedding(text);
    console.log("Embedding dimensions:", embedding.length);
} catch (err) {
    console.error("Failed to generate embedding:", err);
}

// Async/Promise API
async function generateEmbedding(text) {
    try {
        const embedding = await embeddings.generateEmbeddingAsync(text);
        return embedding;
    } catch (err) {
        console.error("Failed to generate embedding:", err);
        throw err;
    }
}

// Callback API with error handling
const cancel = embeddings.generateEmbeddingWithCallbacks(text, {
    onSuccess: (embedding) => {
        console.log("Success! Dimensions:", embedding.length);
    },
    onError: (err) => {
        console.error("Error generating embedding:", err);
    }
});
```

### Model Information

```javascript
// Get model details
const model = embeddings.getModel();
console.log("Using model:", model.name);
console.log("Vector dimensions:", model.dimensions);
```

### Batch Processing

```javascript
// Process multiple texts with Promise.all
const texts = [
    "First document",
    "Second document",
    "Third document"
];

async function batchProcess(texts) {
    try {
        const embeddings = await Promise.all(
            texts.map(text => embeddings.generateEmbeddingAsync(text))
        );
        return embeddings;
    } catch (err) {
        console.error("Batch processing failed:", err);
        throw err;
    }
}
```

### Semantic Search Example

```javascript
// Function to compute cosine similarity between vectors
function cosineSimilarity(a, b) {
    let dotProduct = 0;
    let normA = 0;
    let normB = 0;
    
    for (let i = 0; i < a.length; i++) {
        dotProduct += a[i] * b[i];
        normA += a[i] * a[i];
        normB += b[i] * b[i];
    }
    
    return dotProduct / (Math.sqrt(normA) * Math.sqrt(normB));
}

// Async semantic search implementation
async function semanticSearch(query, documents) {
    try {
        // Generate query embedding
        const queryEmbedding = await embeddings.generateEmbeddingAsync(query);
        
        // Generate document embeddings
        const documentEmbeddings = await Promise.all(
            documents.map(doc => embeddings.generateEmbeddingAsync(doc))
        );
        
        // Calculate similarities
        const similarities = documentEmbeddings.map(docEmb => 
            cosineSimilarity(queryEmbedding, docEmb)
        );
        
        // Find best match
        const bestMatchIndex = similarities.indexOf(Math.max(...similarities));
        return {
            document: documents[bestMatchIndex],
            similarity: similarities[bestMatchIndex]
        };
    } catch (err) {
        console.error("Semantic search failed:", err);
        throw err;
    }
}
```

## Best Practices

### 1. Error Handling

Always wrap embedding generation in try-catch blocks and handle errors appropriately:

```javascript
try {
    const embedding = await embeddings.generateEmbeddingAsync(text);
    // Process embedding...
} catch (err) {
    console.error("Failed to generate embedding:", err);
    // Handle error appropriately
}
```

### 2. Cancellation

Use the callback API when you need cancellation support:

```javascript
let cancel;

function startEmbedding() {
    cancel = embeddings.generateEmbeddingWithCallbacks(text, {
        onSuccess: handleSuccess,
        onError: handleError
    });
}

function stopEmbedding() {
    if (cancel) {
        cancel();
        cancel = null;
    }
}
```

### 3. Resource Management

Consider embedding size and memory usage:

```javascript
const model = embeddings.getModel();
const memoryPerEmbedding = model.dimensions * 4; // 4 bytes per float32

// Calculate memory for batch processing
const batchSize = 1000;
const estimatedMemory = memoryPerEmbedding * batchSize;
console.log(`Estimated memory usage: ${estimatedMemory / 1024 / 1024} MB`);
```

### 4. Choosing the Right API

- Use `generateEmbedding` for simple, synchronous operations
- Use `generateEmbeddingAsync` for Promise-based async operations and better error handling
- Use `generateEmbeddingWithCallbacks` when you need cancellation support or progress updates

### 5. Performance Optimization

- Batch similar requests using Promise.all
- Cache frequently used embeddings
- Consider using Web Workers for heavy computation
- Use cancellation to prevent unnecessary work

## Error Types

Common errors to handle:

1. Invalid input text
2. Provider API errors
3. Rate limiting
4. Network issues
5. Model loading errors
6. Cancellation errors

The Embeddings API provides a flexible interface for text-to-vector conversion with multiple paradigms to suit different use cases. Choose the appropriate API based on your needs for synchronous vs asynchronous operation and error handling requirements. 
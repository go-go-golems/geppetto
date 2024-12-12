# JavaScript Embeddings API

## Overview

The Embeddings API provides JavaScript bindings for generating vector embeddings from text using various embedding models. It supports different providers (like OpenAI, Ollama) and offers a simple interface for text-to-vector conversion.

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

The Embeddings API exposes two main functions:

### generateEmbedding

Converts text into a vector embedding.

```javascript
const text = "Hello, world!";
const embedding = embeddings.generateEmbedding(text);
// Returns: Float32Array of dimensions length
```

### getModel

Returns information about the current embedding model.

```javascript
const model = embeddings.getModel();
// Returns: { name: string, dimensions: number }
```

## Usage Examples

### Basic Text Embedding

```javascript
// Generate embedding for a single text
const text = "The quick brown fox jumps over the lazy dog";
const embedding = embeddings.generateEmbedding(text);
console.log("Embedding dimensions:", embedding.length);
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
// Process multiple texts
const texts = [
    "First document",
    "Second document",
    "Third document"
];

const embeddings = texts.map(text => embeddings.generateEmbedding(text));
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

// Create embeddings for a document collection
const documents = [
    "The weather is sunny today",
    "Machine learning is fascinating",
    "I love programming in JavaScript"
];

const documentEmbeddings = documents.map(doc => 
    embeddings.generateEmbedding(doc)
);

// Search for similar documents
const query = "What's the weather like?";
const queryEmbedding = embeddings.generateEmbedding(query);

// Find most similar document
const similarities = documentEmbeddings.map(docEmb => 
    cosineSimilarity(queryEmbedding, docEmb)
);

const mostSimilarIndex = similarities.indexOf(Math.max(...similarities));
console.log("Most similar document:", documents[mostSimilarIndex]);
```

## Best Practices

### 1. Error Handling

Always wrap embedding generation in try-catch blocks:

```javascript
try {
    const embedding = embeddings.generateEmbedding(text);
    // Process embedding...
} catch (err) {
    console.error("Failed to generate embedding:", err);
    // Handle error appropriately
}
```

### 2. Input Preprocessing

Clean and normalize text before generating embeddings:

```javascript
function preprocessText(text) {
    return text
        .toLowerCase()
        .trim()
        .replace(/\s+/g, ' ');
}

const text = "  Multiple    spaces   and CAPS  ";
const processed = preprocessText(text);
const embedding = embeddings.generateEmbedding(processed);
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

### 4. Caching

Cache embeddings for frequently used texts:

```javascript
const embeddingCache = new Map();

function getCachedEmbedding(text) {
    const key = preprocessText(text);
    if (!embeddingCache.has(key)) {
        embeddingCache.set(key, embeddings.generateEmbedding(key));
    }
    return embeddingCache.get(key);
}
```

## Integration Examples

### With Document Processing

```javascript
class Document {
    constructor(text, metadata = {}) {
        this.text = text;
        this.metadata = metadata;
        this.embedding = null;
    }

    generateEmbedding() {
        this.embedding = embeddings.generateEmbedding(this.text);
        return this.embedding;
    }
}

// Usage
const doc = new Document("Sample text", { source: "user-input" });
doc.generateEmbedding();
```

### With Vector Database

```javascript
class VectorStore {
    constructor() {
        this.documents = [];
        this.embeddings = [];
    }

    addDocument(text, metadata = {}) {
        const embedding = embeddings.generateEmbedding(text);
        this.documents.push({ text, metadata });
        this.embeddings.push(embedding);
    }

    search(query, topK = 5) {
        const queryEmbedding = embeddings.generateEmbedding(query);
        const similarities = this.embeddings.map(emb => 
            cosineSimilarity(queryEmbedding, emb)
        );

        // Get top K results
        return similarities
            .map((score, idx) => ({ score, idx }))
            .sort((a, b) => b.score - a.score)
            .slice(0, topK)
            .map(({ score, idx }) => ({
                document: this.documents[idx],
                similarity: score
            }));
    }
}

// Usage
const store = new VectorStore();
store.addDocument("First document");
store.addDocument("Second document");

const results = store.search("document");
console.log("Search results:", results);
```

## Technical Details

### Provider Configuration

The embeddings provider is configured during initialization and cannot be changed at runtime. The available providers include:

- OpenAI
- Ollama
- Custom providers (if implemented)

### Performance Considerations

1. **Batch Size**: Consider batching multiple texts when possible
2. **Vector Dimensions**: Be aware of the model's dimension size
3. **Memory Usage**: Monitor memory when processing large datasets
4. **Caching**: Implement caching for repeated texts

### Error Types

Common errors to handle:

1. Invalid input text
2. Provider API errors
3. Rate limiting
4. Network issues
5. Model loading errors

## Advanced Topics

### Custom Distance Metrics

Beyond cosine similarity:

```javascript
class EmbeddingMetrics {
    static euclideanDistance(a, b) {
        return Math.sqrt(
            a.reduce((sum, val, i) => sum + Math.pow(val - b[i], 2), 0)
        );
    }

    static manhattanDistance(a, b) {
        return a.reduce((sum, val, i) => sum + Math.abs(val - b[i]), 0);
    }
}
```

### Dimensionality Reduction

For visualization or optimization:

```javascript
function reduceDimensions(embedding, targetDim = 2) {
    // Simple dimension reduction (average pooling)
    const result = new Array(targetDim).fill(0);
    const stride = Math.floor(embedding.length / targetDim);
    
    for (let i = 0; i < targetDim; i++) {
        const start = i * stride;
        const end = start + stride;
        const slice = embedding.slice(start, end);
        result[i] = slice.reduce((a, b) => a + b) / slice.length;
    }
    
    return result;
}
```

The Embeddings API provides a powerful interface for text-to-vector conversion, enabling various natural language processing and machine learning applications. By following these patterns and best practices, you can effectively integrate embeddings into your JavaScript applications. 
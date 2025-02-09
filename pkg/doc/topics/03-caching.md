---
Title: Caching in Geppetto
Slug: caching
Short: Learn how to use and configure caching for both embeddings and chat operations in Geppetto
Topics:
- caching
- embeddings
- chat
- configuration
Commands:
- none
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Geppetto provides a unified caching system for both embeddings and chat operations. This helps reduce API calls, improve performance, and manage costs effectively.

## Basic Usage

### Command-Line Flags

For both embeddings and chat operations, caching can be controlled through command-line flags:

#### Embeddings Caching

```bash
# Enable memory-based caching for embeddings
--embeddings-cache-type=memory --embeddings-cache-max-entries=1000

# Enable disk-based caching for embeddings
--embeddings-cache-type=disk --embeddings-cache-max-size=1073741824 --embeddings-cache-directory=/path/to/cache
```

#### Chat Caching

```bash
# Enable memory-based caching for chat
--ai-cache-type=memory --ai-cache-max-entries=1000

# Enable disk-based caching for chat
--ai-cache-type=disk --ai-cache-max-size=1073741824 --ai-cache-directory=/path/to/cache
```

### Configuration Parameters

Both embeddings and chat share similar caching parameters:

1. **Cache Type** (`embeddings-cache-type` / `ai-cache-type`)
   - `none`: Disable caching (default)
   - `memory`: In-memory LRU cache
   - `disk`: Persistent disk-based cache

2. **Cache Size** (`embeddings-cache-max-size` / `ai-cache-max-size`)
   - Maximum size in bytes for disk cache
   - Default: 1GB (1073741824 bytes)

3. **Cache Entries** (`embeddings-cache-max-entries` / `ai-cache-max-entries`)
   - Maximum number of entries for memory cache
   - Default: 1000 entries

4. **Cache Directory** (`embeddings-cache-directory` / `ai-cache-directory`)
   - Directory for disk cache storage
   - Default: "" (system temp directory)

## Programmatic Usage

### Embeddings Caching

You can configure caching programmatically when creating embedding providers:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/embeddings"
)

// Create config with caching settings
config := &embeddings.EmbeddingsConfig{
    Type:            "openai",
    Engine:          "text-embedding-3-small",
    CacheType:       "memory",
    CacheMaxEntries: 1000,
}

// Create factory with config
factory := embeddings.NewSettingsFactory(config)

// Get provider with caching enabled
provider, err := factory.NewProvider()
if err != nil {
    // Handle error
}

// Use provider as normal
embedding, err := provider.GenerateEmbedding(ctx, "text to embed")
```

### Chat Caching

For chat operations, you can wrap chat steps with caching:

```go
import (
    "github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
    "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
)

// Create chat settings with caching
chatSettings := &settings.ChatSettings{
    CacheType:       "memory",
    CacheMaxEntries: 1000,
}

// Create base chat step
baseStep := chat.NewChatStep(/* ... */)

// Wrap with caching
cachedStep, err := chatSettings.WrapWithCache(baseStep)
if err != nil {
    // Handle error
}

// Use cached step as normal
result, err := cachedStep.Start(ctx, conversation)
```

### Using with Glazed Commands

When building Glazed commands that use caching, you can access the caching settings through the parsed layers:

```go
type CommandSettings struct {
    Embeddings *embeddings.EmbeddingsConfig `glazed.layer:"embeddings"`
    Chat      *settings.ChatSettings       `glazed.layer:"ai-chat"`
}

func (c *MyCommand) Run(
    ctx context.Context,
    parsedLayers *layers.ParsedLayers,
    gp middlewares.Processor,
) error {
    s := &CommandSettings{}
    
    // Initialize settings from parsed layers
    if err := parsedLayers.InitializeStruct("embeddings", &s.Embeddings); err != nil {
        return err
    }
    if err := parsedLayers.InitializeStruct("ai-chat", &s.Chat); err != nil {
        return err
    }

    // Create providers/steps with caching
    provider, err := embeddings.NewSettingsFactory(s.Embeddings).NewProvider()
    if err != nil {
        return err
    }

    baseStep := chat.NewChatStep(/* ... */)
    cachedStep, err := s.Chat.WrapWithCache(baseStep)
    if err != nil {
        return err
    }

    // Use provider and step as needed
    return nil
}
```

## Cache Behavior

### Memory Cache

- Implements LRU (Least Recently Used) eviction
- Thread-safe for concurrent access
- Cleared when process exits
- Best for high-throughput, short-lived processes

### Disk Cache

- Persistent across process restarts
- Uses file modification time for LRU eviction
- Thread-safe through file system locks
- Best for long-running processes or when persistence is needed

## Error Handling

The caching system handles several types of errors:

1. **Configuration Errors**
   - Invalid cache type
   - Missing required parameters
   - Invalid directory permissions

2. **Runtime Errors**
   - Cache full/eviction needed
   - Disk I/O errors
   - Corrupted cache entries

Example error handling:

```go
provider, err := factory.NewProvider()
if err != nil {
    switch {
    case strings.Contains(err.Error(), "unsupported cache type"):
        // Handle invalid configuration
    case strings.Contains(err.Error(), "permission denied"):
        // Handle permission issues
    default:
        // Handle other errors
    }
}
```

## Best Practices

1. **Cache Type Selection**
   - Use `memory` for short-lived processes or small datasets
   - Use `disk` for persistence or large datasets
   - Use `none` when real-time responses are critical

2. **Cache Sizing**
   - Set `max-entries` based on expected unique requests
   - Set `max-size` based on available disk space
   - Monitor cache hit rates to adjust settings

3. **Directory Management**
   - Use absolute paths for cache directories
   - Ensure proper permissions
   - Implement periodic cleanup for old cache files

4. **Error Handling**
   - Always check for errors when creating cached providers/steps
   - Implement fallbacks for cache failures
   - Log cache-related errors for monitoring 
---
Title: Using the Embeddings Tag Function
Slug: emrichen-embeddings
Short: Generate embeddings directly in emrichen templates using the !Embeddings tag function
Topics:
- embeddings
- emrichen
- templates
Commands:
- none
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `!Embeddings` tag function allows you to generate embeddings directly in emrichen templates. It supports both OpenAI and Ollama providers, with configurable dimensions and model selection.

## Basic Usage

The simplest way to use the embeddings tag is to provide the text you want to embed:

```yaml
embedding: !Embeddings
  text: "Text to embed"
```

This will use the default configuration (OpenAI with text-embedding-3-small model).

## Configuration Options

You can override the default settings by providing a config object:

```yaml
embedding: !Embeddings
  text: "Text to embed"
  config:
    type: openai  # or "ollama"
    engine: text-embedding-3-small
    dimensions: 1536  # Optional, defaults to 1536 for OpenAI
```

### Provider Types

Two embedding providers are supported:

1. **OpenAI** (`type: openai`)
   - Requires an API key
   - Default model: text-embedding-3-small
   - Default dimensions: 1536
   - Configurable base URL (defaults to https://api.openai.com/v1)

2. **Ollama** (`type: ollama`)
   - Local deployment
   - Default base URL: http://localhost:11434
   - Requires specifying dimensions (e.g. 384 for all-minilm)
   - No API key required by default

## Configuration Parameters

The following parameters can be configured:

- `type`: The provider type ("openai" or "ollama")
- `engine`: The model to use for embeddings
- `dimensions`: Output dimension of the embeddings
- API keys and base URLs (via environment or configuration)

### Default Settings

```yaml
embeddings-engine: text-embedding-3-small
embeddings-type: openai
embeddings-dimensions: 1536
openai-base-url: https://api.openai.com/v1
ollama-base-url: http://localhost:11434
```

## Environment Variables

For security, it's recommended to provide API keys via environment variables:

- `OPENAI_API_KEY`: For OpenAI embeddings
- `OLLAMA_API_KEY`: For Ollama embeddings (if required)

## Examples

### Using OpenAI

```yaml
document_embedding: !Embeddings
  text: "This is a document that needs to be embedded"
  config:
    type: openai
    engine: text-embedding-3-small
```

### Using Ollama

```yaml
local_embedding: !Embeddings
  text: "This is a document that needs to be embedded"
  config:
    type: ollama
    engine: all-minilm
    dimensions: 384
```

### Using Custom Base URLs

```yaml
custom_endpoint: !Embeddings
  text: "Text to embed"
  config:
    type: openai
    engine: text-embedding-3-small
    base_url: "https://api.mycompany.com/v1"
```

## Error Handling

The tag function will return an error if:

- No text is provided
- Required configuration is missing (e.g., API key for OpenAI)
- Invalid provider type is specified
- Invalid model is specified
- Dimensions are not specified for Ollama
- API calls fail

Make sure to handle these errors appropriately in your templates. 

# Cohere Embed API Documentation

## Overview
The Cohere Embed API endpoint generates text embeddings - lists of floating point numbers that capture semantic information about the input text or images. These embeddings are particularly useful for text classification, semantic search, and clustering applications.

## API Endpoint
- **URL**: `https://api.cohere.com/v2/embed`
- **Method**: POST
- **Content-Type**: application/json

## Headers
| Header Name | Type | Required | Description |
|------------|------|----------|-------------|
| X-Client-Name | string | Optional | The name of the project making the request |

## Request Parameters

### Required Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| model | string | ID of the Embedding model to use |
| input_type | enum | Type of input being passed to the model (Required for v3+ models) |

### Input Type Options
| Value | Description |
|-------|-------------|
| search_document | For embeddings stored in vector databases for search use-cases |
| search_query | For embeddings of search queries to find relevant documents |
| classification | For embeddings used in text classification |
| clustering | For embeddings used in clustering algorithms |
| image | For embeddings with image input |

### Optional Parameters
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| texts | list[string] | - | Array of strings to embed (max 96 texts per call) |
| images | list[string] | - | Array of image data URIs to embed (max 1 image) |
| inputs | list[object] | - | Array of mixed text/image inputs (max 96) |
| max_tokens | integer | - | Maximum tokens to embed per input |
| output_dimension | integer | 1536 | Dimensions of output embedding (256, 512, 1024, or 1536) |
| embedding_types | list[enum] | ["float"] | Types of embeddings to return |
| truncate | enum | "END" | How to handle inputs exceeding max token length |

### Embedding Types
| Type | Description | Model Support |
|------|-------------|---------------|
| float | Default float embeddings | All models |
| int8 | Signed 8-bit integer (-128 to 127) | v3.0+ |
| uint8 | Unsigned 8-bit integer (0 to 255) | v3.0+ |
| binary | Signed binary | v3.0+ |
| ubinary | Unsigned binary | v3.0+ |

### Truncation Options
| Value | Description |
|-------|-------------|
| NONE | Return error if input exceeds max length |
| START | Remove tokens from start until within limit |
| END | Remove tokens from end until within limit |

### Image Requirements
- Format: JPEG or PNG
- Maximum size: 5MB
- Must be provided as data URI
- Supported in Embed v3.0+ models

## Response Structure

### Success Response (200 OK)
```json
{
    "id": string,
    "embeddings": {
        "float": [[float]],
        "int8": [[integer]],
        "uint8": [[integer]],
        "binary": [[integer]],
        "ubinary": [[integer]]
    },
    "texts": [string],
    "images": [
        {
            "width": integer,
            "height": integer,
            "format": string,
            "bit_depth": integer
        }
    ],
    "meta": {
        "api_version": {
            "version": string,
            "is_experimental": boolean
        }
    }
}
```

### Response Fields
| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier for the request |
| embeddings | object | Contains embedding arrays for each requested type |
| embeddings.float | array | Float embeddings (if requested) |
| embeddings.int8 | array | Signed 8-bit integer embeddings (if requested) |
| embeddings.uint8 | array | Unsigned 8-bit integer embeddings (if requested) |
| embeddings.binary | array | Signed binary embeddings (if requested) |
| embeddings.ubinary | array | Unsigned binary embeddings (if requested) |
| texts | array | Input texts that were embedded |
| images | array | Metadata about embedded images |
| meta | object | API metadata |

## Error Codes
| Status Code | Error Type | Description |
|-------------|------------|-------------|
| 400 | Bad Request | Invalid request parameters |
| 401 | Unauthorized | Authentication failed |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 422 | Unprocessable Entity | Request could not be processed |
| 429 | Too Many Requests | Rate limit exceeded |
| 498 | Invalid Token | Invalid authentication token |
| 499 | Client Closed Request | Client closed connection |
| 500 | Internal Server Error | Server-side error |
| 501 | Not Implemented | Functionality not implemented |
| 503 | Service Unavailable | Service temporarily unavailable |
| 504 | Gateway Timeout | Request timed out |

## Example Usage

### Python Example - Text Embedding
```python
import cohere

co = cohere.ClientV2()

response = co.embed(
    texts=["hello", "goodbye"],
    model="embed-v4.0",
    input_type="classification",
    embedding_types=["float"]
)

print(response)
```

### Example Response
```json
{
    "id": "da6e531f-54c6-4a73-bf92-f60566d8d753",
    "embeddings": {
        "float": [
            [0.016296387, -0.008354187, ..., 0.0052719116],
        ]
    },
    "texts": ["hello", "goodbye"],
    "meta": {
        "api_version": {
            "version": "2",
            "is_experimental": false
        },
        "billed_units": {
            "input_tokens": 2
        }
    }
}
```

## Best Practices
1. Choose appropriate input_type for your use case
2. Use output_dimension to balance between embedding size and information retention
3. Consider using quantized embeddings (int8, uint8) for storage efficiency
4. Handle truncation appropriately for your application needs
5. Batch requests efficiently within the 96 texts per call limit
6. Consider image size and format requirements when embedding images

## Rate Limits and Quotas
Please refer to your specific API plan for rate limits and quotas. Contact Cohere support for detailed information about your account's limitations. 
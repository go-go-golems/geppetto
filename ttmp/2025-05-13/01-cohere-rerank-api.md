# Cohere Rerank API Documentation

## Overview
The Cohere Rerank API endpoint takes a query and a list of texts as input, and returns an ordered array where each text is assigned a relevance score. This makes it particularly useful for search applications, content recommendation systems, and any scenario where you need to sort documents by their relevance to a specific query.

## API Endpoint
- **URL**: `https://api.cohere.com/v2/rerank`
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
| model | string | The identifier of the model to use (e.g., `rerank-v3.5`) |
| query | string | The search query to compare documents against |
| documents | list[string] | List of texts to be compared with the query |

### Optional Parameters
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| top_n | integer | null | Limits the number of returned rerank results |
| max_tokens_per_doc | integer | 4096 | Maximum number of tokens per document before truncation |

### Important Notes
- For optimal performance, it's recommended to keep the document list under 1,000 items per request
- Long documents are automatically truncated to `max_tokens_per_doc` tokens
- Structured data should be formatted as YAML strings for best performance

## Response Structure

### Success Response (200 OK)
```json
{
    "results": [
        {
            "index": number,
            "relevance_score": float
        }
    ],
    "id": string,
    "meta": {
        "api_version": {
            "version": string,
            "is_experimental": boolean
        },
        "billed_units": {
            "search_units": number
        }
    }
}
```

### Response Fields
| Field | Type | Description |
|-------|------|-------------|
| results | array | Ordered list of ranked documents |
| results[].index | number | Index of the document in the original input list |
| results[].relevance_score | float | Relevance score (0-1) where 1 indicates highest relevance |
| id | string | Unique identifier for the request |
| meta | object | Metadata about the request |

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

### Python Example
```python
import cohere

co = cohere.ClientV2()

docs = [
    "Carson City is the capital city of the American state of Nevada.",
    "The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean. Its capital is Saipan.",
    "Capitalization or capitalisation in English grammar is the use of a capital letter at the start of a word.",
    "Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
    "Capital punishment has existed in the United States since beforethe United States was a country."
]

response = co.rerank(
    model="rerank-v3.5",
    query="What is the capital of the United States?",
    documents=docs,
    top_n=3
)

print(response)
```

### Example Response
```json
{
    "results": [
        {
            "index": 3,
            "relevance_score": 0.999071
        },
        {
            "index": 4,
            "relevance_score": 0.7867867
        },
        {
            "index": 0,
            "relevance_score": 0.32713068
        }
    ],
    "id": "07734bd2-2473-4f07-94e1-0d9f0e6843cf",
    "meta": {
        "api_version": {
            "version": "2",
            "is_experimental": false
        },
        "billed_units": {
            "search_units": 1
        }
    }
}
```

## Best Practices
1. Keep document count under 1,000 per request for optimal performance
2. Format structured data as YAML strings
3. Consider document length and token limits
4. Use appropriate `top_n` values to limit response size when needed
5. Handle rate limits and errors appropriately in production code

## Rate Limits and Quotas
Please refer to your specific API plan for rate limits and quotas. Contact Cohere support for detailed information about your account's limitations. 
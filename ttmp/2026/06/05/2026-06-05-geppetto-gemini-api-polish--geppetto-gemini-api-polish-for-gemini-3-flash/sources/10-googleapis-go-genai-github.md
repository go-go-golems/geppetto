---
Title: Google Gen AI Go SDK Repository
SourceURL: https://github.com/googleapis/go-genai
SourceTool: defuddle
FetchedAt: 2026-06-05T09:02:51-04:00
Ticket: 2026-06-05-geppetto-gemini-api-polish
Topics:
  - geppetto
  - providers
  - reasoning
  - streaming
  - tools
DocType: source
Status: active
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Official Gemini or SDK reference captured for the Geppetto Gemini API polish ticket.
WhatFor: Use as source material for Gemini 3 Flash API compatibility, thinking, thought signatures, function calling, and SDK migration.
WhenToUse: Read before changing Geppetto's Gemini provider implementation.
---

[![GitHub go.mod Go
version](https://camo.githubusercontent.com/a0f8c5743716ad56dd9612f5d0d33b121c95321783e2942215bac3e9
8af534f4/68747470733a2f2f696d672e736869656c64732e696f2f6769746875622f676f2d6d6f642f676f2d76657273696
f6e2f676f6f676c65617069732f676f2d67656e6169)](https://camo.githubusercontent.com/a0f8c5743716ad56dd9
612f5d0d33b121c95321783e2942215bac3e98af534f4/68747470733a2f2f696d672e736869656c64732e696f2f67697468
75622f676f2d6d6f642f676f2d76657273696f6e2f676f6f676c65617069732f676f2d67656e6169) [![Go
Reference](https://camo.githubusercontent.com/2653b1542e9646c1fc3bf6a320b699244f99a4a508ba91dcd3f9b3
43a6d9e7db/68747470733a2f2f706b672e676f2e6465762f62616467652f676f6f676c652e676f6c616e672e6f72672f676
56e61692e737667)](https://pkg.go.dev/google.golang.org/genai)

## Google Gen AI Go SDK

The Google Gen AI Go SDK provides an interface for developers to integrate Google's generative
models into their Go applications. It supports the [Gemini Developer
API](https://ai.google.dev/gemini-api/docs) and [Gemini Enterprise Agent
Platform](https://docs.cloud.google.com/gemini-enterprise-agent-platform) APIs.

The Google Gen AI Go SDK enables developers to use Google's state-of-the-art generative AI models
(like Gemini) to build AI-powered features and applications. This SDK supports use cases like:

- Generate text from text-only input
- Generate text from text-and-images input (multimodal)
- ...

For example, with just a few lines of code, you can access Gemini's multimodal capabilities to
generate text from text-and-image input.

```
parts := []*genai.Part{
  {Text: "What's this image about?"},
  {InlineData: &genai.Blob{Data: imageBytes, MIMEType: "image/jpeg"}},
}
result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", []*genai.Content{{Parts:
parts}}, nil)
```

## Installation and usage

Add the SDK to your module with `go get google.golang.org/genai`.

## Create Clients

### Imports

```
import "google.golang.org/genai"
```

### Gemini API Client:

```
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:   apiKey,
    Backend:  genai.BackendGeminiAPI,
})
```

### Gemini Enterprise Agent Platform Client:

```
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    Project:  project,
    Location: location,
    Backend:  genai.BackendEnterprise,
})
```

### (Optional) Using environment variables:

You can create a client by configuring the necessary environment variables. Configuration setup
instructions depends on whether you're using the Gemini Developer API or the Gemini API in Gemini
Enterprise Agent Platform.

**Gemini Developer API:** Set `GOOGLE_API_KEY` as shown below:

```
export GOOGLE_API_KEY='your-api-key'
```

**Gemini API on Gemini Enterprise Agent Platform:** Set `GOOGLE_GENAI_USE_ENTERPRISE`,
`GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION`, as shown below:

```
export GOOGLE_GENAI_USE_ENTERPRISE=true
export GOOGLE_CLOUD_PROJECT='your-project-id'
export GOOGLE_CLOUD_LOCATION='us-central1'
```
```
client, err := genai.NewClient(ctx, &genai.ClientConfig{})
```

## License

The contents of this repository are licensed under the [Apache License, version
2.0](http://www.apache.org/licenses/LICENSE-2.0).

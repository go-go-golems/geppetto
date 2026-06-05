---
Title: Gemini API Changelog
SourceURL: https://ai.google.dev/gemini-api/docs/changelog?hl=en
SourceTool: defuddle
FetchedAt: 2026-06-05T09:02:42-04:00
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

This page documents updates to the Gemini API.

## June 1, 2026

- The following Gemini 2.0 models are now [shut
down](https://ai.google.dev/gemini-api/docs/deprecations):
	- `gemini-2.0-flash`
		- `gemini-2.0-flash-001`
		- `gemini-2.0-flash-lite`
		- `gemini-2.0-flash-lite-001`
	Use [`gemini-3.5-flash`](https://ai.google.dev/gemini-api/docs/models/gemini-3.5-flash) or
[`gemini-3.1-flash-lite`](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite)
instead.

## May 28, 2026

- Released `gemini-3.1-flash-image` (Nano Banana 2) and `gemini-3-pro-image` (Nano Banana Pro), the
generally available (GA) versions of our native visual models, [Gemini 3.1 Flash
Image](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-image) and [Gemini 3 Pro
Image](https://ai.google.dev/gemini-api/docs/models/gemini-3-pro-image).
- **Video-to-image generation support**: You can now pass a video file (via direct upload or as a
public YouTube URL) as multimodal context alongside a text prompt to generate high-quality
thumbnails, cinematic movie posters, or summary infographics. This feature is supported exclusively
on the `gemini-3.1-flash-image` model. To learn more, see the [Video-to-image
generation](https://ai.google.dev/gemini-api/docs/image-generation#video-to-image) guide.
- Deprecation announcement: The `gemini-3.1-flash-image-preview` and `gemini-3-pro-image-preview`
models are deprecated and will be [shut down](https://ai.google.dev/gemini-api/docs/deprecations)
on June 25, 2026.

## May 25, 2026

- The `gemini-3.1-flash-lite-preview` model has been [shut
down](https://ai.google.dev/gemini-api/docs/deprecations). Use
[`gemini-3.1-flash-lite`](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite)
instead.

## May 19, 2026

- Released `gemini-3.5-flash`, the generally available (GA) version of [Gemini 3.5
Flash](https://ai.google.dev/gemini-api/docs/models/gemini-3.5-flash), our most intelligent model
for sustained frontier performance on agentic and coding tasks.
- Launched the **Managed Agents in the Gemini API** in public preview. This enables developers to
build and deploy autonomous, stateful agents that run in secure, isolated Google-hosted Linux
sandbox environments. To learn more, see the [Agents
overview](https://ai.google.dev/gemini-api/docs/agents) page and the
[Quickstart](https://ai.google.dev/gemini-api/docs/managed-agents-quickstart).
- Released the general-purpose **Antigravity Agent** managed agent,
[`antigravity-preview-05-2026`](https://ai.google.dev/gemini-api/docs/models/antigravity-preview-05-
2026), in public preview. The Antigravity agent can autonomously plan, reason, write and execute
code, manage files, and browse the web inside its sandbox container. See the [Antigravity
Agent](https://ai.google.dev/gemini-api/docs/antigravity-agent) guide for code samples and
specifications.

## May 7, 2026

- Released `gemini-3.1-flash-lite`, the generally available (GA) version of [Gemini 3.1
Flash-Lite](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite), optimized for
speed, scale, and cost efficiency.
- Deprecation announcement: The `gemini-3.1-flash-lite-preview` model is deprecating on 5/11/26 and
will be [shut down](https://ai.google.dev/gemini-api/docs/deprecations) on May 25, 2026.

## May 6, 2026

- **Upcoming breaking change**: The [Interactions
API](https://ai.google.dev/gemini-api/docs/interactions) request and response schema (`outputs` →
`steps`) and output format configuration (`response_format`) are changing. The new schema becomes
the default on **May 26** and the legacy schema will be removed on **June 8**. See the [migration
guide](https://ai.google.dev/gemini-api/docs/interactions-breaking-changes-may-2026) for details.

## May 5, 2026

- Updated **File Search** to support multimodal search. You can now natively embed and search
through images using the `gemini-embedding-2` model. Grounding metadata now includes `media_id` for
visual citations and `page_numbers` that indicate where information is found. To learn more, see
the [File Search](https://ai.google.dev/gemini-api/docs/file-search) guide.

## May 4, 2026

- Launched event-driven [Webhooks](https://ai.google.dev/gemini-api/docs/webhooks) support in the
Gemini API to replace polling workflows for the Batch API and long-running operations.

## April 30, 2026

- The `gemini-robotics-er-1.5-preview` model has been [shut
down](https://ai.google.dev/gemini-api/docs/deprecations). Use
[`gemini-robotics-er-1.6-preview`](https://ai.google.dev/gemini-api/docs/models/gemini-robotics-er-1
.6-preview) instead.

## April 22, 2026

- Released `gemini-embedding-2` as generally available (GA). To learn more, see the
[Embeddings](https://ai.google.dev/gemini-api/docs/embeddings) page.

## April 21, 2026

- Released new versions of the [Deep Research](https://ai.google.dev/gemini-api/docs/deep-research)
agent with collaborative planning, visualization support, MCP server integration, and File Search:
	-
[`deep-research-preview-04-2026`](https://ai.google.dev/gemini-api/docs/models/deep-research-preview
-04-2026): Designed for speed and efficiency, ideal to be streamed back to a client UI.
		-
[`deep-research-max-preview-04-2026`](https://ai.google.dev/gemini-api/docs/models/deep-research-max
-preview-04-2026): Maximum comprehensiveness for automated context gathering and synthesis.

## April 15, 2026

- Launched [Gemini 3.1 Flash TTS
Preview](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-tts-preview), our
cost-efficient, expressive, and steerable text to speech model. Read the
[Text-to-Speech](https://ai.google.dev/gemini-api/docs/speech-generation) docs to learn more.

## April 14, 2026

- Released `gemini-robotics-er-1.6-preview`, our updated robotics model. It now has new
capabilities like instrument reading, improved spatial and physical reasoning capabilities. To
learn more, see [Gemini Robotics-ER](https://ai.google.dev/gemini-api/docs/robotics-overview) page
and the [blog](https://deepmind.google/blog/gemini-robotics-er-1-6).
- Deprecation announcement: The `gemini-robotics-er-1.5-preview` model will be [shut
down](https://ai.google.dev/gemini-api/docs/deprecations) on April 30, 2026 at 9AM PST.

## April 2, 2026

- Released `gemma-4-26b-a4b-it` and `gemma-4-31b-it`, available on [AI
Studio](https://aistudio.google.com/) and through the Gemini API, as part of the [Gemma
4](https://ai.google.dev/gemma/docs/core) launch.

## April 1, 2026

- Introduced the new [Flex](https://ai.google.dev/gemini-api/docs/flex-inference) and
[Priority](https://ai.google.dev/gemini-api/docs/priority-inference) inference tiers, offering more
options for optimizing cost or latency.

## March 31, 2026

- Launched Veo 3.1 Lite Preview,
[`veo-3.1-lite-generate-preview`](https://ai.google.dev/gemini-api/docs/models/veo-3.1-lite-generate
-preview), our most cost-efficient [video generation](https://ai.google.dev/gemini-api/docs/video)
model, designed for rapid iteration and building high-volume applications.
- The `gemini-2.5-flash-lite-preview-09-2025` model has been shut down. Use
[`gemini-3.1-flash-lite-preview`](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite
-preview) instead.

## March 26, 2026

- Released
[`gemini-3.1-flash-live-preview`](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-live
-preview), the latest audio-to-audio (A2A) model designed for real-time dialogue and voice-first AI
applications. Read the [Live API](https://ai.google.dev/gemini-api/docs/live-api) docs to get
started.

## March 25, 2026

- Launched [Lyria 3](https://ai.google.dev/gemini-api/docs/music-generation) music generation
models: [`lyria-3-clip-preview`](https://ai.google.dev/gemini-api/docs/models/lyria-3-clip-preview)
(30-second clips) and
[`lyria-3-pro-preview`](https://ai.google.dev/gemini-api/docs/models/lyria-3-pro-preview)
(full-length songs). Both models accept text and image inputs and generate high-quality, 48kHz
stereo audio. See the [Music generation](https://ai.google.dev/gemini-api/docs/music-generation)
guide for details and code samples.

## March 23, 2026

- Rolled out [Prepay and Postpay billing plans](https://ai.google.dev/gemini-api/docs/billing) in
AI Studio. Existing accounts may be affected; read the
[Billing](https://ai.google.dev/gemini-api/docs/billing) documentation for more information.

## March 18, 2026

- Released the new [Built-in Tools and Function Calling
Combination](https://ai.google.dev/gemini-api/docs/tool-combination) feature, making it possible to
use Gemini's built-in tools alongside custom function calling tools in a single API call.
- [Grounding with Google
Maps](https://ai.google.dev/gemini-api/docs/maps-grounding#supported_models) is now supported for
Gemini 3 models going forward.

## March 16, 2026

- Introduced revamped [Usage Tiers](https://ai.google.dev/gemini-api/docs/billing#about-billing)
and [Billing Account spend caps](https://ai.google.dev/gemini-api/docs/billing#tier-spend-caps) for
a better user billing experience.

## March 12, 2026

- Introduced [project-level spend
caps](https://ai.google.dev/gemini-api/docs/billing#project-spend-caps) to billing in AI Studio.

## March 10, 2026

- Released `gemini-embedding-2-preview`, our first multimodal embedding model. It supports text,
image, video, audio, and PDF inputs, mapping all modalities into a unified embedding space. To
learn more, see [Embeddings](https://ai.google.dev/gemini-api/docs/embeddings).
- Deprecation announcement: The `gemini-2.5-flash-lite-preview-09-2025` model will be [shut
down](https://ai.google.dev/gemini-api/docs/deprecations) on March 31, 2026.

## March 9, 2026

- The Gemini 3 Pro Preview model has been [shut
down](https://ai.google.dev/gemini-api/docs/deprecations). The `gemini-3-pro-preview` now points to
[`gemini-3.1-pro-preview`](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-pro-preview).

## March 3, 2026

- Launched Gemini 3.1 Flash-Lite Preview, the first Flash-Lite model in the Gemini 3 series. Read
the [model page](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-lite-preview) for
specs, specific updates, and developer guidance.

## February 26, 2026

- Launched Nano Banana 2, [Gemini 3.1 Flash Image
Preview](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-image-preview), a
high-efficiency model optimized for speed and high-volume use cases.
- Deprecation announcement: Gemini 3 Pro Preview (`gemini-3-pro-preview`) will be [shut
down](https://ai.google.dev/gemini-api/docs/deprecations) March 9, 2026.

## February 19, 2026

- Released [Gemini 3.1 Pro
Preview](https://ai.google.dev/gemini-api/docs/models/gemini-3.1-pro-preview), our latest iteration
in the new Gemini 3 series family.
- Launched a separate endpoint `gemini-3.1-pro-preview-customtools`, which is better at
prioritizing custom tools, for users building with a mix of bash and tools.

## February 18, 2026

- Deprecation announcement: The following models will be [shut
down](https://ai.google.dev/gemini-api/docs/deprecations) June 1, 2026:
	- `gemini-2.0-flash`
		- `gemini-2.0-flash-001`
		- `gemini-2.0-flash-lite`
		- `gemini-2.0-flash-lite-001`

## February 17, 2026

- The following models are [shut down](https://ai.google.dev/gemini-api/docs/deprecations):
	- `gemini-2.5-flash-preview-09-25`
		- `imagen-4.0-generate-preview-06-06`
		- `imagen-4.0-ultra-generate-preview-06-06`

## January 29, 2026

- Launched support for the Computer Use tool in `gemini-3-pro-preview` and `gemini-3-flash-preview`.

## January 21, 2026

- Changed the `latest` aliases:
	- `gemini-pro-latest` switched to `gemini-3-pro-preview`
		- `gemini-flash-latest` switched to `gemini-3-flash-preview`

## January 15, 2026

- Deprecation announcement: The following models will be [shut
down](https://ai.google.dev/gemini-api/docs/deprecations) February 17, 2026:
	- `gemini-2.5-flash-preview-09-25`
		- `imagen-4.0-generate-preview-06-06`
		- `imagen-4.0-ultra-generate-preview-06-06`
- The `gemini-2.5-flash-image-preview` model has been shut down.

## January 14, 2026

- The `text-embedding-004` model has been [shut
down](https://ai.google.dev/gemini-api/docs/deprecations).

## January 13, 2026

- Added 4k output resolutions for [Veo](https://ai.google.dev/gemini-api/docs/video) and more
support for portrait videos in all resolutions.

## January 12, 2026

- Launched model lifecycle feature. Some models will now specify the lifecycle stage and
deprecation timeline. See the following documentation for more information:
	- [Model stages](https://ai.google.dev/api/generate-content#ModelStatus)

## January 8, 2026

- Launched support for Cloud Storage buckets and any public and private DB pre-signed URL as data
input source for the Gemini API. The file size limit has also increased from 20MB to 100MB. For
details, see [File input methods guide](https://ai.google.dev/gemini-api/docs/file-input-methods).

## December 19, 2025

- Introduced a breaking change to the Interactions API public preview in v1beta. The
`total_reasoning_tokens` field has been renamed to `total_thought_tokens` to better align with the
concept of "thoughts" in thinking models.

## December 17, 2025

- Launched Gemini 3 Flash Preview, `gemini-3-flash-preview`, delivering fast frontier-class
performance that rivals larger models at a fraction of the cost. With upgraded visual and spatial
reasoning, and agentic coding capabilities. Read the documentation on some new features, including:
	- [Multimodal function
responses](https://ai.google.dev/gemini-api/docs/function-calling#multimodal)
		- [Code execution with
images](https://ai.google.dev/gemini-api/docs/code-execution#images)

## December 12, 2025

- Released `gemini-2.5-flash-native-audio-preview-12-2025`, a new native audio model for the Live
API. This update improves the model's ability to handle complex workflows. To learn more, see the
[Live API guide](https://ai.google.dev/gemini-api/docs/live-guide) and [Gemini 2.5 Flash Native
Audio](https://ai.google.dev/gemini-api/docs/models/gemini-2.5-flash-live).

## December 11, 2025

- Launched the Interactions API in Beta. This API provides a unified interface for interacting with
Gemini models and agents. To learn more, see the [Interactions
API](https://ai.google.dev/gemini-api/docs/interactions) guide.
- Launched the Gemini Deep Research Agent in preview. It can autonomously plan, execute, and
synthesize results for multi-step research tasks. See the [Deep
Research](https://ai.google.dev/gemini-api/docs/deep-research) guide for details.

## December 10, 2025

- Launched enhancements to our [text-to-speech
models](https://ai.google.dev/gemini-api/docs/speech-generation), Gemini 2.5 Flash TTS preview
(optimized for low latency) and Gemini 2.5 Pro TTS preview (optimized for quality), including
enhanced expressivity, precision pacing, and seamless dialogue.

## December 9, 2025

- The following Gemini Live API models are now shut down:
	- `gemini-2.0-flash-live-001`
		- `gemini-live-2.5-flash-preview`

## December 5, 2025

- Gemini 3 billing for [Grounding with Google
Search](https://ai.google.dev/gemini-api/docs/google-search) will begin on January 5, 2026.

## December 4, 2025

- Deprecation announcement: The `gemini-2.5-flash-image-preview` model will be shut down January
15, 2026.

## December 3, 2025

- Deprecation announcement: The `text-embedding-004` model will be shut down January 14, 2026.

## November 20, 2025

- Released Gemini 3 Pro Image Preview, `gemini-3-pro-image-preview`, the next iteration to the Nano
Banana model. Read the [Image generation](https://ai.google.dev/gemini-api/docs/image-generation)
page for more details.

## November 18, 2025

- Launched the first Gemini 3 series model, `gemini-3-pro-preview`, our state-of-the-art reasoning
and multimodal understanding model with powerful agentic and coding capabilities.
	In addition to improvements in intelligence and performance, Gemini 3 Pro Preview
introduces new behavior around:
	- [Media resolution](https://ai.google.dev/gemini-api/docs/media-resolution)
		- [Thought signatures](https://ai.google.dev/gemini-api/docs/thought-signatures)
		- [Thinking levels](https://ai.google.dev/gemini-api/docs/thinking#thinking-levels)
	Read the [Gemini 3 Developer Guide](https://ai.google.dev/gemini-api/docs/gemini-3) for
migration, new features, and specs.

## November 11, 2025

- Deprecation announcement: The following models will be shut down:
	- November 12:
		- `veo-3.0-fast-generate-preview`
				- `veo-3.0-generate-preview`
		- November 14:
		- `gemini-2.0-flash-exp-image-generation`
				- `gemini-2.0-flash-preview-image-generation`

## November 10, 2025

- The following model is shut down:
	- `imagen-3.0-generate-002`
	Use [Imagen 4](https://ai.google.dev/gemini-api/docs/imagen#imagen-4) instead. Refer to the
[Gemini deprecations table](https://ai.google.dev/gemini-api/docs/deprecations) for more details.

## November 6, 2025

- Launched the File Search API to public preview, enabling developers to ground responses in their
own data. Read the new [File Search](https://ai.google.dev/gemini-api/docs/file-search) page for
more info.

## November 4, 2025

- For [Gemini 2.5 Flash Image](https://ai.google.dev/gemini-api/docs/image-generation), the input
token count for images has been reduced from 1290 to 258, lowering the cost of image editing.
- Deprecation announcement: The following models will be shut down:
	- November 18th:
		- `gemini-2.5-flash-lite-preview-06-17`
				- `gemini-2.5-flash-preview-05-20`
		- December 2nd:
		- `gemini-2.0-flash-thinking-exp`
				- `gemini-2.0-flash-thinking-exp-01-21`
				- `gemini-2.0-flash-thinking-exp-1219`
				- `gemini-2.5-pro-preview-03-25`
				- `gemini-2.5-pro-preview-05-06`
				- `gemini-2.5-pro-preview-06-05`
		- December 9th:
		- `gemini-2.0-flash-lite-preview`
				- `gemini-2.0-flash-lite-preview-02-05`
				- `gemini-2.0-flash-exp`
				- `gemini-2.0-pro-exp`
				- `gemini-2.0-pro-exp-02-05`

## October 29, 2025

- Launched the new [logging and datasets](https://ai.google.dev/gemini-api/docs/logs-datasets) tool
for the Gemini API.

## October 20, 2025

- The following Gemini Live API models are now shut down:
	- `gemini-2.5-flash-preview-native-audio-dialog`
		- `gemini-2.5-flash-exp-native-audio-thinking-dialog`
	You can use `gemini-2.5-flash-native-audio-preview-09-2025` instead.
- Deprecation announcement: Shut down for `gemini-2.0-flash-live-001` and
`gemini-live-2.5-flash-preview` coming December 09, 2025.

## October 17, 2025

- **Grounding with Google Maps** is now generally available. For more information, see [Grounding
with Google Maps](https://ai.google.dev/gemini-api/docs/maps-grounding) documentation.

## October 15, 2025

- Released [Veo 3.1 and 3.1 Fast](https://ai.google.dev/gemini-api/docs/video#veo-3.1) models in
public preview, with new features including:
	- Extending Veo-created videos.
		- Referencing up to three images to generate a video.
		- Providing first and last frame images to generate videos from.
	This launch also added more options for Veo 3 output video durations: 4, 6, and 8 seconds.
- Deprecation announcement: Shut down for `veo-3.0-generate-preview` and
`veo-3.0-fast-generate-preview` coming November 12, 2025.

## October 7, 2025

- Launched [Gemini 2.5 Computer Use Preview](https://ai.google.dev/gemini-api/docs/computer-use)

## October 2, 2025

- Launched Gemini 2.5 Flash Image GA: [Image Generation with
Gemini](https://ai.google.dev/gemini-api/docs/image-generation)

## September 29, 2025

- The following Gemini 1.5 models are now shut down:
	- `gemini-1.5-pro`
		- `gemini-1.5-flash-8b`
		- `gemini-1.5-flash`

## September 25, 2025

- Released Gemini Robotics-ER 1.5 model in preview. See the [Robotics
overview](https://ai.google.dev/gemini-api/docs/robotics-overview) to learn about how to use the
model for your robotics application.
- Launched following preview models:
	- `gemini-2.5-flash-preview-09-2025`
		- `gemini-2.5-flash-lite-preview-09-2025`
	See the [Models](https://ai.google.dev/gemini-api/docs/models) page for details.

## September 23, 2025

- Released `gemini-2.5-flash-native-audio-preview-09-2025`, a new native audio model for the Live
API with improved function calling and speech cut off handling. To learn more, see the [Live API
guide](https://ai.google.dev/gemini-api/docs/live-guide) and [Gemini 2.5 Flash Native
Audio](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash-native-audio).

## September 16, 2025

- Deprecation announcement: The following models will be shut down in October 2025:
	- `embedding-001`
		- `embedding-gecko-001`
		- `gemini-embedding-exp-03-07` (`gemini-embedding-exp`)
	See the [Embeddings](https://ai.google.dev/gemini-api/docs/embeddings) page for details on
the latest embeddings model.

## September 10, 2025

- Released support for the [Embeddings model in Batch
API](https://ai.google.dev/gemini-api/docs/batch-api#batch-embedding), and added Batch API to the
[OpenAI compatibility library](https://ai.google.dev/gemini-api/docs/openai#batch) for even easier
ways to get started with batch queries.

## September 9, 2025

- Launched Veo 3 and Veo 3 Fast GA, with lower pricing and new options for aspect ratios,
resolution, and seeding. Read the [Veo
documentation](https://ai.google.dev/gemini-api/docs/video#model-features) for more information.

## August 26, 2025

- Launched [Gemini 2.5 Image
Preview](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash-image-preview), our latest
native image generation model.

## August 18, 2025

- Released [URL context tool](https://ai.google.dev/gemini-api/docs/url-context) to general
availability (GA), a tool for providing URLs as additional context to prompts. Support for using
URL context with the `gemini-2.0-flash` model (available during experimental release) will be
discontinued in one week.

## August 14, 2025

- Released Imagen 4 Ultra, Standard and Fast models as generally available (GA). To learn more, see
the [Imagen](https://ai.google.dev/gemini-api/docs/imagen) page.

## August 7, 2025

- `allow_adult` setting in Image to Video generation are now available in restricted regions. See
the [Veo](https://ai.google.dev/gemini-api/docs/video?example=dialogue#veo-model-parameters) page
for details.

## July 31, 2025

- Launched image-to-video generation for the Veo 3 Preview model.
- Released Veo 3 Fast Preview model.
- To learn more about Veo 3, visit the [Veo](https://ai.google.dev/gemini-api/docs/video) page.

## July 22, 2025

- Released `gemini-2.5-flash-lite`, our fast, low-cost, high-performance Gemini 2.5 model. To learn
more, see [Gemini 2.5
Flash-Lite](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash-lite).

## July 17, 2025

- Launched `veo-3.0-generate-preview`, the latest update to Veo introducing video with audio
generation. To learn more about Veo 3, visit the [Veo](https://ai.google.dev/gemini-api/docs/video)
page.
- Increased rate limits for Imagen 4 Standard and Ultra. Visit the [Rate
limits](https://ai.google.dev/gemini-api/docs/rate-limits) page for more details.

## July 14, 2025

- Released `gemini-embedding-001`, the stable version of our text embedding model. To learn more,
see [embeddings](https://ai.google.dev/gemini-api/docs/embeddings). The
`gemini-embedding-exp-03-07` model will be deprecated on August 14, 2025.

## July 7, 2025

- Launched Gemini API Batch Mode. Batch up requests and send them to process asynchronously. To
learn more, see [Batch Mode](https://ai.google.dev/gemini-api/docs/batch-mode).

## June 26, 2025

- The preview models `gemini-2.5-pro-preview-05-06` and `gemini-2.5-pro-preview-03-25` are now
redirecting to the latest stable version `gemini-2.5-pro`.
- `gemini-2.5-pro-exp-03-25` is shut down.

## June 24, 2025

- Released Imagen 4 Ultra and Standard Preview models. To learn more, see the [Image
generation](https://ai.google.dev/gemini-api/docs/image-generation) page.

## June 17, 2025

- Released `gemini-2.5-pro`, the stable version of our most powerful model, now with adaptive
thinking. To learn more, see [Gemini 2.5
Pro](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-pro) and
[Thinking](https://ai.google.dev/gemini-api/docs/thinking). `gemini-2.5-pro-preview-05-06` will be
redirected to `gemini-2.5-pro` on June 26, 2025.
- Released `gemini-2.5-flash`, our first stable 2.5 Flash model. To learn more, see [Gemini 2.5
Flash](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash).
`gemini-2.5-flash-preview-04-17` will be deprecated on July 15, 2025.
- Released `gemini-2.5-flash-lite-preview-06-17`, a low-cost, high-performance Gemini 2.5 model. To
learn more, see [Gemini 2.5 Flash-Lite
Preview](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash-lite).

## June 05, 2025

- Released `gemini-2.5-pro-preview-06-05`, a new version of our most powerful model, now with
adaptive thinking. To learn more, see [Gemini 2.5 Pro
Preview](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-pro-preview-06-05) and
[Thinking](https://ai.google.dev/gemini-api/docs/thinking). `gemini-2.5-pro-preview-05-06` will be
redirected to `gemini-2.5-pro` on June 26, 2025.

## May 27, 2025

- The last available tuning model, Gemini 1.5 Flash 001, has been shut down. Tuning is no longer
supported on any models. See [Fine tuning with the Gemini
API](https://ai.google.dev/gemini-api/docs/model-tuning).

## May 20, 2025

**API updates:**

- Launched support for [custom video
preprocessing](https://ai.google.dev/gemini-api/docs/video-understanding#customize-video-processing)
 using clipping intervals and configurable frame rate sampling.
- Launched multi-tool use, which supports configuring [code
execution](https://ai.google.dev/gemini-api/docs/code-execution) and [Grounding with Google
Search](https://ai.google.dev/gemini-api/docs/grounding) on the same `generateContent` request.
- Launched support for [asynchronous function
calls](https://ai.google.dev/gemini-api/docs/live-tools#async-function-calling) in the Live API.
- Launched an experimental [URL context tool](https://ai.google.dev/gemini-api/docs/url-context)
for providing URLs as additional context to prompts.

**Model updates:**

- Released `gemini-2.5-flash-preview-05-20`, a Gemini
[preview](https://ai.google.dev/gemini-api/docs/models#model-versions) model optimized for
price-performance and adaptive thinking. To learn more, see [Gemini 2.5 Flash
Preview](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash-preview) and
[Thinking](https://ai.google.dev/gemini-api/docs/thinking).
- Released the
[`gemini-2.5-pro-preview-tts`](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-pro-preview-t
ts) and
[`gemini-2.5-flash-preview-tts`](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash-previ
ew-tts) models, which are capable of [generating
speech](https://ai.google.dev/gemini-api/docs/speech-generation) with one or two speakers.
- Released the `lyria-realtime-exp` model, which [generates
music](https://ai.google.dev/gemini-api/docs/music-generation) in real time.
- Released `gemini-2.5-flash-preview-native-audio-dialog` and
`gemini-2.5-flash-exp-native-audio-thinking-dialog`, new Gemini models for the Live API with native
audio output capabilities. To learn more, see the [Live API
guide](https://ai.google.dev/gemini-api/docs/live-guide#native-audio-output) and [Gemini 2.5 Flash
Native Audio](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash-native-audio).
- Released `gemma-3n-e4b-it` preview, available on [AI Studio](https://aistudio.google.com/) and
through the Gemini API, as part of the [Gemma 3n](https://ai.google.dev/gemma/docs/3n) launch.

## May 7, 2025

- Released `gemini-2.0-flash-preview-image-generation`, a preview model for generating and editing
images. To learn more, see [Image
generation](https://ai.google.dev/gemini-api/docs/image-generation) and [Gemini 2.0 Flash Preview
Image
Generation](https://ai.google.dev/gemini-api/docs/models#gemini-2.0-flash-preview-image-generation).

## May 6, 2025

- Released `gemini-2.5-pro-preview-05-06`, a new version of our most powerful model, with
improvements on code and function calling. `gemini-2.5-pro-preview-03-25` will automatically point
to the new version of the model.

## April 17, 2025

- Released `gemini-2.5-flash-preview-04-17`, a Gemini
[preview](https://ai.google.dev/gemini-api/docs/models#model-versions) model optimized for
price-performance and adaptive thinking. To learn more, see [Gemini 2.5 Flash
Preview](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-flash-preview) and
[Thinking](https://ai.google.dev/gemini-api/docs/thinking).

## April 16, 2025

- Launched context caching for [Gemini 2.0
Flash](https://ai.google.dev/gemini-api/docs/models#gemini-2.0-flash).

## April 9, 2025

**Model updates:**

- Released `veo-2.0-generate-001`, a generally available (GA) text- and image-to-video model,
capable of generating detailed and artistically nuanced videos. To learn more, see the [Veo
docs](https://ai.google.dev/gemini-api/docs/video).
- Released `gemini-2.0-flash-live-001`, a public preview version of the [Live
API](https://ai.google.dev/gemini-api/docs/live) model with billing enabled.
	- **Enhanced Session Management and Reliability**
		- **Session Resumption:** Keep sessions alive across temporary network disruptions.
The API now supports server-side session state storage (for up to 24 hours) and provides handles
(session\_resumption) to reconnect and resume where you left off.
				- **Longer Sessions via Context Compression:** Enable extended
interactions beyond previous time limits. Configure context window compression with a sliding
window mechanism to automatically manage context length, preventing abrupt terminations due to
context limits.
				- **Graceful Disconnect Notification:** Receive a `GoAway` server
message indicating when a connection is about to close, allowing for graceful handling before
termination.
		- **More Control over Interaction Dynamics**
		- **Configurable Voice Activity Detection (VAD):** Choose sensitivity levels or
disable automatic VAD entirely and use new client events (`activityStart`, `activityEnd`) for
manual turn control.
		- **Configurable Interruption Handling:** Decide whether user input should
interrupt the model's response.
		- **Configurable Turn Coverage:** Choose whether the API processes all audio and
video input continuously or only captures it when the end-user is detected speaking.
		- **Configurable Media Resolution:** Optimize for quality or token usage by
selecting the resolution for input media.
		- **Richer Output and Features**
		- **Expanded Voice & Language Options:** Choose from two new voices and 30 new
languages for audio output. The output language is now configurable within `speechConfig`.
		- **Text Streaming:** Receive text responses incrementally as they are generated,
enabling faster display to the user.
		- **Token Usage Reporting:** Gain insights into usage with detailed token counts
provided in the `usageMetadata` field of server messages, broken down by modality and prompt or
response phases.

## April 4, 2025

- Released `gemini-2.5-pro-preview-03-25`, a public preview Gemini 2.5 Pro version with billing
enabled. You can continue to use `gemini-2.5-pro-exp-03-25` on the free tier.

## March 25, 2025

- Released `gemini-2.5-pro-exp-03-25`, a public experimental Gemini model with thinking mode always
on by default. To learn more, see [Gemini 2.5 Pro
Experimental](https://ai.google.dev/gemini-api/docs/models#gemini-2.5-pro-preview-03-25).

## March 12, 2025

**Model updates:**

- Launched an experimental [Gemini 2.0
Flash](https://ai.google.dev/gemini-api/docs/image-generation#gemini) model capable of image
generation and editing.
- Released `gemma-3-27b-it`, available on [AI Studio](https://aistudio.google.com/) and through the
Gemini API, as part of the [Gemma 3](https://ai.google.dev/gemma/docs/core) launch.

**API updates:**

- Added support for [YouTube URLs](https://ai.google.dev/gemini-api/docs/vision#youtube) as a media
source.
- Added support for including an [inline
video](https://ai.google.dev/gemini-api/docs/vision#inline-video) of less than 20MB.

## March 11, 2025

**SDK updates:**

- Released the [Google Gen AI SDK for TypeScript and
JavaScript](https://googleapis.github.io/js-genai) to public preview.

## March 7, 2025

**Model updates:**

- Released `gemini-embedding-exp-03-07`, an
[experimental](https://ai.google.dev/gemini-api/docs/models/experimental-models) Gemini-based
embeddings model in public preview.

## February 28, 2025

**API updates:**

- Support for [Search as a tool](https://ai.google.dev/gemini-api/docs/grounding) added to
`gemini-2.0-pro-exp-02-05`, an experimental model based on Gemini 2.0 Pro.

## February 25, 2025

**Model updates:**

- Released `gemini-2.0-flash-lite`, a generally available (GA) version of [Gemini 2.0
Flash-Lite](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-2.0-flash-lite), which is
optimized for speed, scale, and cost efficiency.

## February 19, 2025

**AI Studio updates:**

- Support for [additional regions](https://ai.google.dev/gemini-api/docs/available-regions)
(Kosovo, Greenland and Faroe Islands).

**API updates:**

- Support for [additional regions](https://ai.google.dev/gemini-api/docs/available-regions)
(Kosovo, Greenland and Faroe Islands).

## February 18, 2025

**Model updates:**

- Gemini 1.0 Pro is no longer supported. For the list of supported models, see [Gemini
models](https://ai.google.dev/gemini-api/docs/models/gemini).

## February 11, 2025

**API updates:**

- Updates on the [OpenAI libraries compatibility](https://ai.google.dev/gemini-api/docs/openai).

## February 6, 2025

**Model updates:**

- Released `imagen-3.0-generate-002`, a generally available (GA) version of [Imagen 3 in the Gemini
API](https://ai.google.dev/gemini-api/docs/imagen).

**SDK updates:**

- Released the [Google Gen AI SDK for Java](https://github.com/googleapis/java-genai) for public
preview.

## February 5, 2025

**Model updates:**

- Released `gemini-2.0-flash-001`, a generally available (GA) version of [Gemini 2.0
Flash](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-2.0-flash) that supports
text-only output.
- Released `gemini-2.0-pro-exp-02-05`, an
[experimental](https://ai.google.dev/gemini-api/docs/models/experimental-models) public preview
version of Gemini 2.0 Pro.
- Released `gemini-2.0-flash-lite-preview-02-05`, an experimental public preview
[model](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-2.0-flash-lite) optimized for
cost efficiency.

**API updates:**

- Added [file input and graph
output](https://ai.google.dev/gemini-api/docs/code-execution#input-output) support to code
execution.

**SDK updates:**

- Released the [Google Gen AI SDK for Python](https://googleapis.github.io/python-genai/) to
general availability (GA).

## January 21, 2025

**Model updates:**

- Released `gemini-2.0-flash-thinking-exp-01-21`, the latest preview version of the model behind
the [Gemini 2.0 Flash Thinking Model](https://ai.google.dev/gemini-api/docs/thinking).

## December 19, 2024

**Model updates:**

- Released Gemini 2.0 Flash Thinking Mode for public preview. Thinking Mode is a test-time compute
model that lets you see the model's thought process while it generates a response, and produces
responses with stronger reasoning capabilities.
	Read more about Gemini 2.0 Flash Thinking Mode in our [overview
page](https://ai.google.dev/gemini-api/docs/thinking-mode).

## December 11, 2024

**Model updates:**

- Released [Gemini 2.0 Flash
Experimental](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-2.0-flash) for public
preview. Gemini 2.0 Flash Experimental's partial list of features includes:
	- Twice as fast as Gemini 1.5 Pro
		- Bidirectional streaming with our Live API
		- Multimodal response generation in the form of text, images, and speech
		- Built-in tool use with multi-turn reasoning to use features like code execution,
Search, function calling, and more

Read more about Gemini 2.0 Flash in our [overview
page](https://ai.google.dev/gemini-api/docs/models/gemini-v2).

## November 21, 2024

**Model updates:**

- Released `gemini-exp-1121`, an even more powerful experimental Gemini API model.

**Model updates:**

- Updated the `gemini-1.5-flash-latest` and `gemini-1.5-flash` model aliases to use
`gemini-1.5-flash-002`.
	- Change to `top_k` parameter: The `gemini-1.5-flash-002` model supports `top_k` values
between 1 and 41 (exclusive). Values greater than 40 will be changed to 40.

## November 14, 2024

**Model updates:**

- Released `gemini-exp-1114`, a powerful experimental Gemini API model.

## November 8, 2024

**API updates:**

- Added [support for Gemini](https://ai.google.dev/gemini-api/docs/openai) in the OpenAI libraries
/ REST API.

## October 31, 2024

**API updates:**

- Added [support for Grounding with Google Search](https://ai.google.dev/gemini-api/docs/grounding).

## October 3, 2024

**Model updates:**

- Released `gemini-1.5-flash-8b-001`, a stable version of our smallest Gemini API model.

## September 24, 2024

**Model updates:**

- Released `gemini-1.5-pro-002` and `gemini-1.5-flash-002`, two new stable versions of Gemini 1.5
Pro and 1.5 Flash, for general availability.
- Updated the `gemini-1.5-pro-latest` model code to use `gemini-1.5-pro-002` and the
`gemini-1.5-flash-latest` model code to use `gemini-1.5-flash-002`.
- Released `gemini-1.5-flash-8b-exp-0924` to replace `gemini-1.5-flash-8b-exp-0827`.
- Released the [civic integrity safety
filter](https://ai.google.dev/gemini-api/docs/safety-settings#safety-filters) for the Gemini API
and AI Studio.
- Released support for two new parameters for Gemini 1.5 Pro and 1.5 Flash in Python and NodeJS:
[`frequencyPenalty`](https://ai.google.dev/api/generate-content#FIELDS.frequency_penalty) and
[`presencePenalty`](https://ai.google.dev/api/generate-content#FIELDS.presence_penalty).

## September 19, 2024

**AI Studio updates:**

- Added thumb-up and thumb-down buttons to model responses, to enable users to provide feedback on
the quality of a response.

**API updates:**

- Added support for Google Cloud credits, which can now be used towards Gemini API usage.

## September 17, 2024

**AI Studio updates:**

- Added an **Open in Colab** button that exports a prompt – and the code to run it – to a Colab
notebook. The feature doesn't yet support prompting with tools (JSON mode, function calling, or
code execution).

## September 13, 2024

**AI Studio updates:**

- Added support for compare mode, which lets you compare responses across models and prompts to
find the best fit for your use case.

## August 30, 2024

**Model updates:**

- Gemini 1.5 Flash supports [supplying JSON schema through model
configuration](https://ai.google.dev/gemini-api/docs/json-mode#supply-schema-in-config).

## August 27, 2024

**Model updates:**

- Released the following [experimental
models](https://ai.google.dev/gemini-api/docs/models/experimental-models):
	- `gemini-1.5-pro-exp-0827`
		- `gemini-1.5-flash-exp-0827`
		- `gemini-1.5-flash-8b-exp-0827`

## August 9, 2024

**API updates:**

- Added support for [PDF processing](https://ai.google.dev/gemini-api/docs/document-processing).

## August 5, 2024

**Model updates:**

- Fine-tuning support released for Gemini 1.5 Flash.

## August 1, 2024

**Model updates:**

- Released `gemini-1.5-pro-exp-0801`, a new experimental version of [Gemini 1.5
Pro](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-1.5-pro).

## July 12, 2024

**Model updates:**

- Support for Gemini 1.0 Pro Vision removed from Google AI services and tools.

## June 27, 2024

**Model updates:**

- General availability release for Gemini 1.5 Pro's 2M context window.

**API updates:**

- Added support for [code execution](https://ai.google.dev/gemini-api/docs/code-execution).

## June 18, 2024

**API updates:**

- Added support for [context caching](https://ai.google.dev/gemini-api/docs/caching).

## June 12, 2024

**Model updates:**

- Gemini 1.0 Pro Vision deprecated.

## May 23, 2024

**Model updates:**

- [Gemini 1.5 Pro](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-1.5-pro)
(`gemini-1.5-pro-001`) is generally available (GA).
- [Gemini 1.5 Flash](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-1.5-flash)
(`gemini-1.5-flash-001`) is generally available (GA).

## May 14, 2024

**API updates:**

- Introduced a 2M context window for Gemini 1.5 Pro (waitlist).
- Introduced pay-as-you-go [billing](https://ai.google.dev/gemini-api/docs/billing) for Gemini 1.0
Pro, with Gemini 1.5 Pro and Gemini 1.5 Flash billing coming soon.
- Introduced increased rate limits for the upcoming paid tier of Gemini 1.5 Pro.
- Added built-in video support to the [File API](https://ai.google.dev/api/rest/v1beta/files).
- Added plain text support to the [File API](https://ai.google.dev/api/rest/v1beta/files).
- Added support for parallel function calling, which returns more than one call at a time.

## May 10, 2024

**Model updates:**

- Released [Gemini 1.5 Flash](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-1.5-flash)
(`gemini-1.5-flash-latest`) in preview.

## April 9, 2024

**Model updates:**

- Released [Gemini 1.5 Pro](https://ai.google.dev/gemini-api/docs/models/gemini#gemini-1.5-pro)
(`gemini-1.5-pro-latest`) in preview.
- Released a new text embedding model, `text-embeddings-004`, which supports [elastic
embedding](https://ai.google.dev/gemini-api/docs/embeddings#elastic-embedding) sizes under 768.

**API updates:**

- Released the [File API](https://ai.google.dev/api/rest/v1beta/files) for temporarily storing
media files for use in prompting.
- Added support for prompting with text, image, and audio data, also known as *multimodal*
prompting. To learn more, see [Prompting with
media](https://ai.google.dev/gemini-api/docs/prompting_with_media).
- Released [System instructions](https://ai.google.dev/gemini-api/docs/system-instructions) in beta.
- Added [Function calling
mode](https://ai.google.dev/gemini-api/docs/function-calling#function_calling_mode), which defines
the execution behavior for function calling.
- Added support for the `response_mime_type` configuration option, which lets you request responses
in [JSON format](https://ai.google.dev/gemini-api/docs/api-overview#json).

## March 19, 2024

**Model updates:**

- Added support for [tuning Gemini 1.0
Pro](https://developers.googleblog.com/en/tune-gemini-pro-in-google-ai-studio-or-with-the-gemini-api
/) in Google AI Studio or with the Gemini API.

## December 13 2023

**Model updates:**

- gemini-pro: New text model for a wide variety of tasks. Balances capability and efficiency.
- gemini-pro-vision: New multimodal model for a wide variety of tasks. Balances capability and
efficiency.
- embedding-001: New embeddings model.
- aqa: A new specially tuned model that is trained to answer questions using text passages for
grounding generated answers.

See [Gemini models](https://ai.google.dev/gemini-api/docs/models/gemini) for more details.

**API version updates:**

- v1: The stable API channel.
- v1beta: Beta channel. This channel has features that may be under development.

See [the API versions topic](https://ai.google.dev/gemini-api/docs/api-versions) for more details.

**API updates:**

- `GenerateContent` is a single unified endpoint for chat and text.
- Streaming available through the `StreamGenerateContent` method.
- Multimodal capability: Image is a new supported modality
- New beta features:
- Updated candidate count: Gemini models only return 1 candidate.
- Different Safety Settings and SafetyRating categories. See [safety
settings](https://ai.google.dev/gemini-api/docs/safety-settings) for more details.
- Tuning models is not yet supported for Gemini models (Work in progress).

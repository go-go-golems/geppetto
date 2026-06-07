---
Title: New Gemini API Updates for Gemini 3
SourceURL: https://developers.googleblog.com/new-gemini-api-updates-for-gemini-3/
SourceTool: defuddle
FetchedAt: 2026-06-05T09:02:43-04:00
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

## New Gemini API updates for Gemini 3

NOV. 25, 2025

[Shrestha Basu Mallick](https://developers.googleblog.com/search/?author=Shrestha+Basu+Mallick)
Product Google DeepMind

[Philipp Schmid](https://developers.googleblog.com/search/?author=Philipp+Schmid) Developer
Relations Engineer

![GeminiAPI_Wagtial_RD1-V01](https://storage.googleapis.com/gweb-developer-goog-blog-assets/images/G
eminiAPI_Wagtial_RD1-V01.original.png)

Gemini 3, our most intelligent model is available for developers to build with via the Gemini API.
To support its state-of-the-art reasoning, autonomous coding and multimodal understanding and
powerful agentic capabilities, we have rolled out several updates to the Gemini API. These changes
are designed to give you more control over how the model reasons, how it processes media, and how
it interacts with the outside world.

### Here is what is new in the Gemini API for Gemini 3

- **Simplified parameters for thinking control:** Starting Gemini 3 onwards, we are introducing a
new parameter called
[thinking\_level](https://ai.google.dev/gemini-api/docs/gemini-3?thinking=high#thinking_level) to
control the maximum depth of the model’s thinking process before it produces a response. Gemini 3
treats these levels as relative guidelines for reasoningrather than strict token guarantees. The
thinking\_level parameter allows you to adjust the depth of the model's internal reasoning. You can
set it to "high" for complex tasks that require optimal thinking (e.g. strategic business analysis,
or scanning code for vulnerabilities). You can set it to "low" for latency and cost-sensitive
applications such as structured data extraction, summarization etc. Read more
[here](https://ai.google.dev/gemini-api/docs/thinking#thinking-levels)
- **Granular control over multimodal vision processing:** The media\_resolution parameter lets you
configure how many tokens are used for image, video and document inputs, allowing you to balance
visual fidelity with token usage. The resolution can be set using media\_resolution\_low,
media\_resolution\_medium, or media\_resolution\_high per individual media part or globally. If
unspecified, the model uses [optimal defaults based on the media
type](https://ai.google.dev/gemini-api/docs/gemini-3?thinking=high#media_resolution). Higher
resolutions improve the model's ability to read fine text or identify small details, but increase
token usage and latency.
- **Thought signatures to improve function calling and image generation performance**: Starting
with Gemini 3, we are enforcing the return of " [Thought
Signatures.](https://ai.google.dev/gemini-api/docs/thought-signatures)". These are encrypted
representations of the model's internal thought process. By passing these signatures back to the
model in subsequent API calls, you ensure that Gemini 3 maintains its chain of reasoning across a
conversation. This is critical for complex, multi-step agentic workflows where preserving the "why"
behind a decision is just as important as the decision itself. If you [use the official
SDKs](https://ai.google.dev/gemini-api/docs/libraries) and standard chat history, thought
signatures are handled automatically for you. On the API the validations work as follows
	- Function Calling has strict validation on the “ [current
turn](https://ai.google.dev/gemini-api/docs/thought-signatures#how_it_works) ”. Missing
signatures will result in a 400 error. For understanding of how signatures show up for various
function calling scenarios please read
[here](https://ai.google.dev/gemini-api/docs/thought-signatures#function-calling)
		- For Text/chat generation, validation is not strictly enforced, but omitting
signatures will degrade the model's reasoning and answer quality.
		- Image generation/editing has strict validation for all model parts including a
thoughtSignature. Missing signatures will result in a 400 error.
- **Grounding and URL Context with Structured Outputs:** You can now combine Gemini hosted tools,
specifically [Grounding with Google Search and URL context with structured
outputs.](https://ai.google.dev/gemini-api/docs/structured-output?example=recipe#structured_outputs_
with_tools) This is especially powerful for building agents that need to fetch live information
from the web or specific webpages and extract that data into a precise JSON format for downstream
tasks.
- **Updates to Grounding with Google Search pricing:** To better support dynamic agentic workflows,
we are transitioning our pricing model from a flat rate (US$35/1k prompts) to a more granular,
usage-based rate of US$14 per 1,000 search queries.

### Best practices for using Gemini 3 Pro through our APIs

We have seen wide excitement for Gemini 3 Pro especially with vibe coding, zero-shot generation,
mathematical problem solving, complex multimodal understanding challenges and a variety of other
use cases. In order to get the best results while pushing the boundaries of Gemini 3. More details
[here](https://www.philschmid.de/gemini-3-prompt-practices).

- **Temperature:** We strongly recommend keeping the temperature parameter at its default value of
1.0
- **Consistency & Defined Parameters:** Maintain a uniform structure throughout your prompts (e.g.,
standardized XML tags) and explicitly define ambiguous terms.
- **Output Verbosity:** By default, Gemini 3 is less verbose and prefers providing direct,
efficient answers. If you require a more conversational or "chatty" response, you must explicitly
ask for it.
- **Multimodal Coherence:** Text, images, audio, or video should all be treated as equal-class
inputs. Instructions should reference specific modalities clearly to ensure the model synthesizes
across them rather than analyzing them in isolation.
- **Constraint Placement:** Place behavioral constraints and role definitions in the System
Instruction or at the very top of the prompt to ensure they anchor the model's reasoning process.
- **Long Context Structure:** When working with large contexts (books, codebases, long videos),
place your specific instructions at the **end** of the prompt (after the data context).

Gemini 3 Pro is our most advanced model for [agentic
coding](https://ai.google.dev/gemini-api/docs/prompting-strategies#agentic-workflows). To help
developers get the best of its capabilities, we’ve worked with our research team to create a
[System Instructions
template](https://ai.google.dev/gemini-api/docs/prompting-strategies#agentic-si-template) for the
model that improved performance on several agentic benchmarks.

To start building with these new features, check out the [**Gemini 3
documentation**](https://ai.google.dev/gemini-api/docs/gemini-3) and read the [**Developer
Guide**](https://ai.google.dev/gemini-api/docs/gemini-3) for technical implementation details.

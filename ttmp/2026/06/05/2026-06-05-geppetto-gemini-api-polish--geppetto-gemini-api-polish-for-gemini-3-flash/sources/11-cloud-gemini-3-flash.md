---
Title: Gemini 3 Flash Enterprise Model Doc
SourceURL: https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/gemini/3-flash
SourceTool: defuddle
FetchedAt: 2026-06-05T09:02:53-04:00
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

Gemini 3 Flash combines Gemini 3 Pro's reasoning capabilities with the Flash line's levels on
latency, efficiency, and cost. It not only enables everyday tasks with improved reasoning, but is
designed to tackle the most complex agentic workflows.

Gemini 3 Flash uses several new features to improve performance, control, and multimodal fidelity:

- **Thinking level**: Use the `thinking_level` parameter to control the amount of internal
reasoning the model performs (*minimal*, *low*, *medium*, or *high*) to balance response quality,
reasoning complexity, latency, and cost. The `thinking_level` parameter replaces `thinking_budget`
for Gemini 3 models.
	For details on the different thinking levels, see
[Thinking](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/thinking#gemini-3-a
nd-later-models).
- **Thought signatures**: Stricter validation of [thought
signatures](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/thought-signatures
) improves reliability in multi-turn function calling.
- **Media resolution**: Use the `media_resolution` parameter (*low*, *medium*, *high*, or *ultra
high*) to control vision processing for multimodal inputs, impacting token usage and latency. See
[Get started with Gemini
3](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/start/get-started-with-gemi
ni-3#media_resolution) for default resolution settings.
	- The *ultra high* media resolution level is only available for the `IMAGE` modality.
		- PDF token counts will be listed under the `IMAGE` modality instead of the
`DOCUMENT` modality in `usage_metadata`.
- **Multimodal function responses**: Function responses can now include [multimodal objects like
images and PDFs in addition to
text](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/tools/function-calling#m
m-fr).
- **Streaming Function calling**: [Stream partial function call
arguments](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/tools/function-call
ing#streaming-fc) to improve user experience during tool use.

For more information on using these features, see [Get started with Gemini
3](https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/start/get-started-with-gemi
ni-3).

[Try in Agent
Platform](https://console.cloud.google.com/agent-platform/studio/multimodal?model=gemini-3-flash-pre
view) [View in Model
Garden](https://console.cloud.google.com/agent-platform/publishers/google/gemini-3-flash-preview)
[(Preview) Deploy example
app](https://console.cloud.google.com/agent-platform/studio/multimodal?suggestedPrompt=How%20does%20
AI%20work&deploy=true&model=gemini-3-flash-preview)

Note: To use the "Deploy example app" feature, you need a Google Cloud project with billing and
Agent Platform API enabled.

<table><tbody><tr><th>Model ID</th><td
colspan="2"><code>gemini-3-flash-preview</code></td></tr><tr><th>Supported inputs & outputs</th><td
colspan="2"><ul><li>Inputs:<p>Text, Code, Images, Audio, Video,
PDF</p></li><li>Outputs:<p>Text</p></li></ul></td></tr><tr><th>Token limits</th><td
colspan="2"><ul><li>Maximum input tokens: 1,048,576</li><li>Maximum output tokens:
65,536</li></ul></td></tr><tr><th>Capabilities</th><td colspan="2"></td></tr><tr><th
rowspan="2">Consumption options</th><td colspan="2"><ul><section><li>Not
supported</li></section></ul></td></tr><tr><td colspan="2">See <a
href="https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/deploy/consumption-optio
ns">Consumption options</a> for more information.</td></tr><tr><th rowspan="6">Technical
specifications</th></tr><tr><td><b>Images</b></td><td><ul><li>Maximum images per prompt:
3000</li><li>Maximum file size per file for inline data or direct uploads through the console: 7
MB</li><li>Maximum file size per file from Google Cloud Storage: 30 MB</li><li>Default resolution
tokens: 1120</li><li>Supported MIME types:<p><code>image/png</code>, <code>image/jpeg</code>,
<code>image/webp</code>, <code>image/heic</code>,
<code>image/heif</code></p></li></ul></td></tr><tr><td><b>Documents</b></td><td><ul><li>Maximum
number of files per prompt: 3000</li><li>Maximum number of pages per file: 3000</li><li>Maximum
file size per file for the API or Cloud Storage imports: 50 MB(application/pdf) or 7
MB(text/plain)</li><li>Maximum file size per file for direct uploads through the console: 7
MB</li><li>Default resolution tokens: 560</li><li>OCR for scanned PDFs: Not used by
default</li><li>Supported MIME types:<p><code>application/pdf</code>,
<code>text/plain</code></p></li></ul></td></tr><tr><td><b>Video</b></td><td><ul><li>Maximum video
length (with audio): Approximately 45 minutes</li><li>Maximum video length (without audio):
Approximately 1 hour</li><li>Maximum number of videos per prompt: 10</li><li>Default resolution
tokens per frame: 70</li><li>Supported MIME types:<p><code>video/x-flv</code>,
<code>video/quicktime</code>, <code>video/mpeg</code>, <code>video/mpegs</code>,
<code>video/mpg</code>, <code>video/mp4</code>, <code>video/webm</code>, <code>video/wmv</code>,
<code>video/3gpp</code></p></li></ul></td></tr><tr><td><b>Audio</b></td><td><ul><li>Maximum audio
length per prompt: Approximately 8.4 hours, or up to 1 million tokens</li><li>Maximum number of
audio files per prompt: 1</li><li>Speech understanding for: Audio summarization, transcription, and
translation</li><li>Supported MIME types:<p><code>audio/x-aac</code>, <code>audio/flac</code>,
<code>audio/mp3</code>, <code>audio/m4a</code>, <code>audio/mpeg</code>, <code>audio/mpga</code>,
<code>audio/mp4</code>, <code>audio/ogg</code>, <code>audio/pcm</code>, <code>audio/wav</code>,
<code>audio/webm</code></p></li></ul></td></tr><tr><td><b>Parameter
defaults</b></td><td><ul><li>Temperature: 0.0-2.0 (default 1.0)</li><li>topP: 0.0-1.0 (default
0.95)</li><li>topK: 64 (fixed)</li><li>candidateCount: 1–8 (default 1)</li></ul></td></tr><tr><th
rowspan="3">Supported regions</th></tr><tr><td><p>Model
availability</p></td><td><ul><section><li>Global</li><ul><li>global</li></ul></section></ul></td></t
r><tr><td colspan="2">See <a
href="https://docs.cloud.google.com/gemini-enterprise-agent-platform/resources/locations">Deployment
s and endpoints</a> for more information.</td></tr><tr><th>Knowledge cutoff date</th><td
colspan="2">January 2025</td></tr><tr><th>Versions</th><td
colspan="2"><ul><li><code>gemini-3-flash-preview</code></li></ul></td></tr><tr><th>Supported
languages</th><td>See <a
href="https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/google-models#expandable
-1">Supported languages</a>.</td></tr><tr><th>Pricing</th><td colspan="2">See <a
href="https://cloud.google.com/gemini-enterprise-agent-platform/generative-ai/pricing">Pricing</a>.<
/td></tr></tbody></table>

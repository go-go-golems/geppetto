Anthropic Beta SDK: streaming events and content block types

The Anthropic Python SDK exposes a beta namespace with type‑definitions for the streaming API. In the streaming API, the server pushes a sequence of events using Server‑Sent Events (SSE). Each event has a type field that acts as a discriminator and determines the structure of the accompanying data. This report summarises the events and content blocks defined under src/anthropic/types/beta in the anthropic‑sdk‑python repository (commit from Oct 2025). All information below comes directly from the auto‑generated type definitions in the repository.

Streaming event overview

Streaming events are exposed through the BetaRawMessageStreamEvent type alias, which is a union of six event classes. The union’s type discriminator tells the SDK which class to instantiate
raw.githubusercontent.com
.

Event type string	Python class	Key fields / notes
message_start	BetaRawMessageStartEvent	Contains a complete message object representing the assistant message that will be streamed
raw.githubusercontent.com
. Appears only once at the start of the stream.
message_delta	BetaRawMessageDeltaEvent	Contains a delta object (information about the container and stop reason), optional context_management information, and a usage object that accumulates token and tool‑usage statistics
raw.githubusercontent.com
.
message_stop	BetaRawMessageStopEvent	Indicates that the message has finished streaming; it has only the type field
raw.githubusercontent.com
.
content_block_start	BetaRawContentBlockStartEvent	Carries a content_block describing the new block of content being streamed and the zero‑based index of the block
raw.githubusercontent.com
.
content_block_delta	BetaRawContentBlockDeltaEvent	Contains a delta for the block and the block index
raw.githubusercontent.com
. The delta describes incremental updates to the block (e.g. new text, thinking, citations, etc.)
raw.githubusercontent.com
.
content_block_stop	BetaRawContentBlockStopEvent	Marks the end of a content block; includes the index
raw.githubusercontent.com
.
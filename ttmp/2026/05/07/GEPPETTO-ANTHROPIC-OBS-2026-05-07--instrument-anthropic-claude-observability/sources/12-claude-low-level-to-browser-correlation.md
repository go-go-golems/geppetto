---
Title: Claude low-level to browser-debug correlation
Ticket: GEPPETTO-ANTHROPIC-OBS-2026-05-07
Status: done
Topics:
  - observability
  - llm
  - inference
DocType: source
Intent: evidence
Owners:
  - manuel
Summary: Correlates Claude provider events, Geppetto publish records, backend pipeline/transport records, and frontend stream-debug entries.
LastUpdated: 2026-05-07T16:58:00-04:00
---

# Claude low-level to browser-debug correlation

## Run metadata

- Session: `392d86c9-2517-4a3a-97e7-2b6be6a0ec73`
- Profile: `haiku`
- Provider/model: `claude` / `claude-haiku-4-5`
- Prompt: `Say exactly: pong`
- Output: `pong`
- Browser stream debug was enabled for this run.

## Summary

The same assistant response can be followed through four layers:

1. **Claude provider events** in Geppetto records (`provider_routed_event`).
2. **Geppetto publish boundary** records (`geppetto_publish_started`, compact and started-only).
3. **Sessionstream/Pinocchio backend records** (`pipeline` ordinals and websocket `ui_event_sent`).
4. **Browser stream debug entries** (`raw-ws`, `parsed-frame`, and `ui-event`).

The important backend-to-browser join key is the **UI event ordinal**. Provider records do not carry UI ordinals directly. The text and final provider events can be correlated by order and timestamp to nearby publish-started records and then to pipeline ordinals. Some provider events, such as `message_start`, `content_block_start`, and `ping`, are low-level diagnostics only and have no direct browser UI event.

## Correlation table

| Step | Claude provider event(s) | Geppetto publish | Backend ordinal / browser event | Browser mapping | Provider → browser | Publish → browser | Backend → browser | Notes |
|---|---|---|---|---|---:|---:|---:|---|
| User accepted | `n/a` | `n/a` | `1` / `ChatMessageAccepted` | direct UI | — | — | -0.3 ms | User event, not provider-driven. |
| Assistant placeholder | `n/a` | `n/a` | `2` / `ChatMessageStarted` | pre-provider UI | — | — | 1.3 ms | Pinocchio creates the streaming assistant placeholder before the Claude SSE stream yields provider events. |
| Claude stream opens | `message_start, content_block_start, ping` | `start` | `—` / `—` | no direct UI ordinal | — | — | — | message_start/content_block_start/ping are useful provider diagnostics but do not add browser-visible content. |
| Text delta #1 | `content_block_delta` | `partial` | `3` / `ChatMessageAppended` | content UI | 1.8 ms | 1.8 ms | 0.8 ms | Claude text_delta length 1 becomes chunk/content "p". |
| Text delta #2 | `content_block_delta` | `partial` | `4` / `ChatMessageAppended` | content UI | 1.1 ms | 1.1 ms | 0.3 ms | Claude text_delta length 3 becomes chunk "ong" and content "pong". |
| Claude stream closes | `content_block_stop, message_delta` | `partial, partial` | `—` / `—` | no standalone UI ordinal | — | — | — | content_block_stop and message_delta advance provider/publish diagnostics; browser finalization waits for message_stop/final handling. |
| Finalization | `message_stop` | `final` | `5` / `ChatMessageFinished` | final UI | 0.9 ms | 0.9 ms | -0.3 ms | message_stop plus final publish maps to browser finished state. |
| Preview cleanup | `message_stop` | `final` | `5` / `ChatAgentModePreviewCleared` | same backend ordinal | 2.9 ms | 2.9 ms | 1.7 ms | The same backend ordinal also clears transient agent preview state. |


## Backend pipeline ordinals

| Ordinal | Pipeline event | UI events | Payload summary |
|---:|---|---|---|
| 1 | `ChatUserMessageAccepted` | `ChatMessageAccepted` | messageId='chat-msg-3-user', content='Say exactly: pong' |
| 2 | `ChatInferenceStarted` | `ChatMessageStarted` | messageId='chat-msg-3', status='streaming' |
| 3 | `ChatTokensDelta` | `ChatMessageAppended` | messageId='chat-msg-3:text:1', chunk='p', content='p', status='streaming' |
| 4 | `ChatTokensDelta` | `ChatMessageAppended` | messageId='chat-msg-3:text:1', chunk='ong', content='pong', status='streaming' |
| 5 | `ChatInferenceFinished` | `ChatMessageFinished, ChatAgentModePreviewCleared` | messageId='chat-msg-3:text:1', content='pong', status='finished' |


## Frontend stream-debug sequence

| ID | Type | Ordinal | Name/frame | Message | Size |
|---:|---|---:|---|---|---:|
| 1 | `ws-lifecycle` |  | `connect-start` | `` |  |
| 2 | `ws-lifecycle` |  | `open` | `` |  |
| 3 | `raw-ws` |  | `` | `` | 35 |
| 4 | `parsed-frame` |  | `hello` | `conn-3` |  |
| 5 | `raw-ws` |  | `` | `` | 65 |
| 6 | `parsed-frame` |  | `snapshot` | `` |  |
| 7 | `snapshot` |  | `` | `` |  |
| 8 | `raw-ws` |  | `` | `` | 67 |
| 9 | `parsed-frame` |  | `subscribed` | `` |  |
| 10 | `raw-ws` |  | `` | `` | 301 |
| 11 | `parsed-frame` | 1 | `ChatMessageAccepted` | `` |  |
| 12 | `ui-event` | 1 | `ChatMessageAccepted` | `chat-msg-3-user` |  |
| 13 | `raw-ws` |  | `` | `` | 311 |
| 14 | `parsed-frame` | 2 | `ChatMessageStarted` | `` |  |
| 15 | `ui-event` | 2 | `ChatMessageStarted` | `chat-msg-3` |  |
| 16 | `raw-ws` |  | `` | `` | 426 |
| 17 | `parsed-frame` | 3 | `ChatMessageAppended` | `` |  |
| 18 | `ui-event` | 3 | `ChatMessageAppended` | `chat-msg-3:text:1` |  |
| 19 | `raw-ws` |  | `` | `` | 434 |
| 20 | `parsed-frame` | 4 | `ChatMessageAppended` | `` |  |
| 21 | `ui-event` | 4 | `ChatMessageAppended` | `chat-msg-3:text:1` |  |
| 22 | `raw-ws` |  | `` | `` | 414 |
| 23 | `parsed-frame` | 5 | `ChatMessageFinished` | `` |  |
| 24 | `ui-event` | 5 | `ChatMessageFinished` | `chat-msg-3:text:1` |  |
| 25 | `raw-ws` |  | `` | `` | 243 |
| 26 | `parsed-frame` | 5 | `ChatAgentModePreviewCleared` | `` |  |
| 27 | `ui-event` | 5 | `ChatAgentModePreviewCleared` | `chat-msg-3:text:1` |  |


## Interpretation

- The browser receives `ChatMessageStarted` before Claude provider events arrive; that placeholder is produced by Pinocchio's inference lifecycle, not by Claude `message_start`.
- Claude `message_start`, `content_block_start`, and `ping` are visible in Geppetto as low-level provider diagnostics but have no direct frontend UI event.
- The two `content_block_delta` records are the browser-visible text stream: first `p`, then `ong`. They map to backend ordinals 3 and 4 and browser `ChatMessageAppended` entries whose content becomes `p` then `pong`.
- Claude's termination sequence (`content_block_stop`, `message_delta`, `message_stop`) is compressed on the UI side: the browser sees ordinal 5 as `ChatMessageFinished` plus `ChatAgentModePreviewCleared`.
- There are no `geppetto_publish_done` records, matching the started-only publish policy.
- Backend-to-browser fanout is effectively same-millisecond in this run. Provider-to-browser latency for text/final events is about 1–3 ms after provider receipt.

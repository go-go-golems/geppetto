---
Title: Claude web-chat runthrough event-size analysis
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
Summary: Size and stage analysis from the successful Claude web-chat Playwright runthrough.
LastUpdated: 2026-05-07T16:45:00-04:00
---

# Claude web-chat runthrough event-size analysis

## Run metadata

- Session: `83473858-b729-49d9-8c33-045efbdd98cd`
- Profile endpoint result: `haiku` from registry `default`
- Visible selector value during the successful run: `haiku`
- Prompt: `Say exactly: pong`
- Visible assistant output: `pong`
- Browser console warnings/errors: none
- Backend API status: session creation, message post, and debug endpoints returned HTTP 200
- Trace level: `provider`

## Record counts and serialized sizes

- Combined backend debug records: 43 records, 20327 bytes
- Geppetto records: 14 records, 6518 bytes
- Pipeline records JSON: 8221 bytes
- Transport records JSON: 5773 bytes
- Reconcile JSON: 252 bytes

## Provider distribution

- `claude`: 14

## Geppetto stage distribution

- `provider_routed_event`: 8
- `geppetto_publish_started`: 6

## Geppetto event type distribution

- `message_start`: 1
- `start`: 1
- `content_block_start`: 1
- `ping`: 1
- `content_block_delta`: 2
- `partial`: 4
- `content_block_stop`: 1
- `message_delta`: 1
- `message_stop`: 1
- `final`: 1


## Observations

- Claude observability is active in web-chat after wiring `WithClaudeOptions` into the Pinocchio web-chat engine factory.
- Provider routed records were captured: 8.
- Compact publish-started records were captured: 6.
- Publish-done records captured: 0; expected value is zero under the current policy.
- The run validated the success path after Anthropic credits were added.
- Compared with the OpenAI Responses run, Claude produced fewer/lighter Geppetto records for this short prompt.

## Largest Geppetto records

1. 709 bytes — provider=`claude`, model=`claude-haiku-4-5`, stage=`provider_routed_event`, eventType=`message_start`, responseID=`msg_01We9TjNT7v1qmsxVAhDK8Lr`
2. 595 bytes — provider=`claude`, model=`claude-haiku-4-5`, stage=`provider_routed_event`, eventType=`message_delta`, responseID=``
3. 527 bytes — provider=`claude`, model=`claude-haiku-4-5`, stage=`provider_routed_event`, eventType=`content_block_delta`, responseID=``
4. 525 bytes — provider=`claude`, model=`claude-haiku-4-5`, stage=`provider_routed_event`, eventType=`content_block_delta`, responseID=``
5. 485 bytes — provider=`claude`, model=`claude-haiku-4-5`, stage=`provider_routed_event`, eventType=`content_block_start`, responseID=``
6. 451 bytes — provider=`claude`, model=`claude-haiku-4-5`, stage=`provider_routed_event`, eventType=`content_block_stop`, responseID=``
7. 423 bytes — provider=`claude`, model=`claude-haiku-4-5`, stage=`provider_routed_event`, eventType=`message_stop`, responseID=``
8. 407 bytes — provider=`claude`, model=`claude-haiku-4-5`, stage=`provider_routed_event`, eventType=`ping`, responseID=``

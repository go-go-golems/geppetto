---
Title: OpenAI Responses Web-Chat Runthrough Event Size Analysis
Ticket: GEPPETTO-OPENAI-OBS-2026-05-07
Status: active
Topics:
  - observability
  - openai
  - webchat
DocType: reference
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Event size metrics from the Playwright web-chat runthrough using the gpt-5-nano-low OpenAI Responses profile.
LastUpdated: 2026-05-07T15:16:00-04:00
WhatFor: "Use to understand which debug and observability payloads dominate trace size in the OpenAI Responses browser runthrough."
WhenToUse: "When tuning Geppetto/Pinocchio debug record retention or deciding which observability payloads to keep compact."
---

# OpenAI Responses Web-Chat Runthrough Event Size Analysis

- Session: `9657aaaf-4bc8-4256-b429-5299e3e9af99`
- Profile selected in browser: `gpt-5-nano-low` (inherits `openai-responses-base`)
- Prompt: `Say exactly: pong`
- Visible assistant response: `pong`
- Debug trace level: `provider`

## Aggregate JSON payload sizes

| Source | Records | JSON bytes |
|---|---:|---:|
| Backend debug `/records` | 52 | 34462 |
| Backend Geppetto `/geppetto` | 19 | 18578 |
| Frontend stream debug entries | 30 | 15217 |
| Reconcile SQLite export | n/a | 266240 |

## Backend debug records by kind

| Kind | Count | Serialized bytes | Average | Max |
|---|---:|---:|---:|---:|
| geppetto | 19 | 18476 | 972 | 3343 |
| pipeline | 6 | 9458 | 1576 | 2147 |
| transport | 27 | 6411 | 237 | 326 |

## Geppetto records by stage

| Stage | Count | Serialized bytes | Average | Max | Object JSON bytes | Event JSON bytes | Metadata JSON bytes |
|---|---:|---:|---:|---:|---:|---:|---:|
| provider_routed_event | 12 | 15349 | 1279 | 3343 | 8870 | 0 | 0 |
| geppetto_publish_started | 7 | 3127 | 446 | 570 | 0 | 0 | 0 |

## Geppetto provider object sizes by event type

| Event type | Count | Object JSON bytes | Average | Max |
|---|---:|---:|---:|---:|
| response.completed | 1 | 2862 | 2862 | 2862 |
| response.output_item.done | 2 | 2008 | 1004 | 1726 |
| response.output_item.added | 2 | 1368 | 684 | 1151 |
| response.in_progress | 1 | 864 | 864 | 864 |
| response.created | 1 | 860 | 860 | 860 |
| response.content_part.done | 1 | 234 | 234 | 234 |
| response.content_part.added | 1 | 231 | 231 | 231 |
| response.output_text.delta | 1 | 217 | 217 | 217 |
| response.output_text.done | 1 | 186 | 186 | 186 |
| keepalive | 1 | 40 | 40 | 40 |

## Frontend stream debug records by type

| Type | Count | Serialized bytes | Average | Max |
|---|---:|---:|---:|---:|
| raw-ws | 10 | 8034 | 803 | 1397 |
| parsed-frame | 10 | 4659 | 465 | 739 |
| ui-event | 7 | 2074 | 296 | 430 |
| ws-lifecycle | 2 | 269 | 134 | 147 |
| snapshot | 1 | 150 | 150 | 150 |

## Top Geppetto records by serialized size

| Bytes | Stage | Event type | Object JSON bytes |
|---:|---|---|---:|
| 3343 | provider_routed_event | response.completed | 2862 |
| 2295 | provider_routed_event | response.output_item.done | 1726 |
| 1721 | provider_routed_event | response.output_item.added | 1151 |
| 1347 | provider_routed_event | response.in_progress | 864 |
| 1339 | provider_routed_event | response.created | 860 |
| 852 | provider_routed_event | response.output_item.done | 282 |
| 805 | provider_routed_event | response.content_part.done | 234 |
| 803 | provider_routed_event | response.content_part.added | 231 |
| 788 | provider_routed_event | response.output_item.added | 217 |
| 788 | provider_routed_event | response.output_text.delta | 217 |

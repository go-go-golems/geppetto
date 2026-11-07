---
Title: Connection-only construction and explicit source selection
Ticket: GEPPETTO-CREDSCOPE-001
Status: active
Topics:
    - security
    - credentials
    - providers
    - geppetto
DocType: design-doc
Intent: long-term
Owners:
    - manuel
RelatedFiles:
    - Path: repo://pkg/doc/playbooks/09-migrate-connection-only-inference.md
      Note: Concise caller migration
    - Path: repo://pkg/inference/engine/factory/factory.go
      Note: Resolve authority before payload construction
    - Path: repo://pkg/steps/ai/connectionfactory/settings.go
      Note: Source-only credential resolution
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-06T19:33:59.581040185-05:00
WhatFor: ""
WhenToUse: ""
---

# Connection-only construction and explicit source selection

## Decision

Inference engine and token-counter constructors take a required `*connection.Connection` argument before payload settings. They reject nil connections and never resolve settings credentials. Provider-specific options contain only payload/observation behavior. Remove `WithConnection`, engine bearer options, token-counter authentication options, and `WithRequestTransport` rather than retaining compatibility wrappers.

A source factory resolves settings into a connection before calling payload constructors. It is independent of payload-engine packages so source resolution is directly testable and package-local engine tests can use it without cycles. Settings factories remain the convenient entry points for applications, including optional host token sources.

Responses settings select exactly the canonical `open-responses` slot. No searching `openai-responses` or `openai`, and no borrowing another slot's endpoint. An explicit connection may of course represent OpenAI or a compatible gateway independently of protocol selection. Retire the legacy Responses protocol alias in factory dispatch. Provider defaults remain restricted to their own declared slots.

Codex exposes source-side authentication and fixed routing, and constructs a connection at the provider boundary. It no longer depends on a Responses-specific request-transport configuration type. Pi passes its complete binding directly to payload constructors.

## Scope and order

1. Source factory and required inference/counter constructors; migrate factories, Pi, fixtures, tests, and examples.
2. Remove obsolete Responses source selection and Codex request-transport coupling; preserve route authority and auth refresh tests.
3. Update docs and downstream callers, validate at the integration boundary, publish a PR against merged main and request `@codex review`.

Embedding constructors already require connections. Direct rerank and decision convenience construction is a separate remaining source boundary; keep their behavior visible and do not silently expand this inference/counter cleanup into unrelated service-policy redesign. The embedding cache identity issue is tracked as #448 and remains outside this change.

## Validation

Use recording transports to assert required connection construction, strict source selection, absent secret settings in payload engines, SDK and refresh behavior, and the cross-path credential matrix. Compile all repository packages/tests to find API migrations. Socket-based tests execute in CI if local execution remains restricted. Carry forward the previous full-CI results without pretending they validate changed code.

## Risks

This intentionally breaks Go callers using old constructor signatures and profiles relying on Responses alias/fallback selection. Update in-repository and available downstream callers; provide migration examples rather than compatibility adapters. Source and connection factories must preserve timeout, outbound policy, carrier, and provider semantics already established in #446.

## Delivered API and migration

The implementation is committed at `f2650a67`. Four inference constructors and two counter constructors now take `(connection, settings, ...)`; the old auth/connection engine options are removed. `connectionfactory.New(settings, source)` selects a source declaration, and generic factories perform that resolution before construction. Codex exposes `Authentication` and canonical `NewConnection` without depending on a Responses adapter type.

`types.ApiTypeOpenAIResponses` and the CLI alias are removed. Responses configuration uses only the `open-responses` key, URL and policy slot. Pi's external catalog protocol string is translated explicitly rather than treated as a settings alias. Current examples and compiled probes have been migrated.

The concise user playbook is `pkg/doc/playbooks/09-migrate-connection-only-inference.md`, help slug `migrate-connection-only-inference`. Recording/source/factory/race checks, all-package compilation, repository/downstream builds and configured golangci analyzers pass. Full socket tests await PR CI due to the recorded sandbox restriction. Receipt: `sources/08-connection-only-validation.txt`.

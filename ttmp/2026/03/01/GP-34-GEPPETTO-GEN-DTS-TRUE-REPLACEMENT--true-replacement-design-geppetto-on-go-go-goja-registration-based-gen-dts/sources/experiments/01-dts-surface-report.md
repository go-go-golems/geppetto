# d.ts Surface Report: `/home/manuel/workspaces/2026-03-01/generate-js-types/geppetto/pkg/doc/types/geppetto.d.ts`

## Totals

- Lines: 498
- Export `const` count: 8
- Export `interface` count: 41
- Export `type` count: 3
- Export `function` count: 3

## Top-Level Export Names

- const: consts, engines, middlewares, profiles, schemas, tools, turns, version
- interface sample (first 20): AfterToolCallPayload, BeforeToolCallPayload, Block, Builder, BuilderOptions, ConnectedProfileStack, Engine, EngineOptions, ExtensionSchemaEntry, MiddlewareContext, MiddlewareRef, MiddlewareSchemaEntry, MiddlewareUse, OnToolErrorPayload, PolicySpec, Profile, ProfileEngineOptions, ProfileMetadata, ProfileMutationOptions, ProfilePatch ...
- type: MiddlewareFn, NextFn, ProfileRegistrySources
- function: createBuilder, createSession, runInference

## Grouped Object Exports

- consts: 10 members
  - BlockKind, BlockMetadataKeys, EventType, HookAction, PayloadKeys, RunMetadataKeys, ToolChoice, ToolErrorHandling, TurnDataKeys, TurnMetadataKeys
- engines: 4 members
  - echo, fromConfig, fromFunction, fromProfile
- middlewares: 2 members
  - fromJS, go
- profiles: 12 members
  - connectStack, createProfile, deleteProfile, disconnectStack, getConnectedSources, getProfile, getRegistry, listProfiles, listRegistries, resolve, setDefaultProfile, updateProfile
- schemas: 2 members
  - listExtensions, listMiddlewares
- tools: 1 members
  - createRegistry
- turns: 8 members
  - appendBlock, newAssistantBlock, newSystemBlock, newToolCallBlock, newToolUseBlock, newTurn, newUserBlock, normalize

## Feature Signals

- inline_arrow_callbacks: 7
- partial_types: 2
- promises: 2
- readonly_fields: 53
- record_types: 28
- string_literal_unions: 5

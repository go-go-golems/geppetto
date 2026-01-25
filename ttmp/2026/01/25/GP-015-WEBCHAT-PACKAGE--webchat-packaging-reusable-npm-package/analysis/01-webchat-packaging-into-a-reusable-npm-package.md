---
Title: Webchat packaging into a reusable npm package
Ticket: GP-015-WEBCHAT-PACKAGE
Status: active
Topics:
    - frontend
    - architecture
    - infrastructure
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/web/package.json
      Note: Current dependencies and build scripts
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx
      Note: Primary UI component to extract
    - Path: pinocchio/cmd/web-chat/web/src/chat/Markdown.tsx
      Note: Markdown rendering helper
    - Path: pinocchio/cmd/web-chat/web/src/sem/registry.ts
      Note: Protocol event handling
    - Path: pinocchio/cmd/web-chat/web/src/sem/timelineMapper.ts
      Note: Entity mapping and normalization
    - Path: pinocchio/cmd/web-chat/web/src/store/appSlice.ts
      Note: App status state
    - Path: pinocchio/cmd/web-chat/web/src/store/store.ts
      Note: Redux store wiring
    - Path: pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts
      Note: Timeline state and entity model
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Transport and hydration client
ExternalSources: []
Summary: In-depth analysis and blueprint for turning the webchat app into reusable npm packages with clear public APIs, integration paths, and theming hooks.
LastUpdated: 2026-01-25T17:15:40-05:00
WhatFor: Textbook-level guide to packaging a React+Redux chat UI for reuse.
WhenToUse: Use when extracting webchat into a reusable package for third-party integration.
---


# Webchat packaging into a reusable npm package

## Goal

Turn the core of the webchat app into reusable npm packages so others can:

- Embed chat into existing pages with minimal work.
- Bring their own UI widgets while reusing the networking and state core.
- Customize styling, theming, and layout without forking the repo.

This document is a detailed, textbook-style blueprint. It assumes minimal prior knowledge of React, Redux, or JavaScript packaging.

> FUNDAMENTAL: Packaging is about contracts.
> 
> A package is not just code. It is a promise that says "if you use these inputs, you get these outputs." The entire design effort is to decide which promises are stable and which are not.

## Executive summary (short)

- Split the system into layers: core client, state, and UI.
- Publish at least two packages:
  - `@org/webchat-core`: transport, protocol mapping, state machine, and a minimal event API.
  - `@org/webchat-react`: React components, hooks, and default widgets.
- Consider a third optional package:
  - `@org/webchat-embed`: a small script that creates a widget for non-React pages.
- Use clear configuration objects and well-typed public APIs.
- Keep UI theming controlled via CSS variables and class hooks.
- Maintain compatibility by pinning peer dependencies and shipping ESM + CJS.

## What exists today (webchat)

The current app lives at `pinocchio/cmd/web-chat/web` and includes:

- UI components: `src/chat/ChatWidget.tsx`, `src/chat/Markdown.tsx`, `src/components/ErrorBoundary.tsx`.
- State: Redux slices in `src/store/*` with `store.ts` wiring.
- Transport: `src/ws/wsManager.ts` for WebSocket and hydration.
- Protocol mapping: `src/sem/registry.ts` and `src/sem/timelineMapper.ts`.
- Generated protobuf types: `src/sem/pb/**`.

These are the natural extraction seams.

> FUNDAMENTAL: Identify seams where data changes form.
> 
> The best extraction points are where data is parsed, normalized, or translated. For webchat, those seams are the transport boundary and the protocol mapping boundary.

## Audience and use cases

### Primary audiences

1. **App integrators** who want a drop-in chat widget on an existing page.
2. **App builders** who want a headless client API with custom UI.
3. **Internal developers** who want to avoid forking.

### Use cases

- Embed chat in an existing React app.
- Embed chat in a non-React page with a single script tag.
- Build a custom widget that still uses the existing WebSocket protocol and hydration.

## Fundamentals (React + Redux + packaging)

> FUNDAMENTAL: React components are functions of props + state.
> 
> React is a UI engine. A component is a function: `View = f(props, state)`. Packaging a React component means you must design the `props` interface and decide where `state` lives.

> FUNDAMENTAL: Redux is a centralized state container.
> 
> Redux keeps all state in one store. Your UI reads from the store and dispatches actions. Packaging with Redux means choosing whether you expose the store to consumers or keep it internal.

> FUNDAMENTAL: npm packages are built, published, and consumed artifacts.
> 
> You do not ship source files; you ship built files (ESM and/or CJS) with type definitions. Consumers import the public API from the `exports` map.

### Fundamentals of module formats

> FUNDAMENTAL: ESM and CJS are different contracts.
> 
> ESM (`import`) is the modern standard and enables tree-shaking. CJS (`require`) is still used by older tooling. A reusable package should typically ship both to maximize compatibility.

Quick mapping:

- **ESM**: `export` / `import`. Best for modern bundlers.
- **CJS**: `module.exports` / `require`. Needed by older toolchains.
- **UMD/IIFE**: Script tag builds for embedding without a bundler.

### Fundamentals of dependency types

> FUNDAMENTAL: Dependencies vs peerDependencies is about ownership.
> 
> If the consumer must control the version (like React), use `peerDependencies`. If your package owns the version (like a small utility), use `dependencies`.

Rules of thumb:

- `react`, `react-dom`: peer dependencies.
- `@reduxjs/toolkit`, `react-redux`: peer dependencies if exposed or used in public types.
- `@bufbuild/protobuf`: dependency if hidden behind your API.

### Fundamentals of CSS distribution

> FUNDAMENTAL: CSS is part of the public API.
> 
> If you ship CSS, consumers must import it or include it. If you use CSS variables, you expose a stable theming contract.

Options:

- **CSS file**: `dist/style.css` and document `import '@org/webchat-react/style.css'`.
- **CSS-in-JS**: co-locates styles but adds runtime cost.
- **CSS modules**: good isolation but harder for theming.

### Non-goals (define early)

Explicit non-goals protect scope and reduce accidental complexity:

- We do not attempt to support every state management library in v1.
- We do not ship server-side rendering support in v1.
- We do not ship backend logging or telemetry in v1.

## Proposed package architecture

### Layered model

```
+------------------------------+
|        App Integrator        |
|  (their UI + their styles)   |
+--------------+---------------+
               |
               v
+------------------------------+
|      webchat-react package   |
|  React components + hooks    |
+--------------+---------------+
               |
               v
+------------------------------+
|      webchat-core package    |
|  transport + state + proto   |
+------------------------------+
```

### Package 1: @org/webchat-core

**Responsibility**: Everything that is not React UI.

- WebSocket connection and hydration (today in `wsManager.ts`).
- Timeline mapping and data normalization (today in `timelineMapper.ts`).
- State container (Redux or a simpler internal store).
- Event and error reporting.
- Protocol types or adapters for protobuf.

**Public API (example)**:

```ts
export type WebchatConfig = {
  baseUrl: string;
  convId?: string;
  onStatus?: (status: string) => void;
  logger?: Logger;
};

export type WebchatClient = {
  connect(): Promise<void>;
  disconnect(): void;
  sendMessage(text: string): Promise<void>;
  hydrate(): Promise<void>;
  subscribe(listener: (event: WebchatEvent) => void): () => void;
  getState(): WebchatState;
};

export function createWebchatClient(config: WebchatConfig): WebchatClient;
```

> FUNDAMENTAL: Stable APIs are smaller than you think.
> 
> Every function you export becomes a promise you must keep. Start small. You can always add API later, but removing it is painful.

### Package 2: @org/webchat-react

**Responsibility**: React components and hooks that render webchat UI on top of the core.

- `ChatProvider` creates or injects a client instance.
- `useChat` hook for state and actions.
- `ChatWidget` as a default UI.
- `ErrorBoundary` and debug panel (optional).

**Public API (example)**:

```tsx
export type ChatProviderProps = {
  config: WebchatConfig;
  children: React.ReactNode;
};

export function ChatProvider(props: ChatProviderProps): JSX.Element;

export function useChat(): {
  state: WebchatState;
  sendMessage: (text: string) => Promise<void>;
  connect: () => Promise<void>;
  disconnect: () => void;
};

export function ChatWidget(props: { className?: string }): JSX.Element;
```

### Optional Package 3: @org/webchat-embed

**Responsibility**: A small UMD or IIFE build that injects a widget into any page.

- Creates a root div.
- Bootstraps React into it.
- Accepts config via `data-*` attributes or a global function.

This package is optional but excellent for "drop-in" embedding.

## Redux integration choices

You must decide how much Redux to expose. There are three common patterns:

### Pattern A: Redux is internal (recommended)

- The core creates its own store internally.
- The React package reads from a context that wraps the client.
- Consumers do not need to know Redux exists.

Pros: Easiest for consumers, minimal coupling.
Cons: Less control for advanced integrations.

### Pattern B: Redux is external and injected

- The consumer creates the store and passes it in.
- The package provides reducers and actions.

Pros: Maximum control and extensibility.
Cons: Requires Redux knowledge and more setup.

### Pattern C: Dual mode (advanced)

- Default internal store, but allow injection.
- Document both modes and keep consistent state shapes.

Pros: Flexible.
Cons: More surface area and test burden.

> FUNDAMENTAL: Choose the simplest thing that fits the majority.
> 
> For a reusable package, the simplest integration wins. Hide Redux by default and only expose it if needed.

## Protocol and data modeling

### Generated protobuf types

The current app includes generated protobuf files under `src/sem/pb/**`.

Options:

1. **Bundle protobuf output into webchat-core**.
   - Pros: single package, consistent types.
   - Cons: increases bundle size and lint noise.
2. **Split protobuf types into a separate package**.
   - Pros: clearer dependency and versioning.
   - Cons: more moving parts.

Recommendation: keep protobuf output in the core package and exclude it from linting. Provide stable mapping functions so consumers never touch protobuf directly.

### Normalization of bigint fields

Normalize at the boundary:

```ts
function toNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'bigint') return Number(value);
  if (typeof value === 'string') {
    const num = Number(value);
    if (Number.isFinite(num)) return num;
  }
  return undefined;
}
```

> FUNDAMENTAL: Boundaries are where bugs are cheapest to fix.
> 
> If you normalize at the boundary, your internal logic can assume invariants. That reduces the cognitive load everywhere else.

## UI theming and customization

### Theme strategy

- Use CSS variables as the core theming API.
- Expose a small list of variables and document them clearly.
- Allow className overrides on high-level components.

Example:

```css
:root {
  --webchat-bg: #0b0c10;
  --webchat-panel: #14161d;
  --webchat-text: #e6e7eb;
  --webchat-accent: #7aa2ff;
}
```

### Slots and component overrides

If you want more customization, allow the integrator to pass render props:

```tsx
export type ChatWidgetProps = {
  Header?: React.ComponentType;
  Message?: React.ComponentType<MessageProps>;
  ToolCall?: React.ComponentType<ToolCallProps>;
};
```

> FUNDAMENTAL: Theming is not the same as customization.
> 
> Theming changes the look. Customization changes the structure. Keep both explicit so you can support each with minimal confusion.

## Packaging and build system

### Output format

Ship both ESM and CJS outputs with type definitions.

- `dist/index.js` (ESM)
- `dist/index.cjs` (CJS)
- `dist/index.d.ts` (types)

### Package.json exports

```json
{
  "name": "@org/webchat-react",
  "version": "0.1.0",
  "main": "./dist/index.cjs",
  "module": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "exports": {
    ".": {
      "import": "./dist/index.js",
      "require": "./dist/index.cjs",
      "types": "./dist/index.d.ts"
    }
  },
  "peerDependencies": {
    "react": ">=18",
    "react-dom": ">=18"
  }
}
```

> FUNDAMENTAL: Peer dependencies reduce version conflicts.
> 
> For React libraries, treat `react` and `react-dom` as peer dependencies so the consumer controls the version.

### Build tooling

Use a simple build tool such as `tsup` or `rollup` to generate ESM/CJS and types.

Example (tsup):

```json
{
  "scripts": {
    "build": "tsup src/index.ts --format esm,cjs --dts"
  }
}
```

## Embedding strategies

### React-native integration

Best for consumers already using React.

```tsx
import { ChatProvider, ChatWidget } from '@org/webchat-react';

function App() {
  return (
    <ChatProvider config={{ baseUrl: '/chat' }}>
      <ChatWidget />
    </ChatProvider>
  );
}
```

### Script tag embedding

For non-React pages:

```html
<div id="webchat"></div>
<script src="https://cdn.example.com/webchat-embed.js"></script>
<script>
  WebchatEmbed.mount('#webchat', { baseUrl: 'https://example.com/chat' });
</script>
```

### Web component (advanced)

Wrap the widget in a custom element (e.g., `<webchat-widget>`). This is the most portable but harder to implement correctly with React and CSS scoping.

## Error handling and observability

### Client-side logging

Provide a default logger but allow injection:

```ts
export type Logger = {
  info: (msg: string, ctx?: Record<string, unknown>) => void;
  warn: (msg: string, ctx?: Record<string, unknown>) => void;
  error: (msg: string, err?: unknown, ctx?: Record<string, unknown>) => void;
};
```

### Error boundary

Expose a standard `ErrorBoundary` or integrate it into `ChatWidget` by default.

### Error queue

If Redux is internal, you can optionally ship a debug panel that reads from an error queue. Keep it off by default if the consumer prefers minimal UI.

## Documentation strategy

Ship the following docs with the package:

- Quick start (3 minutes to embed).
- Configuration reference (every option, default, and effect).
- Theming guide (CSS variables and slot overrides).
- Advanced integration guide (custom transport, custom state).

> FUNDAMENTAL: Docs are part of the API.
> 
> If a feature is not documented, it does not exist for users.

## Migration plan from current webchat app

### Step 1: Identify modules to extract

- Core transport: `src/ws/wsManager.ts`.
- Protocol mapping: `src/sem/registry.ts`, `src/sem/timelineMapper.ts`.
- State slices: `src/store/*`.
- UI: `src/chat/ChatWidget.tsx`, `src/chat/Markdown.tsx`.

### Step 2: Create packages

In a monorepo, add:

```
/packages/webchat-core
/packages/webchat-react
/packages/webchat-embed (optional)
```

### Step 3: Move or mirror code

- Move core files to `webchat-core`.
- Move UI components to `webchat-react` and import from core.
- Keep the existing app as a consumer of these packages to validate integration.

### Step 4: Stabilize public API

- Expose only what you intend to support long-term.
- Freeze type names, config shapes, and event structures.

### Step 5: Cut a first version

- Version `0.1.0` is fine for early consumers.
- Document what is experimental.

## Testing strategy for the package

- Unit tests for pure functions (mapping, normalization).
- Integration tests for client workflows (connect, send, hydrate).
- Storybook for visual regression.
- Optional Playwright tests for an embedded example app.

> FUNDAMENTAL: Tests should validate contracts, not implementations.
> 
> If you design good interfaces, your tests can focus on behavior rather than internal details.

## Risks and tradeoffs

- **Bundle size**: protobuf and markdown libs can increase size; consider tree-shaking and optional imports.
- **Versioning**: once published, API changes are sticky; keep the first release small.
- **CSS leakage**: if you use global CSS, you risk collisions; prefer class scoping and CSS variables.
- **State ownership**: the more you expose, the harder it is to evolve the API.

## Concrete next steps (if implemented later)

1. Create `packages/webchat-core` and `packages/webchat-react` directories.
2. Add a build pipeline (tsup or rollup) and a root workspace config.
3. Extract protocol mapping and transport into core.
4. Wrap UI in `ChatProvider` and `ChatWidget` in webchat-react.
5. Update the existing webchat app to consume the new packages.
6. Write minimal README docs and publish a 0.1.0 prerelease.

## Summary

Packaging webchat is an exercise in finding stable boundaries: transport and protocol in core, UI in React, and optional embedding for non-React consumers. Keep the public API small, treat theming as a deliberate interface, and use peer dependencies to avoid dependency conflicts. This approach makes the webchat reusable without forcing every integrator to learn Redux or the internal protocol details.

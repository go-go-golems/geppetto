---
Title: go-go-os dynamic window menus and context menus design
Ticket: OS-01-ADD-MENUS
Status: active
Topics:
    - frontend
    - architecture
    - ui
    - menus
    - go-go-os
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-os/apps/inventory/src/App.tsx
      Note: Real app contribution/command wiring reference and focused-window chat debug behavior.
    - Path: go-go-os/packages/engine/src/chat/components/ChatConversationWindow.tsx
      Note: Current chat header profile/debug controls that should map into focused-window menus.
    - Path: go-go-os/packages/engine/src/chat/state/profileSlice.ts
      Note: Current global profile selection state; highlights need for per-conversation scoping.
    - Path: go-go-os/packages/engine/src/components/shell/windowing/WindowTitleBar.tsx
      Note: Title bar lacks right-click/context-menu hooks; needed for requested title-bar menus.
    - Path: go-go-os/packages/engine/src/components/shell/windowing/desktopContributions.ts
      Note: Defines current contribution contracts and merge semantics that must be extended compatibly.
    - Path: go-go-os/packages/engine/src/components/shell/windowing/useDesktopShellController.tsx
      Note: Controller currently composes only static/default menus and routes commands; primary insertion point for dynamic window menu and context-menu runtime.
    - Path: go-go-os/packages/engine/src/components/widgets/ContextMenu.tsx
      Note: Existing context menu primitive is string-based and shell-disconnected; base to evolve.
ExternalSources: []
Summary: Evidence-based architecture proposal for adding focused-window dynamic menu sections and widget-level right-click context menus in go-go-os, including API design, wiring model, migration plan, and test strategy.
LastUpdated: 2026-02-24T10:24:17-05:00
WhatFor: Use this to implement a unified menu system where active windows and widgets can expose command-driven menus without coupling menu rendering to app-specific shell code.
WhenToUse: Use when adding or reviewing DesktopShell menu/context-menu capabilities, focused-window command surfacing, or profile-registry menu wiring for chat windows.
---


# go-go-os dynamic window menus and context menus design

## Executive Summary

This ticket proposes a menu architecture upgrade for `go-go-os` that preserves the current strengths (simple contribution model, command handlers, deterministic composition) while adding two missing capabilities:

1. The currently focused window can expose dynamic, app-specific top menu sections and actions.
2. Any widget that opts in can expose right-click context menu actions, including title bars.

Current code has a global static menu model (`DesktopContribution.menus`) and command routing model (`DesktopContribution.commands`) that work well for app-wide commands. However, there is no runtime registration path for window-scoped menus, and there is no shell-integrated context-menu runtime. The only context menu primitive is a standalone widget used in Storybook demos.

The recommended solution is:

1. Keep command IDs as the primary wiring contract.
2. Add a menu runtime layer in `DesktopShell` that composes:
- static contributions,
- focused-window dynamic sections,
- optional app-provided overlays.
3. Add a context-menu runtime in `DesktopShell` that any window/widget can invoke via hook or target wrapper.
4. Extend command invocation context (source, windowId, widgetId, payload) without breaking existing command handlers.
5. Add optional per-conversation chat profile selection scope so focused chat windows can expose profile registry choices that truly apply to that conversation.

This design is incremental, backward-compatible for existing apps (`todo`, `crm`, `book-tracker-debug`, and current `inventory` contributions), and directly supports the user request for chat-window debug/profile actions and title-bar right-click menus.

## Problem Statement

The requested features are:

1. Active-window-specific menu entries (for example, focused chat window contributes conversation debug actions and profile switching from profile registry).
2. Right-click menu actions on title bars and other widgets that opt in.
3. A clear and clean action wiring model so widgets can expose menus without ad-hoc event plumbing.

Current architecture does not provide these features end-to-end.

## Scope and Deliverables

This document covers:

1. Current-state analysis of menu/window/action architecture in `go-go-os`.
2. Gap analysis against requested functionality.
3. Proposed API and runtime design.
4. Detailed implementation plan with file-level guidance.
5. Testing and migration strategy.

This ticket does not implement the code yet; it provides the exact implementation blueprint.

## Current State of Affairs (Evidence-Based)

### 1) Repo and Runtime Boundaries

`go-go-os` is a monorepo with:

1. shared engine in `packages/engine`
2. apps in `apps/*`
3. desktop shell in `packages/engine/src/components/shell/windowing/*`

Apps `todo`, `crm`, and `book-tracker-debug` use `<DesktopShell stack={STACK} />` directly, while `inventory` uses rich desktop contributions and app windows.

### 2) DesktopShell Composition

The runtime path is:

1. `DesktopShell` delegates to `useDesktopShellController` and renders `DesktopShellView`.
2. `useDesktopShellController` builds menus/icons, routes commands, maps window state, and wires drag/resize.
3. `DesktopShellView` renders menu bar, icon layer, window layer, and toast.

Evidence:

- controller entry and return contract: `packages/engine/src/components/shell/windowing/useDesktopShellController.tsx:101`
- shell view rendering tree: `packages/engine/src/components/shell/windowing/DesktopShellView.tsx:32`

### 3) Menu System Today

#### 3.1 Menu data model is static section -> item lists

`DesktopMenuSection` is currently only:

1. `id`
2. `label`
3. `items`

`DesktopMenuItem` has only `id`, `label`, `commandId`, `shortcut?`, `disabled?`.

Evidence: `packages/engine/src/components/shell/windowing/types.ts:1`

#### 3.2 Menu sources are only:

1. `DesktopShellProps.menus` override, or
2. composed `DesktopContribution.menus`, or
3. default generated menus.

No focused-window dynamic layer exists.

Evidence:

- menu source selection: `useDesktopShellController.tsx:174`
- contribution composition: `desktopContributions.ts:98`
- merge strategy: append by section id only (`desktopContributions.ts:52`)

#### 3.3 Menu rendering is disconnected from source context

`DesktopMenuBar` emits `(commandId, menuId)` but controller only accepts `commandId`, losing menu invocation metadata.

Evidence:

- menu bar callback shape: `DesktopMenuBar.tsx:13`
- controller callback shape: `useDesktopShellController.tsx:90`

### 4) Command Routing Today

Commands route in order:

1. contribution handlers (`routeContributionCommand`)
2. built-in router (`routeDesktopCommand`)
3. optional prop callback `onCommand(commandId)`

Evidence: `useDesktopShellController.tsx:345`

This is a good baseline but currently command context is minimal (no source/widget metadata).

### 5) Window Focus and Interaction Today

#### 5.1 Focus is tracked centrally

`windowing.desktop.focusedWindowId` is set by `openWindow` and `focusWindow` reducers.

Evidence: `packages/engine/src/desktop/core/state/windowingSlice.ts:27`

#### 5.2 Pointer behavior only supports left-button focus/drag flow

- Window surface only focuses on left click (`event.button !== 0` ignored).
- Title bar only handles pointer down for drag; no right-click behavior.

Evidence:

- `WindowSurface.tsx:47`
- `WindowTitleBar.tsx:23`
- interaction controller for move/resize starts from pointer down: `useWindowInteractionController.ts:93`

### 6) Context Menu Support Today

A generic widget exists (`ContextMenu`) but is not integrated into desktop shell runtime.

Current limitations:

1. entries are plain strings or separators only,
2. no command IDs,
3. no disabled/shortcut/check states,
4. no shell-level source context,
5. no integration with window focus or command router.

Evidence: `packages/engine/src/components/widgets/ContextMenu.tsx:4`

Only active use is a Storybook showcase with local `onContextMenu` state.

Evidence: `packages/engine/src/components/widgets/MacOS1Showcase.stories.tsx:69`

### 7) Inventory App: Practical Menu Wiring Today

`apps/inventory` demonstrates the current best available model:

1. defines global menus in contribution `menus`,
2. defines command handlers in contribution `commands`,
3. manually derives focused chat conversation for debug actions by inspecting `windowing.windows` and `focusedWindowId`.

Evidence:

- contributions start: `apps/inventory/src/App.tsx:472`
- menu entries: `App.tsx:518`
- command handlers: `App.tsx:567`
- focus-aware conversation resolver: `App.tsx:136`

This proves the command model works, but also shows focus-specific behavior is hand-coded in command handlers rather than in menu composition.

### 8) Profile Registry Integration Today

`ChatConversationWindow` already has a profile selector in header actions (`enableProfileSelector`, `profileRegistry`).

Evidence: `packages/engine/src/chat/components/ChatConversationWindow.tsx:234`

However, profile state is globally shared across all conversations:

1. `chatProfiles` stores a single `selectedProfile` and `selectedRegistry`.
2. `selectCurrentProfileSelection` is global.
3. `useConversation` reconnects each conversation using that global selection.

Evidence:

- profile state shape: `chat/state/profileSlice.ts:4`
- global selector: `chat/state/selectors.ts:164`
- useConversation dependencies include global selected profile/registry: `chat/runtime/useConversation.ts:68`

Implication: profile selection from one chat window affects all chat windows. For true focused-window profile menu behavior, we need scoped profile selection.

## Gap Analysis Against Requested Features

### Gap A: Active window cannot publish dynamic top-menu model

There is no runtime API for the focused window body to register menu sections/actions with shell.

Result:

- app-specific dynamic sections require global static menus + conditional handler logic,
- menu labels/availability cannot cleanly depend on focused window runtime state.

### Gap B: Right-click is not a first-class shell capability

No shell-level context menu runtime exists, and title bars do not handle right-click.

Result:

- browser native context menu appears unless each component manually prevents default,
- widget menus are ad hoc and disconnected from desktop command routing.

### Gap C: Action wiring lacks invocation context

Command handlers only receive `commandId` and minimal shell context.

Missing invocation metadata:

1. source (`menubar`, `context-menu`, etc.),
2. menu id,
3. window id,
4. widget id,
5. payload.

Result: hard to write generic handlers and analytics/debug traces.

### Gap D: Profile selection is global, not conversation/window scoped

Even with dynamic menu support, selecting profile for focused chat would currently update global state.

Result: behavior conflicts with the mental model of window-specific chat profile context.

## Design Goals

### Primary goals

1. Focused windows can register and update dynamic menu sections declaratively.
2. Any widget can expose a context menu through a shared, command-driven API.
3. Command routing remains centralized and deterministic.
4. Existing static contribution APIs remain valid.
5. UI behavior is predictable and easy to test.

### Secondary goals

1. Keyboard/a11y support can be improved without changing command contracts.
2. Observability hooks can record command invocation context.
3. Style remains theme-token based and consistent with existing parts/tokens.

### Non-goals for first implementation

1. Full macOS menubar semantics (nested submenus, full keyboard traversal) in v1.
2. Persisting menu UI transient state in Redux.
3. Storing non-serializable callback functions in Redux state.

## Proposed Solution

## 1) Introduce a Unified Menu Action Model

Keep command-id-first wiring and enrich entry metadata.

```ts
export interface DesktopActionItem {
  id: string;
  label: string;
  commandId: string;
  shortcut?: string;
  disabled?: boolean;
  checked?: boolean;
  danger?: boolean;
  payload?: unknown;
}

export interface DesktopActionSeparator {
  separator: true;
}

export type DesktopActionEntry = DesktopActionItem | DesktopActionSeparator;

export interface DesktopActionSection {
  id: string;
  label: string;
  items: DesktopActionEntry[];
  order?: number;
  merge?: 'append' | 'prepend' | 'replace';
}
```

Compatibility rule:

- existing `DesktopMenuItem`/`DesktopMenuSection` are treated as `DesktopActionItem`/`DesktopActionSection` with defaults.

## 2) Extend Command Invocation Context

Introduce invocation envelope while preserving old command handler signature compatibility.

```ts
export interface DesktopCommandInvocation {
  commandId: string;
  source: 'menubar' | 'context-menu' | 'icon' | 'shortcut' | 'programmatic';
  menuId?: string;
  windowId?: string;
  widgetId?: string;
  payload?: unknown;
}

export interface DesktopCommandContext {
  dispatch: (action: unknown) => unknown;
  getState?: () => unknown;
  focusedWindowId: string | null;
  openCardWindow: (cardId: string, options?: { dedupe?: boolean }) => void;
  closeWindow: (windowId: string) => void;
  invocation?: DesktopCommandInvocation;
}
```

Compatibility strategy:

1. existing handlers still receive `commandId` and `ctx`.
2. new code can inspect `ctx.invocation` when needed.

## 3) Add Focused-Window Dynamic Menu Registration

Add a runtime registry (React state/ref in shell controller, not Redux) keyed by `windowId`.

```ts
export interface WindowMenuRegistration {
  windowId: string;
  sections: DesktopActionSection[];
}

export interface DesktopMenuRuntime {
  registerWindowMenus: (windowId: string, sections: DesktopActionSection[]) => () => void;
  updateWindowMenus: (windowId: string, sections: DesktopActionSection[]) => void;
}
```

### Composition algorithm

Effective top menu =

1. base static sections (default or contributions/props),
2. focused window sections (if any),
3. optional app-level overlays.

Merge rule per section id:

1. if no collision -> insert by order,
2. collision + `replace` -> replace item list,
3. collision + `prepend`/`append` -> merge deterministically.

This cleanly supports:

- focused chat window exposing `Chat` or `Profile` sections,
- focused code editor exposing `Edit` actions,
- focused confirm window exposing request-specific actions.

## 4) Add Shell-Level Context Menu Runtime

Add `contextMenuState` to shell controller local state.

```ts
export interface DesktopContextMenuState {
  open: boolean;
  x: number;
  y: number;
  sourceWindowId?: string;
  sourceWidgetId?: string;
  items: DesktopActionEntry[];
}
```

### Public APIs for widgets/windows

Two ergonomics options can coexist:

1. Hook-first:

```ts
const { openContextMenu } = useDesktopMenus();

onContextMenu={(e) => {
  e.preventDefault();
  openContextMenu({
    x: e.clientX,
    y: e.clientY,
    windowId,
    widgetId: 'chat.timeline',
    items: [...]
  });
}}
```

2. Wrapper component:

```tsx
<ContextMenuTarget
  windowId={windowId}
  widgetId="chat.timeline"
  items={...}
>
  <ChatTimeline />
</ContextMenuTarget>
```

The shell overlay renders a context menu component that dispatches commands through the same command router used by top menus.

## 5) Title Bar Right-Click Integration

Enhance `WindowTitleBar` and `WindowSurface`:

1. add `onContextMenu` callback plumbing,
2. right-click focuses target window (if not focused),
3. open title-bar context menu with default actions and dynamic window actions.

Default title-bar context menu baseline:

1. `Close Window` -> `window.close-focused` or `window.close:{id}`
2. separator
3. `Tile Windows`
4. `Cascade Windows`
5. app/window-specific section items

## 6) Profile Registry Menu for Focused Chat Window

For chat windows in inventory:

1. when focused and profile data loaded, window registers sections:
- `Chat` (debug actions: Events, Timeline, Copy conv id)
- `Profile` (profiles from registry)
2. selecting profile dispatches command with payload `{ convId, profile, registry }`.

To make this truly per-window/per-conversation:

- introduce scoped profile selection in chat state.

## 7) Profile Selection Scoping (Recommended Companion Change)

Current single global selection should evolve to scoped model.

```ts
export interface ChatProfileSelectionScope {
  profile: string | null;
  registry: string | null;
}

export interface ChatProfilesState {
  availableProfilesByRegistry: Record<string, ChatProfileListItem[]>;
  selectionByScope: Record<string, ChatProfileSelectionScope>; // scope key: conv:<id> or global
  loadingByScope: Record<string, boolean>;
  errorByScope: Record<string, string | null>;
}
```

Hook signatures evolve to include scope:

```ts
useCurrentProfile(scopeKey?: string)
useSetProfile(scopeKey?: string)
useProfiles(basePrefix?: string, registry?: string, options?: { enabled?: boolean; scopeKey?: string })
useConversation(convId, basePrefix?, options?: { profileScopeKey?: string })
```

For chat windows:

- scope key = `conv:${convId}`.

This avoids cross-window profile coupling.

## Proposed API Reference (Concrete)

## New/updated desktop-react exports

```ts
// packages/engine/src/desktop/react/index.ts
export interface DesktopCommandInvocation { ... }
export interface DesktopActionItem { ... }
export interface DesktopActionSection { ... }

export interface UseDesktopMenusApi {
  registerWindowMenus: (windowId: string, sections: DesktopActionSection[]) => () => void;
  updateWindowMenus: (windowId: string, sections: DesktopActionSection[]) => void;
  openContextMenu: (args: {
    x: number;
    y: number;
    items: DesktopActionEntry[];
    windowId?: string;
    widgetId?: string;
  }) => void;
  closeContextMenu: () => void;
}

export function useDesktopMenus(): UseDesktopMenusApi;
```

## DesktopShell props compatibility

Keep existing props. Add optional richer callback:

```ts
onCommand?: (commandId: string) => void; // existing
onCommandInvocation?: (invocation: DesktopCommandInvocation) => void; // new optional
```

If only `onCommand` is provided, behavior remains unchanged.

## Widget API pattern

Widgets can expose menu items through helper hook:

```ts
export function useWidgetContextMenu(args: {
  windowId?: string;
  widgetId: string;
  items: DesktopActionEntry[] | (() => DesktopActionEntry[]);
}): {
  onContextMenu: (e: React.MouseEvent) => void;
};
```

## Pseudocode for Key Flows

### A) Effective Menubar Computation

```ts
function computeEffectiveSections(baseSections, focusedWindowId, windowMenuRegistry) {
  const focusedSections = focusedWindowId
    ? windowMenuRegistry.get(focusedWindowId) ?? []
    : [];

  return mergeSections(baseSections, focusedSections);
}
```

### B) Command Dispatch Entry Point

```ts
function dispatchDesktopCommand(invocation: DesktopCommandInvocation) {
  const commandId = invocation.commandId;

  const contributionHandled = routeContributionCommand(commandId, handlers, {
    ...ctx,
    invocation,
  });
  if (contributionHandled) return;

  const builtinHandled = routeDesktopCommand(commandId, builtinCtx);
  if (builtinHandled) return;

  onCommandInvocation?.(invocation);
  onCommand?.(commandId);
}
```

### C) Title Bar Right-Click

```ts
function onTitleBarContextMenu(event, windowId) {
  event.preventDefault();
  focusWindow(windowId);

  const defaultItems = [
    { id: 'close', label: 'Close Window', commandId: 'window.close-focused' },
    { separator: true },
    { id: 'tile', label: 'Tile Windows', commandId: 'window.tile' },
    { id: 'cascade', label: 'Cascade Windows', commandId: 'window.cascade' },
  ];

  const dynamicItems = getDynamicTitleItems(windowId);

  openContextMenu({
    x: event.clientX,
    y: event.clientY,
    windowId,
    widgetId: 'window.title-bar',
    items: [...defaultItems, ...dynamicItems],
  });
}
```

### D) Chat Window Registers Focused Dynamic Menus

```ts
function InventoryChatAssistantWindow({ convId, windowId }) {
  const profiles = useProfiles('', 'default', { scopeKey: `conv:${convId}` }).profiles;

  useEffect(() => {
    return registerWindowMenus(windowId, [
      {
        id: 'chat',
        label: 'Chat',
        items: [
          { id: 'events', label: 'Open Event Viewer', commandId: 'debug.event-viewer', payload: { convId } },
          { id: 'timeline', label: 'Open Timeline Debug', commandId: 'debug.timeline-debug', payload: { convId } },
        ],
      },
      {
        id: 'profile',
        label: 'Profile',
        items: profiles.map((p) => ({
          id: `profile:${p.slug}`,
          label: p.display_name ?? p.slug,
          checked: currentProfile === p.slug,
          commandId: 'chat.profile.select',
          payload: { convId, profile: p.slug, registry: 'default' },
        })),
      },
    ]);
  }, [convId, profiles, windowId]);
}
```

## Detailed Implementation Plan

## Phase 0: Baseline and Contracts

1. Add/adjust types for `DesktopActionEntry`, `DesktopCommandInvocation`.
2. Preserve old menu types as aliases or supersets.
3. Add unit tests for type-level/merge-level behavior.

Files:

- `packages/engine/src/components/shell/windowing/types.ts`
- `packages/engine/src/components/shell/windowing/desktopContributions.ts`
- `packages/engine/src/components/shell/windowing/desktopContributions.test.ts`

## Phase 1: Shell Runtime for Dynamic Menus and Context Menus

1. Create runtime registry/hooks file (new): `desktopMenuRuntime.tsx`.
2. Extend `useDesktopShellController` to:
- hold window menu registry,
- compute effective sections,
- hold context menu state,
- route context menu commands using shared dispatch path.
3. Extend `DesktopShellControllerResult` with context menu props/state.
4. Update `DesktopShellView` to render context menu overlay.

Files:

- `packages/engine/src/components/shell/windowing/useDesktopShellController.tsx`
- `packages/engine/src/components/shell/windowing/DesktopShellView.tsx`
- `packages/engine/src/components/shell/windowing/DesktopShell.tsx`
- `packages/engine/src/components/shell/windowing/index.ts`
- `packages/engine/src/desktop/react/index.ts`

## Phase 2: Window/Title Bar Right-Click

1. Add `onContextMenu` support to `WindowTitleBar` and `WindowSurface`.
2. Wire right-click focus + menu opening logic in controller.
3. Ensure drag only starts on left button remains unchanged.

Files:

- `packages/engine/src/components/shell/windowing/WindowTitleBar.tsx`
- `packages/engine/src/components/shell/windowing/WindowSurface.tsx`
- `packages/engine/src/components/shell/windowing/WindowLayer.tsx`
- `packages/engine/src/components/shell/windowing/WindowTitleBar.test.ts`

## Phase 3: ContextMenu Component Upgrade

1. Evolve `ContextMenu` from string-only entries to action entries.
2. Keep backward compatibility with string entries for existing stories.
3. Add disabled/check/shortcut rendering states.
4. Add keyboard close (`Escape`) and optional focus trap improvements.

Files:

- `packages/engine/src/components/widgets/ContextMenu.tsx`
- `packages/engine/src/components/widgets/ContextMenu.stories.tsx`
- `packages/engine/src/components/widgets/index.ts`

## Phase 4: Inventory App Adoption

1. Pass `windowId` through `renderAppWindow` chat branch.
2. In chat app window component, register focused dynamic menu sections.
3. Wire context menu actions for title bar and optional timeline widgets.

Files:

- `apps/inventory/src/App.tsx`
- potentially `packages/engine/src/chat/components/ChatConversationWindow.tsx` (if adding optional menu registration prop)

## Phase 5: Profile Selection Scope (Companion)

1. Refactor chat profile slice to scoped selections.
2. Update selectors/hooks to accept scope key.
3. Update useConversation to consume scoped profile.
4. Ensure inventory chat uses `conv:<id>` scope.

Files:

- `packages/engine/src/chat/state/profileSlice.ts`
- `packages/engine/src/chat/state/selectors.ts`
- `packages/engine/src/chat/runtime/useCurrentProfile.ts`
- `packages/engine/src/chat/runtime/useSetProfile.ts`
- `packages/engine/src/chat/runtime/useProfiles.ts`
- `packages/engine/src/chat/runtime/useConversation.ts`
- `apps/inventory/src/app/store.ts` (if state shape changes)

## Phase 6: Storybook + Docs + Regression Tests

1. Add new stories for:
- focused dynamic menubar
- title-bar context menu
- widget context menu targets
2. Add tests for merge behavior and command invocation metadata.
3. Update docs in engine docs on menu contribution/runtime APIs.

Files:

- `packages/engine/src/components/shell/windowing/DesktopShell.stories.tsx`
- `packages/engine/src/components/shell/windowing/DesktopPrimitives.stories.tsx`
- `packages/engine/src/components/widgets/MacOS1Showcase.stories.tsx`
- `packages/engine/docs/theming-and-widget-playbook.md`

## Testing Strategy

### Unit tests

1. section merge precedence (`append/prepend/replace`) and deterministic order.
2. context menu invocation -> command router with invocation metadata.
3. right-click title bar opens context menu and focuses target window.
4. legacy command handlers still operate with new invocation path.

### Integration tests (engine)

1. open multiple windows, switch focus, verify menubar sections update.
2. right-click title bar in focused/unfocused window.
3. right-click widget target dispatches expected command payload.
4. command fallthrough to `onCommand` remains intact.

### App-level tests (inventory)

1. focused chat window shows chat/profile menu sections.
2. profile selection command updates scoped conversation profile.
3. debug commands target focused conversation.

### Manual QA checklist

1. `todo`, `crm`, and `book-tracker-debug` unaffected (no dynamic menus required).
2. menubar still works with keyboard open/escape.
3. browser native context menu suppressed where custom context menu is expected.
4. mobile/touch fallback behavior still safe (no right-click assumptions).

## Risks and Mitigations

### Risk 1: API churn in `DesktopShellProps` and contribution contracts

Mitigation:

1. additive API design,
2. keep old `onCommand(commandId)` path,
3. keep existing section/item shape valid.

### Risk 2: Dynamic menu state causing unnecessary rerenders

Mitigation:

1. keep registry in refs + memoized signatures,
2. recompute effective sections only on focused window/menu-registry change.

### Risk 3: Context menu callback closures becoming unstable

Mitigation:

1. prefer command IDs + payload over inline callbacks,
2. keep ephemeral UI state local to controller, not Redux.

### Risk 4: Profile scope migration complexity

Mitigation:

1. do profile scope in dedicated phase,
2. provide compatibility fallback to global scope if scope key omitted,
3. add explicit migration tests.

## Alternatives Considered

### Alternative A: Keep static menus and only add more conditional handlers

Rejected because:

1. menubar UI still cannot reflect focused-window semantics cleanly,
2. logic remains hidden in command handlers.

### Alternative B: Store full menu providers/functions in Redux

Rejected because:

1. non-serializable functions in Redux are brittle,
2. unnecessary global state churn for transient UI.

### Alternative C: Build separate context menu routing system

Rejected because:

1. duplicates command wiring,
2. increases conceptual overhead for widget authors.

### Alternative D: Per-app ad hoc right-click handling only

Rejected because:

1. inconsistent UX across widgets/apps,
2. no reusable shell contract.

## Open Questions

1. Should dynamic focused-window sections merge into existing section IDs (for example `debug`) by default, or should they default to isolated section IDs?
2. Do we want checked/radio group semantics in v1, or defer to v2?
3. Should title-bar context menu include built-ins by default for dialog windows?
4. For profile scoping, should profile list caches be registry-wide shared with scoped selections, or fully scope-local?
5. Do we add keyboard shortcuts execution globally in this ticket, or keep shortcut labels display-only for now?

## Recommended Sequence for Implementation

1. Implement shell dynamic menu/context runtime with compatibility path first.
2. Add title-bar right-click integration.
3. Adopt in `inventory` chat windows.
4. Add scoped profile selection.
5. Expand stories/tests/docs.

This sequence minimizes risk while delivering user-visible value early.

## References

### Core shell and menu architecture

- `go-go-os/packages/engine/src/components/shell/windowing/useDesktopShellController.tsx`
- `go-go-os/packages/engine/src/components/shell/windowing/DesktopShellView.tsx`
- `go-go-os/packages/engine/src/components/shell/windowing/desktopContributions.ts`
- `go-go-os/packages/engine/src/components/shell/windowing/DesktopMenuBar.tsx`
- `go-go-os/packages/engine/src/components/shell/windowing/desktopCommandRouter.ts`
- `go-go-os/packages/engine/src/components/shell/windowing/types.ts`
- `go-go-os/packages/engine/src/components/shell/windowing/desktopShellTypes.ts`

### Window interactions and focus state

- `go-go-os/packages/engine/src/components/shell/windowing/WindowSurface.tsx`
- `go-go-os/packages/engine/src/components/shell/windowing/WindowTitleBar.tsx`
- `go-go-os/packages/engine/src/components/shell/windowing/useWindowInteractionController.ts`
- `go-go-os/packages/engine/src/desktop/core/state/windowingSlice.ts`

### Context menu primitives and theming

- `go-go-os/packages/engine/src/components/widgets/ContextMenu.tsx`
- `go-go-os/packages/engine/src/components/widgets/MacOS1Showcase.stories.tsx`
- `go-go-os/packages/engine/src/theme/desktop/primitives.css`
- `go-go-os/packages/engine/src/theme/desktop/tokens.css`

### Inventory app integration and profile flow

- `go-go-os/apps/inventory/src/App.tsx`
- `go-go-os/apps/inventory/src/app/store.ts`
- `go-go-os/packages/engine/src/chat/components/ChatConversationWindow.tsx`
- `go-go-os/packages/engine/src/chat/state/profileSlice.ts`
- `go-go-os/packages/engine/src/chat/state/selectors.ts`
- `go-go-os/packages/engine/src/chat/runtime/useProfiles.ts`
- `go-go-os/packages/engine/src/chat/runtime/useConversation.ts`


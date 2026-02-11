---
Title: 'Reusable Webchat: Modular + Themable Architecture Plan'
Ticket: PI-006-REUSABLE-WEBCHAT
Status: active
Topics:
    - frontend
    - webchat
    - refactor
    - design
    - css
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.stories.tsx
      Note: Storybook entrypoint for theming demos and regression checks
    - Path: pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx
      Note: Current monolithic UI structure and class usage to be modularized
    - Path: pinocchio/cmd/web-chat/web/src/chat/Markdown.tsx
      Note: Markdown rendering and toolbar styles that need data-part mapping
    - Path: pinocchio/cmd/web-chat/web/src/chat/chat.css
      Note: Current global styling and token mapping to new theme contract
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Config assumptions (basePrefix/WS URL) to expose for reuse
ExternalSources: []
Summary: Plan to refactor the current Pinocchio webchat UI into a reusable, modular React package with a stable CSS theming contract (tokens + data-part selectors) and optional default theme CSS.
LastUpdated: 2026-02-03T02:20:00-05:00
WhatFor: Provide a detailed implementation plan for modularizing and theming the existing webchat without changing its current behavior.
WhenToUse: Use when implementing PI-006, packaging the webchat UI, or reviewing theming/styling decisions.
---


# Reusable Webchat: Modular + Themable Architecture Plan

## Executive Summary

We will refactor the current Pinocchio webchat UI into a reusable, themeable React package while preserving its existing behavior. The plan introduces a **small public styling contract** (CSS tokens + stable `data-part` selectors), a **default theme** with low-specificity styles, and a **modular component API** (slot overrides + renderer map) so the UI can be embedded in other apps without copying CSS or rewriting components.

The refactor keeps the existing Redux + WS + SEM pipeline intact and focuses on: (1) restructuring UI components into a reusable module, (2) replacing class-based global styling with a stable theming contract, and (3) providing packaging hooks (exports + CSS bundles) to consume the webchat as a drop-in widget.

## Problem Statement

The current webchat UI is:
- **Monolithic**: `ChatWidget.tsx` renders all layout + card variants in one file.
- **Hard to theme**: Styling is a single global `chat.css` with `:root` tokens and class selectors that cannot be safely overridden by consumers.
- **Not reusable**: It assumes a specific app layout and global CSS, and it is not packaged as a library or themable component.

We need a modular, reusable, themable version of the same UI **without breaking existing behavior**.

## Proposed Solution

### 1) Adopt a small, stable styling contract

We will use the “Combo A” CSS packaging approach from the imported guidance:

**Public styling surfaces**
- **Tokens**: CSS variables prefixed with `--pwchat-*`.
- **Stable selectors**: `data-pwchat` root marker + `data-part` for main regions + minimal state attributes.

**Root markers**
- `data-pwchat=""` on the root.
- `data-part="root"` on the root.
- `data-theme="default"` only when not `unstyled`.

**Stable parts (public API)**
- Layout: `root`, `header`, `timeline`, `composer`, `statusbar`.
- Timeline: `turn`, `bubble`, `content`.
- Composer: `composer-input`, `composer-actions`, `send-button`.

**Stable state attributes**
- `data-state="idle|streaming"` (on `turn` or `timeline`).
- `data-state="connected|connecting|disconnected|error"` (on `statusbar` or root).
- `data-disabled` (on buttons/inputs).
- `data-role="user|assistant|tool|system"` (on `turn`).

Everything else remains internal and can change without breaking consumers.

### 2) Create a default theme CSS (optional, low specificity)

We will ship a default theme CSS that:
- Sets token values under `:where([data-pwchat][data-theme="default"])`.
- Applies layout styles under `:where([data-pwchat] [data-part="..."])`.
- Uses `:where()` to keep specificity low so consumers can override easily.

Consumers can opt out via `unstyled` and apply their own CSS.

### 3) Modularize the UI into composable components

Refactor `ChatWidget.tsx` into smaller components with a **renderer map** and **slot overrides**:

- `ChatWidget` (root composition + wiring)
- `ChatHeader`
- `ChatStatusbar`
- `ChatTimeline`
- `ChatComposer`
- Timeline cards: `MessageCard`, `ToolCallCard`, `ToolResultCard`, `LogCard`, `ThinkingModeCard`, `PlanningCard`, `GenericCard`

Expose a `renderers` map for timeline entity kinds and a `components` map for slot overrides (header, statusbar, composer, etc.). This makes the UI reusable without forking the source.

### 4) Provide a small, stable public API

Proposed `ChatWidget` props:

```ts
type ThemeVars = Partial<Record<`--pwchat-${string}`, string>>;

type ChatWidgetComponents = {
  Header: React.ComponentType<HeaderSlotProps>;
  Statusbar: React.ComponentType<StatusSlotProps>;
  Composer: React.ComponentType<ComposerSlotProps>;
};

type ChatWidgetRenderers = Record<string, React.ComponentType<{ e: RenderEntity }>>;

type ChatWidgetProps = {
  unstyled?: boolean;
  theme?: 'default' | string;
  themeVars?: ThemeVars;
  className?: string;
  rootProps?: React.HTMLAttributes<HTMLDivElement>;
  components?: Partial<ChatWidgetComponents>;
  renderers?: Partial<ChatWidgetRenderers>;
  partProps?: Partial<Record<ChatPart, React.HTMLAttributes<HTMLElement>>>;
};
```

The **default behavior** will mirror the current UI with no required props.

### 5) Package the UI for reuse

Within `cmd/web-chat/web`, create a `webchat` module that exports the widget and styles:

- `src/webchat/index.ts` (exports `ChatWidget`, types, helpers)
- `src/webchat/styles/webchat.css` (structure + parts)
- `src/webchat/styles/theme-default.css` (tokens)

This keeps the existing Vite app working while enabling an eventual library build/export.

## Design Decisions

1. **Tokens + data-part selectors** are the only stable styling surface.
   - Rationale: Keeps the API small, makes overrides easy, and avoids coupling to internal class names.

2. **Default theme is optional and low-specificity** using `:where()`.
   - Rationale: Consumers can override without specificity battles, and can opt out entirely.

3. **Slot overrides + renderer map** instead of prop explosion.
   - Rationale: Keeps the API small while enabling customization of header/composer and timeline cards.

4. **No behavior changes in the first pass.**
   - Rationale: The goal is structural/theming refactor, not a redesign.

## Alternatives Considered

- **CSS Modules for everything**: rejected because it hides selectors from consumers and complicates theming.
- **Styled-components/emotion**: rejected to avoid runtime CSS-in-JS overhead and to keep the bundle simple.
- **Hard-coded theme props** (e.g., `primaryColor`, `borderRadius`) instead of tokens: rejected because it leads to an unbounded, hard-to-support API surface.

## Implementation Plan

### Phase 0: Baseline mapping (no code changes)
1. Inventory current DOM parts and classes in `src/chat/ChatWidget.tsx` and `src/chat/chat.css`.\n+   - Map classes → new parts: root, header, timeline, composer, statusbar, turn, bubble, content.\n+2. Define the v1 token set (`--pwchat-*`) and map old variables (`--bg`, `--panel`, `--border`, etc.) to new names.\n+3. Decide which state attributes are public (`data-state`, `data-role`, `data-disabled`).\n+
### Phase 1: Styling contract + CSS scaffolding
4. Create `src/webchat/styles/theme-default.css` with **token values** under:\n+   - `:where([data-pwchat][data-theme=\"default\"]) { ... }`.\n+5. Create `src/webchat/styles/webchat.css` for **layout + part styles** under:\n+   - `:where([data-pwchat] [data-part=\"...\"]) { ... }`.\n+6. Keep layout and visuals equivalent to current UI (dark theme, pills, cards, composer) and avoid hard-coded colors outside tokens.\n+
### Phase 2: Module structure + component refactor
7. Add `src/webchat/` module:\n+   - `src/webchat/index.ts` (exports + types).\n+   - `src/webchat/components/*` for modular UI pieces.\n+   - `src/webchat/styles/*` for CSS.\n+8. Split `ChatWidget` into components:\n+   - `ChatWidget` (root + wiring)\n+   - `ChatHeader` + `ChatStatusbar`\n+   - `ChatTimeline` + `ChatComposer`\n+   - Cards: `MessageCard`, `ToolCallCard`, `ToolResultCard`, `LogCard`, `ThinkingModeCard`, `PlanningCard`, `GenericCard`.\n+9. Replace class names with `data-part` attributes in all rendered nodes.\n+10. Replace inline styles with token-backed CSS rules where possible (leave only the minimal inline styles that depend on runtime values).\n+11. Add `data-role` on turns and `data-state` on status/timeline for styling hooks.\n+
### Phase 3: Public API + theming hooks
12. Add `ChatWidgetProps`:\n+   - `unstyled?: boolean`\n+   - `theme?: string` (default \"default\")\n+   - `themeVars?: Record<\`--pwchat-${string}\`, string>`\n+   - `className?`, `rootProps?`, `partProps?`\n+13. Add customization props:\n+   - `components?: Partial<ChatWidgetComponents>` (header/statusbar/composer slots)\n+   - `renderers?: Partial<ChatWidgetRenderers>` (timeline entity override map).\n+14. Expose `RenderEntity` and slot prop types from `src/webchat/index.ts`.\n+
### Phase 4: App + Storybook integration
15. Update `src/App.tsx` to import from `src/webchat` module.\n+16. Update `src/chat/ChatWidget.stories.tsx` (or relocate to `src/webchat`) to add:\n+   - Default theme story\n+   - Unstyled + custom CSS story\n+   - ThemeVars override story\n+   - Custom renderer story (override one timeline entity kind)\n+17. Remove legacy `chat.css` usage once parity is verified; ensure new styles are imported from `src/webchat/styles`.\n+
### Phase 5: Validation
18. Run `npm run check` in `pinocchio/cmd/web-chat/web`.\n+19. Manual smoke: `npm run dev` (optional) and confirm WS/hydration still works.\n+20. Storybook build or visual smoke to confirm theming overrides.\n+21. Update diary + changelog and mark tasks done.\n*** End Patch"}}

## Open Questions

- Should we publish a light theme alongside default? (Can be done after Phase 3.)
- Do we need a separate package root (outside `cmd/web-chat/web`) for library distribution?
- Should `data-state` go on the root or on specific parts (statusbar vs timeline)?

## References

- Imported guidance: `sources/local/css-packaging-combo.md`
- Imported guidance: `sources/local/webchat-css-org.md`
- Existing UI: `pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx`
- Existing CSS: `pinocchio/cmd/web-chat/web/src/chat/chat.css`
- Markdown renderer: `pinocchio/cmd/web-chat/web/src/chat/Markdown.tsx`

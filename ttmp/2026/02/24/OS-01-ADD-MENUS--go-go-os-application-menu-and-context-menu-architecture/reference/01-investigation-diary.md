---
Title: Investigation diary
Ticket: OS-01-ADD-MENUS
Status: active
Topics:
    - frontend
    - architecture
    - ui
    - menus
    - go-go-os
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-os/apps/inventory/src/App.tsx
      Note: Primary integration target for app-specific menu and focused chat actions.
    - Path: go-go-os/packages/engine/src/chat/runtime/useConversation.ts
      Note: Evidence of global profile selection flow into websocket/session wiring.
    - Path: go-go-os/packages/engine/src/components/shell/windowing/useDesktopShellController.tsx
      Note: Primary investigation target for menu and command flow evidence.
    - Path: go-go-os/packages/engine/src/components/widgets/MacOS1Showcase.stories.tsx
      Note: Shows current ad-hoc context menu usage baseline.
    - Path: go-go-os/packages/engine/src/theme/desktop/primitives.css
      Note: Confirms context-menu styling primitives already available for shell integration.
ExternalSources: []
Summary: Chronological investigation log for OS-01-ADD-MENUS including commands, evidence, findings, design decisions, and delivery steps.
LastUpdated: 2026-02-24T10:24:17-05:00
WhatFor: Use this as a reproducible audit trail of how the menu/context-menu architecture analysis was performed and which files were used as evidence.
WhenToUse: Use when reviewing the design proposal, onboarding engineers, or continuing implementation planning in this ticket.
---


# Investigation diary

## Goal

Produce an exhaustive, implementation-ready architecture analysis for adding:

1. focused-window dynamic menu sections/actions,
2. right-click context menus for title bars and opt-in widgets,
3. clean command wiring for menu actions,

then store the analysis in the ticket and upload the deliverable to reMarkable.

## Context

The user requested a new docmgr ticket (`OS-01-ADD-MENUS`) and a deep analysis of `go-go-os/`, with intern-friendly detail, pseudocode, and API-level clarity.

Skills used in order:

1. `frontend-review-docmgr-remarkable` (primary workflow)
2. `docmgr` (ticket/document lifecycle + relate/changelog conventions)
3. `remarkable-upload` (bundle upload process)

## Chronological Log

## 2026-02-24 10:17 - Ticket Setup

### Commands

```bash
docmgr status --summary-only
docmgr ticket create-ticket --ticket OS-01-ADD-MENUS --title "go-go-os application menu and context-menu architecture" --topics frontend,architecture,ui,menus,go-go-os
docmgr doc add --ticket OS-01-ADD-MENUS --doc-type design-doc --title "go-go-os dynamic window menus and context menus design"
docmgr doc add --ticket OS-01-ADD-MENUS --doc-type reference --title "Investigation diary"
```

### Result

1. Ticket created under `geppetto/ttmp/2026/02/24/OS-01-ADD-MENUS--...`.
2. Primary design doc and diary doc created.

## 2026-02-24 10:18 - Repo and Surface Mapping

### Commands

```bash
pwd && ls -la
ls -la go-go-os
docmgr ticket list
find go-go-os -maxdepth 3 -type d
cat go-go-os/package.json
cat go-go-os/README.md
find go-go-os/packages -maxdepth 3 -type d
```

### Findings

1. `go-go-os` uses `packages/engine` as the shared desktop runtime and widgets package.
2. `apps/inventory` is the only app with substantial desktop contribution wiring.
3. Other apps (`todo`, `crm`, `book-tracker-debug`) use default `DesktopShell` configuration.

## 2026-02-24 10:19 - Menu/Window/Profile Evidence Sweep

### Commands

```bash
rg --files go-go-os/packages/engine/src
rg -n "menu|context|title bar|DesktopContribution|focus|profile" go-go-os/packages/engine/src go-go-os/apps/*/src -S
wc -l <key files>
```

### Findings

1. Menu system is in `components/shell/windowing/*`.
2. ContextMenu widget exists but is mostly isolated from shell runtime.
3. Profile registry integration exists in chat window header UI, not shell menus.

## 2026-02-24 10:20 - DesktopShell Architecture Read

### Commands

```bash
sed -n '1,260p' packages/engine/src/components/shell/windowing/{types.ts,desktopShellTypes.ts,desktopContributions.ts,desktopCommandRouter.ts}
sed -n '1,560p' packages/engine/src/components/shell/windowing/useDesktopShellController.tsx
sed -n '1,260p' packages/engine/src/components/shell/windowing/{DesktopShell.tsx,DesktopShellView.tsx,DesktopMenuBar.tsx,WindowLayer.tsx,WindowSurface.tsx,WindowTitleBar.tsx}
sed -n '1,320p' packages/engine/src/desktop/core/state/{types.ts,windowingSlice.ts,selectors.ts}
```

### Findings

1. Menu sections are static (`id`, `label`, `items`).
2. Contribution menus are merged globally by section id with append semantics.
3. No focused-window dynamic menu registration path exists.
4. Command routing is deterministic but context-light (`commandId` centric).
5. Window focus state exists and is reliable (`focusedWindowId`), making it a good anchor for dynamic menu composition.

## 2026-02-24 10:21 - Inventory App Integration Read

### Commands

```bash
sed -n '1,760p' apps/inventory/src/App.tsx
sed -n '1,360p' packages/engine/src/chat/components/ChatConversationWindow.tsx
sed -n '1,260p' packages/engine/src/chat/state/profileSlice.ts
sed -n '1,260p' packages/engine/src/chat/runtime/{useProfiles.ts,useConversation.ts,useCurrentProfile.ts,useSetProfile.ts}
sed -n '1,220p' apps/inventory/src/app/store.ts
```

### Findings

1. Inventory uses `DesktopContribution` menus/commands heavily.
2. Focused-window chat conversation resolution is currently done manually in command handlers.
3. Profile selector is in `ChatConversationWindow` header, not in menubar/context-menu.
4. Profile selection state is global (`selectedProfile` + `selectedRegistry` singletons), which is a mismatch for true per-window/per-conversation profile switching.

## 2026-02-24 10:22 - Context Menu and Theme Sweep

### Commands

```bash
sed -n '1,220p' packages/engine/src/components/widgets/ContextMenu.tsx
sed -n '1,280p' packages/engine/src/components/widgets/MacOS1Showcase.stories.tsx
rg -n "context-menu|windowing-menu|windowing-window-title-bar" packages/engine/src/theme/desktop/*.css -S
sed -n '1,330p' packages/engine/src/theme/desktop/shell.css
sed -n '340,470p' packages/engine/src/theme/desktop/primitives.css
sed -n '70,170p' packages/engine/src/theme/desktop/tokens.css
```

### Findings

1. The visual primitives for context menus exist (`data-part="context-menu"`, related tokens).
2. Shell title bar has no right-click behavior.
3. Existing ContextMenu data model uses plain strings; no command IDs/payload metadata.

## 2026-02-24 10:23 - Line-Anchored Evidence Capture

### Commands

```bash
nl -ba <key files> | sed -n '<ranges>'
```

Captured line-anchored evidence for:

1. shell controller command and menu flow,
2. title bar/window pointer behavior,
3. context menu primitive limits,
4. inventory app contribution model,
5. global profile selection behavior.

This was used directly in the design doc’s current-state and gap-analysis sections.

## 2026-02-24 10:24 - Documentation Authoring

### Actions

1. Replaced design doc template with full architecture report:
- current-state map,
- gap analysis,
- API proposal,
- pseudocode,
- phase-by-phase implementation plan,
- test strategy,
- risk/alternatives/open questions.
2. Updated this diary with command trail and findings.

## Key Findings Summary

1. Current menu system is global and mostly static.
2. Focus state infrastructure is good and can power dynamic active-window menus.
3. Context menu UI exists but not shell-integrated and not command-aware.
4. Inventory app already demonstrates command routing extensibility and focus-aware debug behavior.
5. Profile selection is global and should be scoped to conversation to satisfy per-window profile menus cleanly.

## Tricky Points and Decision Rationale

### 1) Where to store dynamic menu/context runtime

Decision: keep runtime registries in `DesktopShell` controller local state/ref, not Redux.

Why:

1. menu/context UI state is transient and view-local,
2. avoids non-serializable function storage in Redux,
3. reduces global invalidation pressure.

### 2) How to wire actions cleanly

Decision: command-id-first model with optional payload + invocation context.

Why:

1. reuses existing contribution routing model,
2. keeps behavior deterministic,
3. avoids ad-hoc widget callback plumbing.

### 3) Profile menu semantics for focused chat windows

Decision: recommend scoped profile selections by conversation.

Why:

1. global selected profile conflicts with per-window expectations,
2. requested behavior implies conversation-local control.

## Quick Reference

### Most critical evidence files

1. `go-go-os/packages/engine/src/components/shell/windowing/useDesktopShellController.tsx`
2. `go-go-os/packages/engine/src/components/shell/windowing/desktopContributions.ts`
3. `go-go-os/packages/engine/src/components/shell/windowing/WindowTitleBar.tsx`
4. `go-go-os/packages/engine/src/components/widgets/ContextMenu.tsx`
5. `go-go-os/apps/inventory/src/App.tsx`
6. `go-go-os/packages/engine/src/chat/components/ChatConversationWindow.tsx`
7. `go-go-os/packages/engine/src/chat/state/profileSlice.ts`
8. `go-go-os/packages/engine/src/chat/runtime/useConversation.ts`

### Deliverable doc

- `design-doc/01-go-go-os-dynamic-window-menus-and-context-menus-design.md`

## Usage Examples

### Example 1: Continue implementation in next session

1. Read design doc sections in order:
- executive summary,
- current-state evidence,
- proposed API,
- implementation phases.
2. Start from Phase 0 and Phase 1 file list.
3. Keep backward compatibility by preserving old `onCommand(commandId)` and existing contribution types.

### Example 2: Intern onboarding

1. Start with this diary’s chronological log to understand why each conclusion exists.
2. Open line-anchored files and compare to the proposed APIs.
3. Implement a minimal vertical slice:
- title-bar right-click -> context menu -> command routing,
- focused-window dynamic menu section registration.

## Related

1. Design doc in this ticket (`design-doc/01-go-go-os-dynamic-window-menus-and-context-menus-design.md`)
2. Ticket tasks (`tasks.md`)
3. Ticket changelog (`changelog.md`)


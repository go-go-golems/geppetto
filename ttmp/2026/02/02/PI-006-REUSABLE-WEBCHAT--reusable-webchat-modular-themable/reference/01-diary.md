---
Title: Diary
Ticket: PI-006-REUSABLE-WEBCHAT
Status: active
Topics:
    - frontend
    - webchat
    - refactor
    - design
    - css
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Implementation diary for reusable, modular, themable webchat plan
LastUpdated: 2026-02-03T03:05:00-05:00
WhatFor: Track investigation steps and decisions while designing the reusable webchat architecture.
WhenToUse: Update after each research or design decision while planning modular webchat theming.
---

# Diary

## Goal

Maintain a detailed, step-by-step diary while designing a modular, reusable, themeable version of the current Pinocchio webchat UI, grounded in the existing TS codebase and the imported CSS packaging guidance.

## Step 1: Create ticket + import CSS guidance sources

I created the PI-006 docmgr ticket, added the design doc + diary, and imported the two CSS packaging guidance files from `/tmp`. The imports now live under the ticket `sources/local/` directory for traceability.

### Prompt Context

**User prompt (verbatim):**
"Create a new ticket PI-006-REUSABLE-WEBCHAT with docmgr. Then Import the files /tmp/css-packaging-combo.md and /tmp/webchat-css-org.md using `docmgr import file --file /tmp/.... --ticket 006-REUSABLE-WEBCHAT`, then read these two documents and design a detailed implementation plan on how to create a amodular reusable themable version of the webchat as it is now. Store that design document in the new ticket as well, and upload it to remarkable.

As you work and analyze all the existing TS code and make decisions on what styling to use how, keep a detailed frequent diary as you go, with frequent entries."

**Assistant interpretation:** Create a new PI-006 ticket, import the two guidance docs, read them, and then write a detailed architecture + implementation plan for a reusable/themable webchat. Keep a frequent diary while inspecting the existing TS code and making styling decisions, then upload the design doc to reMarkable.

**Inferred user intent:** Keep a focused, reviewable plan that preserves current functionality but makes the UI modular, themeable, and safe to embed elsewhere.

**Commit (code):** N/A (docs-only so far)

### What I did
- Created ticket `PI-006-REUSABLE-WEBCHAT` with design + diary docs.
- Imported `/tmp/css-packaging-combo.md` and `/tmp/webchat-css-org.md` into ticket sources.

### Why
- Establish a dedicated workspace and capture the CSS packaging guidance as inputs for the design plan.

### What worked
- Ticket and docs were created successfully.
- Source files imported into `sources/local/`.

### What didn't work
- The first import attempt using `--ticket 006-REUSABLE-WEBCHAT` failed because the ticket was created as `PI-006-REUSABLE-WEBCHAT`. The import succeeded after using the correct ticket ID.

### What I learned
- The ticket ID must match exactly; `PI-006-REUSABLE-WEBCHAT` is the canonical ID.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Check `geppetto/ttmp/2026/02/02/PI-006-REUSABLE-WEBCHAT--reusable-webchat-modular-themable/sources/local/` for imported guidance files.

### Technical details
- Commands:
  - `docmgr ticket create-ticket --ticket PI-006-REUSABLE-WEBCHAT ...`
  - `docmgr doc add --ticket PI-006-REUSABLE-WEBCHAT --doc-type design-doc ...`
  - `docmgr doc add --ticket PI-006-REUSABLE-WEBCHAT --doc-type reference --title "Diary"`
  - `docmgr import file --file /tmp/css-packaging-combo.md --ticket PI-006-REUSABLE-WEBCHAT`
  - `docmgr import file --file /tmp/webchat-css-org.md --ticket PI-006-REUSABLE-WEBCHAT`

## Step 2: Read CSS packaging guidance inputs

I read both imported guidance docs to extract the recommended theming contract (tokens + stable data-part selectors), the default-theme strategy (`:where()` low-specificity), and the minimal prop surface (`unstyled`, `themeVars`, slot overrides).

### What I did
- Read `sources/local/css-packaging-combo.md`.
- Read `sources/local/webchat-css-org.md`.
- Captured the key ideas to carry into the plan: token names, part names, state attributes, and default theme layout rules.

### Why
- The plan must adhere to the published styling contract to keep the API surface small and stable.

### What worked
- Both guidance docs are aligned and give concrete token + selector shapes that fit a webchat UI.

### What didn't work
- N/A

### What I learned
- The recommended contract is: CSS variables for theming + stable `data-part` selectors + one role attribute (`data-role`) for user/assistant bubble styling.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- Ensure the final token list stays small and does not leak internal component structure.

### What should be done in the future
- Apply the contract to the existing TS components and identify which DOM nodes should become stable parts.

### Code review instructions
- Review the imported guidance docs in `sources/local/`.

### Technical details
- Files read:
  - `.../sources/local/css-packaging-combo.md`
  - `.../sources/local/webchat-css-org.md`

## Step 3: Inspect current TS + CSS structure (what must stay compatible)

I walked through the existing webchat UI implementation to understand the DOM structure, class usage, inline styles, and the minimal set of parts that should become stable selectors.

### What I did
- Read `src/chat/ChatWidget.tsx` to map the layout (header, status, timeline, composer) and the timeline card variants.
- Read `src/chat/chat.css` to capture the current class-based styling and theme variables.
- Read `src/chat/Markdown.tsx` to see how markdown + code blocks render and where buttons/toolbar styles are reused.
- Read `src/ws/wsManager.ts` to understand current configuration assumptions (basePrefix, location-based WS URL).

### Why
- The new theming contract must preserve the current structure and interaction model, while moving from class names to stable data-part selectors and tokens.

### What worked
- The existing UI already separates the logical regions (header, main timeline, composer), which maps cleanly to a small set of public `data-part` selectors.

### What didn't work
- N/A

### What I learned
- Styling is currently centralized in a single `chat.css` file with global classes and root-level tokens (`--bg`, `--panel`, etc.).
- `ChatWidget` directly imports CSS and uses class names + inline styles rather than a themed contract.
- The timeline renders several card variants (message, tool_call, tool_result, log, thinking_mode, planning), which suggests the reusable API should include a renderer map/slot approach rather than hardcoding class names.

### What was tricky to build
- Identifying which DOM nodes should be stable parts without leaking every internal wrapper into the public contract.

### What warrants a second pair of eyes
- Validate the minimum public parts list (header, timeline, composer, turn/bubble/content, statusbar) is sufficient for expected customization.

### What should be done in the future
- Replace class names with `data-part` and `data-state` attributes and migrate inline styles into token-backed CSS rules.

### Code review instructions
- Review `pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx`.
- Review `pinocchio/cmd/web-chat/web/src/chat/chat.css`.
- Review `pinocchio/cmd/web-chat/web/src/chat/Markdown.tsx`.
- Review `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`.

### Technical details
- Files read:
  - `pinocchio/cmd/web-chat/web/src/chat/ChatWidget.tsx`
  - `pinocchio/cmd/web-chat/web/src/chat/chat.css`
  - `pinocchio/cmd/web-chat/web/src/chat/Markdown.tsx`
  - `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`

## Step 4: Define the theming contract + modularization strategy

Based on the guidance docs and the current TS structure, I drafted the concrete styling contract (tokens + parts + state attributes) and a modularization plan for the widget.

### What I decided
- **Tokens**: adopt the `--pwchat-*` token set (bg, fg, surface-1/2, border, accent, muted, typography, radius, spacing, key dimensions).
- **Stable selectors**: use `data-pwchat` + `data-part` with a small part list (root, header, timeline, composer, statusbar, turn, bubble, content, composer-input, composer-actions, send-button).
- **States**: `data-state` for connection + streaming, `data-role` for user/assistant/tool/system, `data-disabled` for controls.
- **Default theme**: ship a low-specificity theme using `:where([data-pwchat][data-theme="default"])` and `:where([data-pwchat] [data-part="..."])`.
- **Modularity**: split `ChatWidget` into layout + card components; expose a renderer map and slot overrides instead of a large prop surface.

### Why
- This matches the guidance docs and keeps the public contract small while still making the UI fully themeable and reusable.

### What worked
- The existing DOM structure maps cleanly onto the part list without inventing new wrappers.

### What didn't work
- N/A

### What I learned
- The current global CSS tokens (`--bg`, `--panel`, etc.) can be mapped 1:1 into `--pwchat-*` without changing visuals.

### What was tricky to build
- Balancing the need for customization against keeping the public selector list small.

### What warrants a second pair of eyes
- Validate the parts list before implementation so we do not commit to an overly granular contract.

### What should be done in the future
- Implement the new CSS contract and refactor `ChatWidget` to emit data attributes.

### Code review instructions
- Review the design doc `design-doc/01-reusable-webchat-modular-themable-architecture-plan.md`.

### Technical details
- Drafted the design doc and implementation plan based on current TS/CSS and imported guidance.

## Step 5: Publish the design plan to reMarkable

I uploaded the design document to the reMarkable device for review.

### What I did
- Ran a dry-run upload to confirm the PDF name and destination.
- Uploaded the generated PDF to `/ai/2026/02/03/PI-006-REUSABLE-WEBCHAT`.

### Why
- Provide a portable review artifact outside the repo.

### What worked
- The PDF upload completed successfully.

### What didn't work
- N/A

### What I learned
- The `remarquee upload md` command is sufficient for single-doc uploads with a specified remote directory.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- If you want the diary bundled too, we can re-run upload as a multi-file bundle.

### Technical details
- Commands:
  - `remarquee upload md --dry-run --remote-dir /ai/2026/02/03/PI-006-REUSABLE-WEBCHAT <design-doc>`
  - `remarquee upload md --remote-dir /ai/2026/02/03/PI-006-REUSABLE-WEBCHAT <design-doc>`

## Step 6: Implement modular webchat module + theming contract

I created the new `src/webchat/` module, split the UI into smaller components, and replaced class-based styling with the `data-pwchat`/`data-part` contract. I also added the tokenized default theme CSS and migrated the UI to the new styling surface.

### What I did
- Added `src/webchat/` with `ChatWidget`, `types`, `parts`, `cards`, and modular components (Header/Statusbar/Composer/Timeline).
- Added `src/webchat/styles/theme-default.css` (token values) and `src/webchat/styles/webchat.css` (layout + parts) with low-specificity selectors.
- Converted UI markup to `data-pwchat` + `data-part` + `data-state` + `data-role` attributes and mapped current styles into tokens.
- Introduced theming/customization props: `unstyled`, `theme`, `themeVars`, `partProps`, `components`, `renderers`.
- Moved the legacy `src/chat` folder into `web/legacy/chat` to prevent dual maintenance and avoid bundling old CSS.

### Why
- This establishes a reusable, themeable contract while keeping current behavior stable.

### What worked
- The modular layout and data-part selectors map cleanly to the existing UI structure.

### What didn't work
- N/A

### What I learned
- The current styles translate cleanly into a tokenized theme without visual regression.

### What was tricky to build
- Keeping the parts list small while still styling the internal card/pill/button elements.

### What warrants a second pair of eyes
- Confirm the part list is minimal enough and aligns with the intended public API.

### Code review instructions
- Start with `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`.
- Review the new theme + layout CSS in `pinocchio/cmd/web-chat/web/src/webchat/styles/`.
- Review component boundaries in `pinocchio/cmd/web-chat/web/src/webchat/components/`.

### Technical details
- Files added/updated:
  - `src/webchat/ChatWidget.tsx`
  - `src/webchat/types.ts`
  - `src/webchat/parts.ts`
  - `src/webchat/cards.tsx`
  - `src/webchat/components/*`
  - `src/webchat/styles/theme-default.css`
  - `src/webchat/styles/webchat.css`
  - `src/webchat/index.ts`
  - `src/webchat/ChatWidget.stories.tsx`
  - `src/App.tsx`

## Step 7: Validate TypeScript + linting

I ran the frontend checks to ensure the refactor compiles and passes lint.

### What I did
- Ran `npm run check` in `pinocchio/cmd/web-chat/web` (typecheck + biome lint).

### What worked
- Typecheck and lint passed after import ordering fixes and type corrections.

### What didn't work
- N/A

### What I learned
- The new module structure fits cleanly into the existing tooling setup.

### Technical details
- Command: `npm run check`

## Step 8: Commit modular webchat refactor

I staged the refactor and committed it in the `pinocchio` repo.

### What I did
- Staged the new `src/webchat` module, legacy moves, and `src/App.tsx` update.
- Committed with message `webchat: modular, themeable widget`.
- Let the pre-commit hook run `npm run check` (typecheck + biome lint).

### What worked
- The commit completed with the hook checks passing.

### What didn't work
- N/A

### Technical details
- Commit: `48e4416`
- Repo: `pinocchio`
- Hook command: `npm run check`

## Step 9: Storybook smoke-test

I built Storybook to smoke-test the new webchat module.

### What I did
- Ran `npm run build-storybook` in `pinocchio/cmd/web-chat/web`.

### What worked
- Storybook build completed successfully.

### What didn't work
- N/A (build emitted typical eval/chunk-size warnings).

### Technical details
- Command: `npm run build-storybook`

---

## 0) The public styling contract (what you promise)

You commit to exactly **two public styling surfaces**:

1. **Tokens (CSS variables)** — a finite list prefixed `--pwchat-*`
2. **Stable selectors** via:

   * root marker: `data-pwchat=""`
   * stable parts: `data-part="…"`
   * small set of state/variant attributes: `data-state`, `data-disabled`, plus one purposeful role attribute for turns/messages.

Everything else in the DOM remains internal and can change.

---

## 1) DOM structure contract

### Always render these on the root

* `data-pwchat=""`
* `data-part="root"`
* `data-theme="default"` **only when not `unstyled`** (more below)

### Stable “parts” (keep small)

For a chat widget, these are the parts I’d publish as stable:

**Layout**

* `root` — the widget container
* `header` — top bar (title, profile selector, actions)
* `timeline` — scrollable list area
* `composer` — input area container
* `statusbar` — connection / small status row (optional)

**Within timeline**

* `turn` — a single “row” in the timeline (user/assistant/tool)
* `bubble` — the visible bubble container within a turn
* `content` — the rendered content area (markdown, tool result UI, etc.)

**Within composer**

* `composer-input` — textarea/input
* `composer-actions` — button row (send, attachments, etc.)
* `send-button` — the send button itself

That’s ~11 parts. It’s enough to style almost everything without exposing wrapper soup.

### Minimal stable state/variant attributes

You need a *tiny* set that covers the common styling needs:

* `data-state="idle|streaming"` on `turn` or `timeline` (streaming indicator)
* `data-state="connected|connecting|disconnected"` on `statusbar` (or root)
* `data-disabled` on disabled controls (send button, input)
* `data-role="user|assistant|tool|system"` on `turn` (this is the one extra attribute that is worth making stable, because it’s how people style user vs assistant bubbles reliably)

Example stable selectors consumers can rely on:

```css
.myChat [data-part="turn"][data-role="assistant"] [data-part="bubble"] { … }
.myChat [data-part="send-button"][data-disabled] { opacity: .5; }
.myChat [data-part="statusbar"][data-state="disconnected"] { … }
```

---

## 2) Tokens (CSS variables): define once, keep boring

Prefix everything with `--pwchat-`.

### Core colors

* `--pwchat-bg`
* `--pwchat-fg`
* `--pwchat-surface-1`
* `--pwchat-surface-2`
* `--pwchat-border`
* `--pwchat-accent`
* `--pwchat-accent-fg`
* `--pwchat-muted`
* `--pwchat-muted-fg`

### Typography

* `--pwchat-font-sans`
* `--pwchat-font-mono`
* `--pwchat-font-size`
* `--pwchat-line-height`

### Shape & spacing

* `--pwchat-radius`
* `--pwchat-gap`
* `--pwchat-padding`

### A *couple* component dimensions (only if needed)

* `--pwchat-header-h`
* `--pwchat-composer-min-h`

That’s a solid v1. (You can always add tokens later; removing/renaming is breaking.)

---

## 3) Default theme CSS (optional, low specificity, scoped)

### Theme marker

* Root always has `data-pwchat`.
* Default theme is activated by `data-theme="default"` **unless `unstyled`**.

### Default theme tokens (low specificity via `:where()`)

```css
/* theme-default.css */
:where([data-pwchat][data-theme="default"]) {
  --pwchat-bg: #0b0f19;
  --pwchat-fg: #e6e6e6;
  --pwchat-surface-1: rgba(255,255,255,.04);
  --pwchat-surface-2: rgba(255,255,255,.06);
  --pwchat-border: rgba(255,255,255,.12);
  --pwchat-accent: #7c5cff;
  --pwchat-accent-fg: #0b0f19;
  --pwchat-muted: rgba(255,255,255,.60);
  --pwchat-muted-fg: rgba(255,255,255,.82);

  --pwchat-font-sans: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Arial, "Apple Color Emoji", "Segoe UI Emoji";
  --pwchat-font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  --pwchat-font-size: 14px;
  --pwchat-line-height: 1.4;

  --pwchat-radius: 12px;
  --pwchat-gap: 10px;
  --pwchat-padding: 10px;

  --pwchat-header-h: 44px;
  --pwchat-composer-min-h: 52px;
}
```

### Default layout & part styles (also low specificity)

```css
:where([data-pwchat][data-part="root"]) {
  font-family: var(--pwchat-font-sans);
  font-size: var(--pwchat-font-size);
  line-height: var(--pwchat-line-height);

  background: var(--pwchat-bg);
  color: var(--pwchat-fg);

  border: 1px solid var(--pwchat-border);
  border-radius: var(--pwchat-radius);
  overflow: hidden;

  display: flex;
  flex-direction: column;
  min-width: 280px;
}

:where([data-pwchat] [data-part="header"]) {
  height: var(--pwchat-header-h);
  display: flex;
  align-items: center;
  gap: var(--pwchat-gap);
  padding: 0 var(--pwchat-padding);
  border-bottom: 1px solid var(--pwchat-border);
  background: var(--pwchat-surface-1);
}

:where([data-pwchat] [data-part="timeline"]) {
  flex: 1;
  overflow: auto;
  padding: var(--pwchat-padding);
  display: flex;
  flex-direction: column;
  gap: var(--pwchat-gap);
}

:where([data-pwchat] [data-part="turn"]) {
  display: flex;
}

:where([data-pwchat] [data-part="turn"][data-role="user"]) {
  justify-content: flex-end;
}

:where([data-pwchat] [data-part="bubble"]) {
  max-width: min(720px, 92%);
  background: var(--pwchat-surface-2);
  border: 1px solid var(--pwchat-border);
  border-radius: calc(var(--pwchat-radius) - 4px);
  padding: 10px 12px;
}

:where([data-pwchat] [data-part="turn"][data-role="user"] [data-part="bubble"]) {
  background: color-mix(in oklab, var(--pwchat-accent) 20%, var(--pwchat-surface-2));
  border-color: color-mix(in oklab, var(--pwchat-accent) 40%, var(--pwchat-border));
}

:where([data-pwchat] [data-part="composer"]) {
  padding: var(--pwchat-padding);
  border-top: 1px solid var(--pwchat-border);
  background: var(--pwchat-surface-1);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

:where([data-pwchat] [data-part="composer-input"]) {
  width: 100%;
  min-height: var(--pwchat-composer-min-h);
  resize: vertical;

  background: transparent;
  color: var(--pwchat-fg);
  border: 1px solid var(--pwchat-border);
  border-radius: calc(var(--pwchat-radius) - 6px);
  padding: 10px 12px;
  outline: none;
}

:where([data-pwchat] [data-part="composer-actions"]) {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

:where([data-pwchat] [data-part="send-button"]) {
  border: 1px solid var(--pwchat-border);
  background: var(--pwchat-accent);
  color: var(--pwchat-accent-fg);
  border-radius: calc(var(--pwchat-radius) - 6px);
  padding: 8px 10px;
}

:where([data-pwchat] [data-part="send-button"][data-disabled]) {
  opacity: .55;
  cursor: not-allowed;
}
```

Note: `color-mix()` is optional; if you want maximum compatibility you can skip it. The key pattern is the scoping and low specificity.

---

## 4) Component API (minimal) for `@pinocchio/webchat/react`

### Types

```ts
export type ThemeVars = Partial<Record<`--pwchat-${string}`, string>>;

export type RootPartProps = React.HTMLAttributes<HTMLDivElement> & {
  "data-pwchat"?: "";
  "data-part"?: "root";
  "data-theme"?: string;
};

export type HeaderSlotProps = {
  partProps: React.HTMLAttributes<HTMLDivElement>;
  title?: React.ReactNode;
  right?: React.ReactNode;
};

export type ComposerSlotProps = {
  partProps: React.HTMLAttributes<HTMLDivElement>;
  inputProps: React.TextareaHTMLAttributes<HTMLTextAreaElement>;
  actionsProps: React.HTMLAttributes<HTMLDivElement>;
  sendButtonProps: React.ButtonHTMLAttributes<HTMLButtonElement>;
  onSend: () => void;
};

export type TurnSlotProps = {
  partProps: React.HTMLAttributes<HTMLDivElement>;
  bubbleProps: React.HTMLAttributes<HTMLDivElement>;
  contentProps: React.HTMLAttributes<HTMLDivElement>;
  role: "user" | "assistant" | "tool" | "system";
  entity: TimelineEntity; // from your protocol mapping
};

export type Components = {
  Header: React.ComponentType<HeaderSlotProps>;
  Composer: React.ComponentType<ComposerSlotProps>;
  Turn: React.ComponentType<TurnSlotProps>;
};
```

### Props

```ts
export type WebchatWidgetProps = {
  // behavior
  endpoint: WebchatEndpoint;
  convId?: string;
  profile?: string;

  // styling contract
  unstyled?: boolean;
  themeVars?: ThemeVars;

  // slots
  components?: Partial<Components>;

  // escape hatches
  className?: string;
  rootProps?: React.HTMLAttributes<HTMLDivElement>;
};
```

### Applying theme marker + tokens

```tsx
export function WebchatWidget({
  endpoint,
  convId,
  profile,
  unstyled,
  themeVars,
  components,
  className,
  rootProps,
}: WebchatWidgetProps) {
  const theme = unstyled ? undefined : "default";

  const Header = components?.Header ?? DefaultHeader;
  const Composer = components?.Composer ?? DefaultComposer;
  const Turn = components?.Turn ?? DefaultTurn;

  // ... get timeline + sendPrompt from your client/provider
  const timeline = useTimelineEntities();
  const { sendPrompt, connection } = useWebchat();

  return (
    <div
      data-pwchat=""
      data-part="root"
      data-theme={theme}
      className={className}
      style={themeVars as React.CSSProperties}
      {...rootProps}
    >
      <Header
        partProps={{ "data-part": "header" }}
        title={"Pinocchio"}
        right={<ConnectionBadge state={connection} />}
      />

      <div data-part="timeline">
        {timeline.map((entity) => (
          <Turn
            key={entity.id}
            partProps={{
              "data-part": "turn",
              "data-role": inferRole(entity),
            } as any}
            bubbleProps={{ "data-part": "bubble" }}
            contentProps={{ "data-part": "content" }}
            role={inferRole(entity)}
            entity={entity}
          />
        ))}
      </div>

      <Composer
        partProps={{ "data-part": "composer" }}
        inputProps={{ "data-part": "composer-input" } as any}
        actionsProps={{ "data-part": "composer-actions" }}
        sendButtonProps={{ "data-part": "send-button" } as any}
        onSend={() => sendPrompt(/* ... */)}
      />
    </div>
  );
}
```

Key point: **slots never need to “remember” `data-part` wiring** because you pass `partProps`.

---

## 5) Slots that don’t break styling: `partProps` pattern

### Default turn renderer

```tsx
function DefaultTurn({
  partProps,
  bubbleProps,
  contentProps,
  entity,
}: TurnSlotProps) {
  return (
    <div {...partProps}>
      <div {...bubbleProps}>
        <div {...contentProps}>
          <RenderEntity entity={entity} />
        </div>
      </div>
    </div>
  );
}
```

### Consumer can override without losing selectors

```tsx
function MyTurn({ partProps, bubbleProps, contentProps, entity }: TurnSlotProps) {
  return (
    <article {...partProps}>
      <section {...bubbleProps}>
        <div {...contentProps}>
          <MyFancyEntity entity={entity} />
        </div>
      </section>
    </article>
  );
}

<WebchatWidget components={{ Turn: MyTurn }} />
```

They still get `data-part="turn"`, `data-part="bubble"`, etc.

---

## 6) Packaging: publish CSS cleanly (explicit import)

### Export a default theme CSS file

Consumers opt in:

```ts
import "@pinocchio/webchat/theme-default.css";
```

### `package.json` (single package with explicit CSS export)

```json
{
  "name": "@pinocchio/webchat",
  "sideEffects": ["**/*.css"],
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "import": "./dist/index.js"
    },
    "./theme-default.css": "./dist/theme-default.css"
  },
  "peerDependencies": {
    "react": "^18 || ^19",
    "react-dom": "^18 || ^19"
  }
}
```

### Folder layout

```
src/
  index.ts
  react/
    WebchatWidget.tsx
    parts.ts
    theme/
      default.css
dist/
  index.js
  index.d.ts
  theme-default.css
```

You can keep `parts.ts` optional, but it’s handy to avoid typos:

```ts
export const PWCHAT_PARTS = {
  root: "root",
  header: "header",
  timeline: "timeline",
  turn: "turn",
  bubble: "bubble",
  content: "content",
  composer: "composer",
  composerInput: "composer-input",
  composerActions: "composer-actions",
  sendButton: "send-button",
  statusbar: "statusbar",
} as const;
```

---

## 7) Consumer customization patterns you enable

### A) Override tokens locally (no CSS file needed)

```tsx
<div
  className="myChat"
  style={{
    ["--pwchat-accent" as any]: "hotpink",
    ["--pwchat-radius" as any]: "16px",
  }}
>
  <WebchatWidget endpoint={...} />
</div>
```

### B) Write CSS against stable `data-part`

```css
.myChat [data-part="header"] {
  border-bottom: 2px solid var(--pwchat-accent);
}

.myChat [data-part="turn"][data-role="user"] [data-part="bubble"] {
  border-style: dashed;
}
```

### C) Turn off default theme completely

```tsx
<WebchatWidget endpoint={...} unstyled />
```

Now the consumer owns all CSS, but still has stable `data-part` hooks.

### D) Replace structure with slots safely

```tsx
<WebchatWidget
  endpoint={...}
  components={{
    Header: MyHeader,
    Composer: MyComposer,
  }}
/>
```

---

## 8) What to change in the current webchat UI to match this

To apply this cleanly to the existing `ChatWidget.tsx` codebase:

* Stop relying on global CSS class selectors as the styling “API”
* Add the stable `data-pwchat` + `data-part` attributes in the top-level widget and major regions
* Ensure “role/state” attributes live on the element consumers will style:

  * `data-role` on the `turn`
  * `data-state` on `statusbar` / root if you show connection state
* Convert key replaceable regions into slots and pass `partProps` so overrides don’t break theming:

  * Header
  * Turn renderer (or per-kind renderer map)
  * Composer

Everything else can remain internal.

---


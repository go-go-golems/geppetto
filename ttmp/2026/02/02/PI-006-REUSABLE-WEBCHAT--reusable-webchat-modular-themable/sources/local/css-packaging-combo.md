Below is a reusable **“Combo A” CSS packaging pattern guide** you can copy for any React component library (an IDE, a dashboard shell, a file picker, whatever) where you want:

* **Default nice look** out of the box
* **Maximum styling freedom** for consumers
* A **small, supportable** API surface (not 18 styling knobs)

This pattern is: **CSS variables (tokens) + stable `data-part` selectors + optional default theme CSS + slots get `partProps`.**

---

## 0) The contract you’re promising (keep this small)

You are committing to **two public styling surfaces**:

### A) Tokens (CSS variables)

A finite list like `--ide-bg`, `--ide-fg`, `--ide-border`, `--ide-radius`, etc.

### B) Structure selectors (stable `data-*`)

A finite list of “parts”, e.g.:

* `data-part="root"`
* `data-part="toolbar"`
* `data-part="tabs"`
* `data-part="tab"`
* `data-part="editor"`
* `data-part="statusbar"`

…and a small list of state attributes:

* `data-state="active|inactive"`
* `data-collapsed`
* `data-disabled`

**Everything else is internal and can change** without breaking users.

---

## 1) How you structure the DOM (so styling is reliable)

### Always render these

* `data-ide=""` on the root (or `data-yourcomponent=""`)
* `data-part="root"` on the root

### Render stable parts for major regions only

Don’t expose selectors for every wrapper div—only for things users will style.

Example:

```tsx
export function MyShell(props) {
  return (
    <div data-ide="" data-part="root">
      <div data-part="toolbar" />
      <div data-part="workspace">
        <div data-part="sidebar" data-collapsed />
        <div data-part="main">
          <div data-part="tabs" />
          <div data-part="content" />
        </div>
      </div>
      <div data-part="statusbar" />
    </div>
  );
}
```

### State goes on the element people style

Example: tabs

```tsx
<button data-part="tab" data-state={active ? "active" : "inactive"} />
```

This gives consumers durable selectors like:

```css
.myThing [data-part="tab"][data-state="active"] { … }
```

---

## 2) Tokens: design them once, keep them boring

### Recommended token categories

**Core colors**

* `--ide-bg`, `--ide-fg`
* `--ide-surface-1`, `--ide-surface-2`
* `--ide-border`
* `--ide-accent`, `--ide-accent-fg`
* `--ide-muted`, `--ide-muted-fg`

**Typography**

* `--ide-font-sans`, `--ide-font-mono`
* `--ide-font-size`, `--ide-line-height`

**Shape & spacing**

* `--ide-radius`
* `--ide-gap`
* `--ide-padding`

**A small number of component dims** (only if needed)

* `--ide-toolbar-h`
* `--ide-statusbar-h`

Rule of thumb: **prefer general tokens**; add component tokens only when users ask.

---

## 3) Default theme CSS: optional, low-specificity, scoped

### Scope the theme to the component root

Use the root marker + a theme marker:

* Root always has `data-ide`
* Default theme adds `data-theme="default"` (unless `unstyled`)

In CSS:

```css
:where([data-ide][data-theme="default"]) {
  --ide-bg: #0b0f19;
  --ide-fg: #e6e6e6;
  --ide-border: rgba(255,255,255,.12);
  --ide-accent: #7c5cff;
  --ide-radius: 12px;
  --ide-gap: 8px;
  --ide-padding: 10px;
}
```

**Why `:where()`?** It keeps specificity extremely low, so consumers can override without fighting.

### Default layout styles (also low-specificity)

```css
:where([data-ide][data-part="root"]) {
  background: var(--ide-bg);
  color: var(--ide-fg);
  border: 1px solid var(--ide-border);
  border-radius: var(--ide-radius);
  overflow: hidden;
}

:where([data-ide] [data-part="toolbar"]) {
  height: var(--ide-toolbar-h, 40px);
  border-bottom: 1px solid var(--ide-border);
}
```

Keep defaults “pretty but not opinionated”—avoid heavy typography or strong shadows unless your brand demands it.

---

## 4) The component API: only what you need

### Minimal props

* `unstyled?: boolean`
* `themeVars?: Record<"--ide-…", string>` (optional convenience)
* `components?: { …slots }` (deep override)
* `className?` + `rootProps?` (escape hatch)

Example:

```ts
export type ThemeVars = Partial<Record<`--ide-${string}`, string>>;

export type Components = {
  Toolbar: React.ComponentType<ToolbarSlotProps>;
  Statusbar: React.ComponentType<StatusbarSlotProps>;
};

export type Props = {
  unstyled?: boolean;
  themeVars?: ThemeVars;
  components?: Partial<Components>;
  className?: string;
  rootProps?: React.HTMLAttributes<HTMLDivElement>;
};
```

### Apply theme marker only when not unstyled

```tsx
const themeAttr = unstyled ? undefined : "default";

return (
  <div
    data-ide=""
    data-part="root"
    data-theme={themeAttr}
    className={className}
    style={themeVars as React.CSSProperties}
    {...rootProps}
  >
    …
  </div>
);
```

That’s the whole “default theme is optional” trick.

---

## 5) Slots that don’t break styling: pass `partProps`

When you allow users to replace a subcomponent, **don’t make them recreate your `data-part` wiring**.

Do this:

```ts
export type ToolbarSlotProps = {
  partProps: React.HTMLAttributes<HTMLDivElement>;
  left?: React.ReactNode;
  right?: React.ReactNode;
};
```

Default toolbar:

```tsx
function DefaultToolbar({ partProps, left, right }: ToolbarSlotProps) {
  return (
    <div {...partProps}>
      <div data-part="toolbar-left">{left}</div>
      <div data-part="toolbar-right">{right}</div>
    </div>
  );
}
```

When rendering the slot:

```tsx
const Toolbar = components?.Toolbar ?? DefaultToolbar;

<Toolbar
  partProps={{ "data-part": "toolbar" } as any}
  left={…}
  right={…}
/>
```

Now users can override safely:

```tsx
function MyToolbar({ partProps, left, right }: ToolbarSlotProps) {
  return (
    <header {...partProps}>
      {left}
      <div style={{ marginLeft: "auto" }}>{right}</div>
    </header>
  );
}
```

They still get your `data-part="toolbar"` selector for theming.

---

## 6) Packaging: how to distribute CSS cleanly

### Recommend: export the CSS as an explicit file

Consumers opt in:

```ts
import "@scope/my-component/theme-default.css";
```

In `package.json`:

```json
{
  "name": "@scope/my-component",
  "sideEffects": ["**/*.css"],
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "import": "./dist/index.js"
    },
    "./theme-default.css": "./dist/theme-default.css"
  },
  "peerDependencies": {
    "react": "^18 || ^19"
  }
}
```

**Why `sideEffects`?** So bundlers don’t tree-shake away the CSS import.

### Folder layout

```
src/
  index.ts
  MyComponent.tsx
  parts.ts              // constants for data-part names (optional)
  theme/
    default.css
dist/
  index.js
  index.d.ts
  theme-default.css
```

---

## 7) Consumer customization patterns you’re enabling

### A) Override tokens locally

```tsx
<div style={{ ["--ide-accent" as any]: "hotpink" }}>
  <MyComponent />
</div>
```

### B) Write CSS against `data-part`

```css
.myShell [data-part="toolbar"] {
  border-bottom: 2px solid var(--ide-accent);
}
.myShell [data-part="tab"][data-state="active"] {
  font-weight: 600;
}
```

### C) Turn off your theme completely

```tsx
<MyComponent unstyled />
```

Now the consumer supplies everything, but still has a stable DOM contract.

### D) Replace structure with slots

```tsx
<MyComponent components={{ Toolbar: MyToolbar }} />
```

---

## 8) Your “don’t regret it later” checklist

* ✅ Publish a **small** list of tokens (don’t add 50 on day one)
* ✅ Publish a **small** list of parts (only major regions)
* ✅ Use `:where()` in default theme CSS
* ✅ Scope default theme to `[data-thing][data-theme="default"]`
* ✅ Provide `unstyled` mode
* ✅ Slots receive `partProps` (so theming stays consistent)
* ❌ Don’t add `styles` + `classNames` + `sx` + theme provider all at once
  (pick one lane; tokens are that lane)

---

## 9) Copy/paste template you can start from

### Component skeleton

```tsx
type ThemeVars = Partial<Record<`--x-${string}`, string>>;

export function X({
  unstyled,
  themeVars,
  components,
  className,
  rootProps,
}: {
  unstyled?: boolean;
  themeVars?: ThemeVars;
  components?: Partial<{ Toolbar: React.ComponentType<any> }>;
  className?: string;
  rootProps?: React.HTMLAttributes<HTMLDivElement>;
}) {
  const theme = unstyled ? undefined : "default";
  const Toolbar = components?.Toolbar ?? DefaultToolbar;

  return (
    <div
      data-x=""
      data-part="root"
      data-theme={theme}
      className={className}
      style={themeVars as React.CSSProperties}
      {...rootProps}
    >
      <Toolbar partProps={{ "data-part": "toolbar" } as any} />
      <div data-part="content" />
    </div>
  );
}
```

### Default theme CSS

```css
:where([data-x][data-theme="default"]) {
  --x-bg: #111;
  --x-fg: #eee;
  --x-border: rgba(255,255,255,.12);
  --x-radius: 12px;
}

:where([data-x][data-part="root"]) {
  background: var(--x-bg);
  color: var(--x-fg);
  border: 1px solid var(--x-border);
  border-radius: var(--x-radius);
}
```


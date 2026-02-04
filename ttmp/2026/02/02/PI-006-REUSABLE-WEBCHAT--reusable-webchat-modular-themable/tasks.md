# Tasks

## TODO

- [x] Map current chat DOM/classes to new `data-part` contract and token list.
- [x] Add new webchat stylesheets (`theme-default.css`, `webchat.css`) with low-specificity selectors.
- [x] Introduce `src/webchat/` module with public types + exports.
- [x] Refactor ChatWidget into modular components (header, timeline, composer, cards).
- [x] Replace class names with `data-pwchat`/`data-part`/`data-state`/`data-role` attributes.
- [x] Implement theming props: `unstyled`, `theme`, `themeVars`, `partProps`.
- [x] Implement customization props: `components` slots + `renderers` map.
- [x] Update Markdown/toolbar/button styles to use new parts/tokens.
- [x] Update Storybook stories to demonstrate default theme + overrides.
- [x] Remove legacy `chat.css` usage and ensure Vite app still imports new styles.
- [x] Run `npm run check` in `pinocchio/cmd/web-chat/web`.
- [x] Validate Storybook build or dev smoke-test (optional).
- [x] Update diary + changelog; mark tasks complete.

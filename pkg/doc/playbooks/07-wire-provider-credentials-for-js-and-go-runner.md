---
Title: "Wire provider credentials for JS and go runner"
Slug: wire-provider-credentials-js-go-runner
Short: Provider/model settings are now explicit app config, not profile runtime patches.
Topics:
  - profiles
  - configuration
  - javascript
  - extraction
Commands:
  - geppetto
  - temporal-relationships
Flags:
  - profile
  - profile-registries
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Playbook
---

# Wire provider credentials for JS and go runner

Provider credentials and engine settings must now be wired explicitly:

- Go apps: resolve base `StepSettings` from config/env/flags, then build the engine from those final settings.
- JS apps: call `gp.engines.fromConfig(...)` with explicit `apiType`, `model`, `apiKey`, and optional `baseURL`.

Profiles no longer carry provider credentials or engine-setting patches. They only contribute runtime metadata such as:

- `system_prompt`
- `tools`
- `middlewares`

Example JS:

```javascript
const gp = require("geppetto");

const engine = gp.engines.fromConfig({
  apiType: "openai",
  model: "gpt-4.1-mini",
  apiKey: process.env.OPENAI_API_KEY,
});
```

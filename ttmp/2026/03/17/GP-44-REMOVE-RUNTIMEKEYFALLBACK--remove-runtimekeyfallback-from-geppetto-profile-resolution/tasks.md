# Tasks

## Implementation Checklist

- [x] Confirm every in-repo `RuntimeKeyFallback` definition and call site.
- [x] Confirm whether runtime-key fallback influences engine construction, provider selection, or runtime fingerprinting.
- [x] Draft a design and implementation guide for removing the field.
- [ ] Remove `RuntimeKeyFallback` from [pkg/profiles/registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go).
- [ ] Remove `RuntimeKey` from `ResolvedProfile` in [pkg/profiles/registry.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/registry.go).
- [ ] Simplify [pkg/profiles/service.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service.go) so `ResolveEffectiveProfile` no longer parses or emits runtime-key data.
- [ ] Update [pkg/profiles/service_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/service_test.go) to stop asserting runtime-key behavior.
- [ ] Update [pkg/profiles/source_chain.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/profiles/source_chain.go) call flow if any pass-through fields remain after API cleanup.
- [ ] Remove `runtimeKeyFallback` / `runtimeKey` handling from [pkg/js/modules/geppetto/api_profiles.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_profiles.go).
- [ ] Remove `runtimeKey` handling from [pkg/js/modules/geppetto/api_engines.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/api_engines.go).
- [ ] Update JS tests in [pkg/js/modules/geppetto/module_test.go](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/module_test.go).
- [ ] Update generated or templated type definitions in [pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl) and [pkg/doc/types/geppetto.d.ts](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/types/geppetto.d.ts).
- [ ] Update examples in [examples/js/geppetto](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/examples/js/geppetto) to stop showing runtime-key options or outputs.
- [ ] Update docs in [pkg/doc/topics/01-profiles.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/01-profiles.md), [pkg/doc/topics/13-js-api-reference.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/13-js-api-reference.md), and [pkg/doc/topics/14-js-api-user-guide.md](/home/manuel/workspaces/2026-03-17/add-opinionated-apis/geppetto/pkg/doc/topics/14-js-api-user-guide.md).
- [ ] Review downstream code for any consumer that relied on `ResolvedProfile.RuntimeKey` as an output-only label.
- [ ] Run `go test ./pkg/profiles ./pkg/js/modules/geppetto`.
- [ ] Run repo-wide grep for `RuntimeKeyFallback`, `runtimeKeyFallback`, and `runtimeKey` to confirm complete cleanup.

## Review Checklist

- [ ] Verify that registry resolution still uses registry-stack precedence and profile slug only.
- [ ] Verify that runtime fingerprint output is unchanged for identical profile/runtime inputs.
- [ ] Verify that `engines.fromProfile` metadata still contains `profileRegistry`, `profileSlug`, and `runtimeFingerprint`.
- [ ] Verify that no doc still claims `runtimeKey` is a supported resolver knob.

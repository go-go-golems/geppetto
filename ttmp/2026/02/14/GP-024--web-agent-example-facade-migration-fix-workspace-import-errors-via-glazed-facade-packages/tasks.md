# Tasks

## Phase 0: Analysis and planning

- [x] Read `glaze help migrating-to-facade-packages` in full.
- [x] Store help output under ticket `sources/`.
- [x] Create detailed implementation plan document.

## Phase 1: Main command migration (`cmd/web-agent-example/main.go`)

- [ ] Replace legacy imports:
  - `geppetto/pkg/layers` -> `geppetto/pkg/sections`
  - `glazed/pkg/cmds/layers` -> `glazed/pkg/cmds/values`
  - `glazed/pkg/cmds/parameters` -> `glazed/pkg/cmds/fields`
- [ ] Replace legacy API usage:
  - `CreateGeppettoLayers` -> `CreateGeppettoSections`
  - `WithLayersList` -> `WithSections`
  - `parameters.NewParameterDefinition` -> `fields.New`
  - `RunIntoWriter(...*layers.ParsedLayers...)` -> `RunIntoWriter(...*values.Values...)`
  - `InitializeStruct(layers.DefaultSlug, ...)` -> `DecodeSectionInto(values.DefaultSlug, ...)`
- [ ] Run `gofmt` on changed files.

## Phase 2: Compile/test recovery

- [ ] Run `go test ./cmd/web-agent-example`.
- [ ] Run `go test ./...` in `web-agent-example` if feasible.

## Phase 3: Resolver test coverage

- [ ] Add tests for `noCookieRequestResolver`:
  - WS `conv_id` required + default runtime behavior
  - chat request parsing (`prompt`/`text`), conv_id generation
  - unsupported method error mapping
- [ ] Run tests after adding coverage.

## Phase 4: Ticket bookkeeping

- [ ] Update diary with each implementation slice and command outcomes.
- [ ] Update changelog with each commit.
- [ ] Check off completed tasks as work lands.

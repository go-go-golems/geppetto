# Tasks

## TODO

- [ ] Confirm the “serializable bags” rule applies across geppetto/moments/pinocchio/go-go-mento (no exceptions).
- [ ] Define canonical serializable tool representation to store on Turn (e.g., `[]tools.ToolDefinition`) and add a typed Turn.Data key for it.
- [ ] Add `toolcontext.WithRegistry/RegistryFrom` helper for passing runtime registry via `context.Context`.
- [ ] Update writers (routers/handlers/toolhelpers) to stop setting `Turn.Data[turns.DataKeyToolRegistry]`.
- [ ] Update engines (OpenAI/Claude/Gemini) to read tool definitions from Turn.Data instead of a registry object.
- [ ] Update tracing middleware to read tool definitions from Turn.Data (no registry needed).
- [ ] Update sqlite tool middleware to mutate runtime registry via context (or append defs + rebuild registry; pick one).
- [ ] Update persistence to remove registry special-casing once Turn.Data no longer contains registry objects.
- [ ] Add tests: turn YAML round-trip with tool defs; engine request includes tools; execution fails clearly if ctx registry missing.


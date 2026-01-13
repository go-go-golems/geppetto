## Timeline aggregation for unified events in simple-chat-agent (web_search panel)

### Purpose
- Adapt the simple-chat-agent timeline to the unified streaming events by aggregating related provider events into a single, evolving UI entity instead of emitting one entity per event.
- First concrete target: a web_search panel that tracks the lifecycle and progress of built-in web search calls from OpenAI Responses.

### Current timeline pattern (bobatea)
- Entities are created/updated/completed/deleted via append-only lifecycle messages:
  - `UIEntityCreated{ID, Renderer{Kind|Key}, Props, StartedAt}`
  - `UIEntityUpdated{ID, Patch, Version, UpdatedAt}`
  - `UIEntityCompleted{ID, Result}`
  - `UIEntityDeleted{ID}`
- The controller maintains an append-only store and delegates rendering to Bubble Tea models registered in a `timeline.Registry`.
- Selection and key routing are centralized in the controller; models implement local state and render strings.

Refs: `bobatea/docs/timeline.md`, `bobatea/docs/how-to-build-a-entity-renderer-and-wire-it-up.md`, `bobatea/docs/how-to-build-a-timeline-backend-for-llm-chat-applications.md`.

### New unified events relevant to web search
- Event types surfaced by the Responses engine:
  - `EventWebSearchStarted{ItemID, Query?}`
  - `EventWebSearchSearching{ItemID}`
  - `EventWebSearchOpenPage{ItemID, URL}`
  - `EventWebSearchDone{ItemID}`
  - (Optionally) `EventToolSearchResults{Tool:"web_search", ItemID, Results:[{url,title,snippet,ext}]}`

### Aggregation strategy
- One timeline entity per provider `ItemID` for the web search call.
- Renderer kind: `web_search` (renderer key may be `renderer.web_search_panel.v1`).
- Entity identity: `EntityID{ LocalID: itemID, Kind: "web_search" }`.
- Backend forwarder maps incoming events to create/update/complete that single entity.

### Web search panel: renderer state and props
- Initial Props on create:
  - `status: "searching" | "in_progress" | "completed" | "failed"`
  - `query?: string`
  - `opened_urls: []string`
  - `results: []Result` where `Result{url,title,snippet,ext?}`
  - `progress?: string` (freeform, e.g., "searching…")
  - `error?: string`

- Update patches:
  - On `EventWebSearchSearching`: `{ status: "searching", progress: "searching" }`
  - On `EventWebSearchOpenPage`: `{ opened_urls: append }`
  - On `EventToolSearchResults`: `{ results: append }`
  - On `EventWebSearchDone`: complete + `{ status: "completed" }`

- Renderer behavior (Bubble Tea model):
  - Header line with spinner + query when `status` not completed.
  - Subsection for opened pages as they stream in.
  - Results list grows incrementally; each result renders `title` with a dim `url` and optional snippet.
  - On completed, stop spinner, show counts.

Example pseudocode for the model:
```go
type WebSearchModel struct {
  width int
  status string
  query string
  opened []string
  results []Result
}

func (m *WebSearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  switch v := msg.(type) {
  case timeline.EntityPropsUpdatedMsg:
    if s, ok := v.Patch["status"].(string); ok { m.status = s }
    if q, ok := v.Patch["query"].(string); ok { m.query = q }
    if urls, ok := v.Patch["opened_urls"].([]string); ok { m.opened = urls }
    if rs, ok := v.Patch["results"].([]Result); ok { m.results = rs }
  case timeline.EntitySetSizeMsg:
    m.width = v.Width
  }
  return m, nil
}
```

### Backend mapping (pinocchio simple-chat-agent)
- Location: `pinocchio/cmd/agents/simple-chat-agent/pkg/backend/tool_loop_backend.go` (UI forwarder).
- Maintain a map `itemID -> created` in the backend (or derive deterministically from messages) to ensure a single entity per `ItemID`.
- Mapping rules:
  - `EventWebSearchStarted{ItemID, Query}` → If not exists, send `UIEntityCreated{ ID: {LocalID:itemID, Kind:"web_search"}, Renderer:{Kind:"web_search"}, Props:{ status:"searching", query, opened_urls:[], results:[] }, StartedAt: now }`. If exists and `query` provided, send `UIEntityUpdated` to set `query`.
  - `EventWebSearchSearching{ItemID}` → `UIEntityUpdated{ Patch:{ status:"searching", progress:"searching" } }`.
  - `EventWebSearchOpenPage{ItemID, URL}` → `UIEntityUpdated{ Patch:{ opened_urls: append(URL) } }`.
  - `EventToolSearchResults{Tool:"web_search", ItemID, Results}` → `UIEntityUpdated{ Patch:{ results: append(Results...) } }`.
  - `EventWebSearchDone{ItemID}` → `UIEntityUpdated{ Patch:{ status:"completed" } }` and `UIEntityCompleted{}`.

Example pseudocode (forwarder branch):
```go
case *events.EventWebSearchStarted:
  id := timeline.EntityID{LocalID: e.ItemID, Kind: "web_search"}
  if !created[id.LocalID] {
    p.Send(timeline.UIEntityCreated{
      ID: id,
      Renderer: timeline.RendererDescriptor{Kind: "web_search"},
      Props: map[string]any{"status":"searching", "query": e.Query, "opened_urls": []string{}, "results": []Result{}},
      StartedAt: time.Now(),
    })
    created[id.LocalID] = true
  } else if e.Query != "" {
    p.Send(timeline.UIEntityUpdated{ID: id, Patch: map[string]any{"query": e.Query}, Version: time.Now().UnixNano(), UpdatedAt: time.Now()})
  }

case *events.EventWebSearchSearching:
  id := timeline.EntityID{LocalID: e.ItemID, Kind: "web_search"}
  p.Send(timeline.UIEntityUpdated{ID: id, Patch: map[string]any{"status":"searching", "progress":"searching"}, Version: time.Now().UnixNano(), UpdatedAt: time.Now()})

case *events.EventWebSearchOpenPage:
  id := timeline.EntityID{LocalID: e.ItemID, Kind: "web_search"}
  // Backend keeps current opened_urls; send full list or a delta and merge in the renderer.
  p.Send(timeline.UIEntityUpdated{ID: id, Patch: map[string]any{"opened_urls.append": e.URL}, Version: time.Now().UnixNano(), UpdatedAt: time.Now()})

case *events.EventToolSearchResults:
  if e.Tool == "web_search" {
    id := timeline.EntityID{LocalID: e.ItemID, Kind: "web_search"}
    p.Send(timeline.UIEntityUpdated{ID: id, Patch: map[string]any{"results.append": e.Results}, Version: time.Now().UnixNano(), UpdatedAt: time.Now()})
  }

case *events.EventWebSearchDone:
  id := timeline.EntityID{LocalID: e.ItemID, Kind: "web_search"}
  p.Send(timeline.UIEntityUpdated{ID: id, Patch: map[string]any{"status":"completed"}, Version: time.Now().UnixNano(), UpdatedAt: time.Now()})
  p.Send(timeline.UIEntityCompleted{ID: id})
```

Note: The `.append` convention above is illustrative. Either:
- Keep lists in backend state and send full arrays in each patch, or
- Implement an append semantic in the renderer by interpreting special keys (preferred only if standardized across renderers).

### Renderer registration
- Provide a renderer factory `WebSearchFactory` and register it in the timeline registry used by the chat model, e.g. via a hook when building the UI.
- Descriptor suggestion: `RendererDescriptor{Kind: "web_search"}`.

### Interaction and UX
- Spinner while searching; update to a checkmark on completion.
- Show `query` prominently; opened URLs below as they stream in.
- Results listed with titles; toggle details (snippets) on selection with TAB (leveraging controller’s TAB routing policy).
- Error state: if a `failed` event is surfaced in the future, show error banner; for now `EventWebSearchDone` marks completion.

### Data considerations
- If `EventToolSearchResults` isn’t emitted yet by the engine, the panel still provides value by showing progress and opened pages; results can be added later without changing identity or flow.
- Use provider `ItemID` for stable grouping; if absent, derive from `message_id` + event index but prefer explicit `ItemID` (Responses provides it).

### Extensibility
- Apply the same aggregator pattern to other built-ins:
  - File search: `EventFileSearch*` → `file_search` panel
  - Code interpreter: `EventCodeInterpreter*` + code deltas → `code_interpreter` panel
  - Image generation: partial image + completed → `image_generation` panel
  - MCP activity: `EventMCP*` → `mcp` panel

### Minimal changes summary
- Backend: extend UI forwarder with web_search aggregation branches using `ItemID` as `LocalID`.
- UI: implement and register `web_search` renderer; no changes to controller are needed.
- Store: optionally persist web_search progress by logging `EventWebSearch*` and appending results.

### Test plan
- Run `cmd/examples/openai-tools --mode server-tools` and verify router printers show `EventWebSearch*`.
- In simple-chat-agent, trigger a server-side web_search; observe a single `web_search` entity:
  - Created on `Started`
  - Updated on `Searching` and `OpenPage`
  - Completed on `Done`
  - If available, results append in-place

### References
- `bobatea/docs/timeline.md`
- `bobatea/docs/how-to-build-a-entity-renderer-and-wire-it-up.md`
- `bobatea/docs/how-to-build-a-timeline-backend-for-llm-chat-applications.md`
- `geppetto/pkg/events/chat-events.go` (event types)
- `geppetto/pkg/steps/ai/openai_responses/engine.go` (emission sites)



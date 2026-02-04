Below is a concrete, “make-it-reusable” backend design for the Pinocchio webchat that keeps the public surface small, fixes the big integration pain points (mounting, UI vs API, auth/policy), and borrows the best bits from your `mento-webchat` refactor (stream ordering/versioning + clearer lifecycle separation).

I’m going to talk in terms of *what you should commit to publicly* (stable API), what becomes internal/private, and what refactor steps get you there without a big-bang rewrite.

---

## 1) What’s “messy” today in `pinocchio/pkg/webchat` (and why it blocks reuse)

From reading the current code, the biggest reuse blockers aren’t “bugs”, they’re *coupling*:

### A) Router is doing too much

`router.go` currently owns:

* static UI serving
* websocket upgrade + read loop + ping/pong
* `/chat` request policy resolution + idempotency + queueing + run start
* `/timeline` snapshot API
* debug endpoints
* conversation lifecycle (via `getOrCreateConv`)
* event router wiring (redis/inmem)
* timeline projection hooks

This makes it hard to embed without also inheriting everything else.

### B) Mounting under a subpath is effectively broken

Your `Router.Mount(mux, prefix)` uses `mux.Handle(prefix, r.mux)`, but `http.ServeMux` **does not strip prefixes**.
So if you mount at `/api/webchat`, the inner mux sees `/api/webchat/chat`, but it only registered `/chat`, `/ws`, etc → no match.

This is one of the main reasons integrating into “my app” feels painful.

### C) UI serving is not optional in a clean way

`NewRouter(ctx, parsed, staticFS embed.FS)` always sets up static handlers and `/` index logic. If you embed this into an existing server you often **don’t want it to own `/`** at all.

### D) Tight coupling to Pinocchio CLI/layers

The core backend package assumes `*layers.ParsedLayers` for settings and redis/router composition. That’s fine for the CLI, but it makes it harder to reuse as a library in a different host application that does not use Glazed layers.

### E) Streaming order/versioning is “local”

Pinocchio’s `StreamCoordinator` attaches a `seq` via an in-process atomic counter. Your Mento refactor has a better idea: when streaming comes from Redis streams, use the Redis stream ID (xid) to derive a monotonic version and thread it through.

That matters for:

* correct ordering across restarts / multiple instances
* consistent hydration / reconciliation with persistence

---

## 2) The backend “contract” you should promise (keep it small)

Mirroring your frontend “Combo A” idea: the backend needs **two public surfaces**:

### A) Wire protocol surface (HTTP + WS)

This is what your NPM UI package depends on. Keep it stable and version it.

**Recommend**: put everything under a versioned base, e.g. `/v1`.

* `GET /v1/profiles` → list available profiles
* `POST /v1/chat` → submit user prompt (and optional overrides)
* `GET /v1/ws?conv_id=…&profile=…` → websocket stream
* `GET /v1/timeline?conv_id=…&since_version=…&limit=…` → hydration snapshot

Optional but useful (and already exists in your other project):

* `POST /v1/cancel` → cancel running inference (only if you support cancel)
* debug endpoints under `/v1/debug/*` behind an option

### B) Host integration surface (Go API)

A very small set of constructors + interfaces so host apps can plug in what they need.

**Recommend exported interfaces** (minimal):

* `RequestResolver` (your current `EngineFromReqBuilder` is basically this)
* `EngineBuilder` (you already have this as an interface)
* `TimelineStore` (already present)
* `SubscriberFactory` (already present conceptually)
* `ProfileRegistry` (already present)

Everything else (conversation state, queue details, connection pool internals, translator caches) should become internal.

---

## 3) Proposed architecture: split into 3 reusable layers

### Layer 1: “Core” runtime (`Service`)

Owns *conversations and streaming*, not HTTP.

Responsibilities:

* conversation lifecycle + eviction
* stream coordinator per conversation
* run strategy (queue vs reject) per conversation
* timeline projection integration (optional)

### Layer 2: “API transport” (`APIHandler`)

Owns the HTTP routes and websocket upgrade, nothing else.

Responsibilities:

* parse requests
* call `Service.GetOrCreateConversation(...)`
* call `Service.StartRun(...)`
* attach/detach websocket connections

### Layer 3: “UI transport” (`UIHandler`) optional

Serves static assets (your embedded web UI), but is fully optional and mountable.

Responsibilities:

* `GET /` -> index.html
* static assets routing
* optional base path rewriting (if you choose to support non-root UI)

This split is the single biggest cleanup win.

---

## 4) A concrete Go API you can standardize on

### 4.1 `Service` (host-app reusable core)

```go
type Options struct {
  Profiles ProfileRegistry
  EngineBuilder EngineBuilder

  // Event bus integration
  SubscriberFactory SubscriberFactory // per-conv
  Publisher message.Publisher         // for sinks

  // Optional
  TimelineStore TimelineStore
  IdleTimeout time.Duration
  EvictInterval time.Duration
  EvictWhen func(*Conversation) bool  // optional override

  // Request policy
  RequestResolver RequestResolver // optional; default provided
}

type Service struct { ... }

func NewService(opts Options) (*Service, error)
func (s *Service) GetOrCreate(ctx context.Context, convID, profile string, overrides map[string]any) (*ConversationHandle, error)
func (s *Service) StartRun(ctx context.Context, convID string, req RunRequest) (RunResponse, error)
func (s *Service) AddConn(convID string, conn *websocket.Conn)
func (s *Service) RemoveConn(convID string, conn *websocket.Conn)
```

**Key point:** Host apps can now embed the service and mount any HTTP stack.

### 4.2 `APIHandler` (mountable HTTP handler)

```go
type APIConfig struct {
  BasePath string // e.g. "/api/webchat" or "/v1"
  EnableDebug bool
  EnableProfileCookie bool
  WSUpgrader websocket.Upgrader
}

func NewAPIHandler(svc *Service, cfg APIConfig) http.Handler
```

Then mounting is always:

```go
mux.Handle("/api/webchat/", http.StripPrefix("/api/webchat", webchat.NewAPIHandler(svc, ...)))
```

This single rule solves your “I can’t integrate” problem.

### 4.3 `UIHandler` (optional)

```go
type UIConfig struct {
  BasePath string // for <base href> injection or runtime config endpoint
}

func NewUIHandler(uiFS fs.FS, cfg UIConfig) http.Handler
```

You mount UI separately from API, in any topology you want:

* same server, same base path
* separate origin
* behind a reverse proxy

---

## 5) ConversationManager: copy the *shape* from Mento, adapt the internals

Your Mento refactor has the right separation:

* `ConversationManager` owns the `map[convID]*Conversation`
* `GetOrCreate(...)` compares engine signature and rebuilds stream/engine if needed
* `OnConnectionAdded()` ensures stream running
* eviction loop cleans up idle conversations

Pinocchio already has most of this logic, but it’s split across `Router.getOrCreateConv`, `Conversation`, `ConnectionPool`, and `Server`.

### What to change in Pinocchio:

1. Move `getOrCreateConv` into a `ConversationManager` type
   (like Mento’s `conversation_manager.go`)
2. Move `addConn/removeConn` into manager methods
3. Add eviction loop (very important if `conv_id` can be random UUIDs)
4. Keep queue/idempotency, but make it **a conversation-level internal module** (not in router.go)

**Eviction rule** (simple and safe):

* evict if:

  * no connections
  * stream not running
  * no run running / no queue running
  * last activity older than N (optional)

---

## 6) Websocket streaming: what to standardize and what to fix

### 6.1 Standardize handshake + keepalive

Keep your existing behavior (it’s already close to Mento):

* On connect, server sends a `ws.hello` SEM event with:

  * `conv_id`
  * `profile`
  * `server_time_ms`
  * `protocol_version` (add this)
  * `capabilities`: `{"timeline": true/false, "queue": true/false}`

* Keep the lightweight `ws.ping` → `ws.pong` for the web UI (easy in JS).

Optionally later: add real websocket ping control frames for more robust idle detection.

### 6.2 Fix ordering/versioning using the Mento approach

This is the biggest “proper WS streaming” insight in your other project:

* If events come from Redis streams, you can read the `xid` / stream ID from Watermill message metadata (`xid`, `redis_xid`, etc).
* Convert it to a monotonic sequence/version.
* Thread that into your outgoing SEM envelope as:

  * `event.stream_id` (string)
  * `event.seq` (monotonic integer)

In Pinocchio today:

* `StreamCoordinator` uses a local `atomic.Uint64` and adds `seq`.
* It also adds `stream_id` if present, but `seq` is not tied to stream ID.

**Proposed behavior:**

* if Redis stream ID is present, `seq = derivedFromStreamID`
* else fallback to `atomic++`

This keeps ordering stable across:

* restarts
* multiple consumers
* potential multi-instance scenarios

### 6.3 Make ConnectionPool non-blocking (optional but very reusable)

Right now your `ConnectionPool.Broadcast` writes to every connection under a mutex. That’s fine for small scale, but a slow client can stall broadcasts.

A reusable pattern is:

* on Add, create a `client` with a buffered `send chan []byte`
* start 1 goroutine per client for writing
* Broadcast tries to enqueue; if channel full, drop client or drop message

This change is internal, does not expand your public API, and makes the backend “library-safe”.

---

## 7) Timeline/hydration: make it a first-class optional module

Pinocchio’s timeline pieces are *good*, but wiring is scattered:

* `TimelineStore` + `/timeline` endpoint
* `TimelineProjector.ApplySemFrame` called from stream callback
* `timeline.upsert` pushed over websocket

To make this reusable:

### A) Define a small “hydration contract”

* HTTP snapshot API: `GET /v1/timeline`
* WS incremental: `timeline.upsert` events

### B) Keep projector/store internal

Expose only:

* `TimelineStore` interface (already)
* `WithTimelineStore(store)` option on `Service`

Everything else stays private.

### C) Consider adding “version hints” (if you want multi-instance correctness)

Mento’s hydration path uses a version derived from Redis stream ID.

If you ever want Pinocchio’s timeline store to be correct across multiple writers, consider a *non-breaking extension*:

```go
type TimelineStoreV2 interface {
  UpsertWithVersion(ctx, convID string, entity *TimelineEntityV1, versionHint uint64) (uint64, error)
}
```

Then your projector can do:

* if store implements `TimelineStoreV2` and you have a derived seq/version → use it
* else fallback to existing `Upsert`

This preserves your current simple stores but enables “serious” deployments.

---

## 8) Request policy / auth: formalize it as a single interface

You already introduced `EngineFromReqBuilder` in Pinocchio. That’s basically the right seam.

Rename conceptually to something like `RequestResolver` or `Policy`:

```go
type RequestResolver interface {
  ResolveWS(req *http.Request) (WSRequest, error)
  ResolveChat(req *http.Request) (ChatRequest, error)
}
```

Where the resolved request includes:

* conv_id
* profile
* overrides
* identity info (optional; could be extracted from context)

Then your API handlers do **no policy logic**. They call the resolver, and that’s it.

This is how you make it reusable in apps with different auth:

* “my app already has user session in context”
* “I want conv_id derived from user+workspace”
* “I want to forbid overrides except for admins”
* etc.

---

## 9) Proposed folder layout (clean, reusable, testable)

You can do this within the Go module without changing external behavior initially:

```
pkg/webchat/
  service/
    service.go               // Service + Options
    conversation_manager.go   // GetOrCreate, eviction
    conversation.go           // Conversation state, run queue
    run_queue.go              // idempotency + queue semantics
    stream_coordinator.go     // consume + cursor/version derivation
    connection_pool.go        // pool + per-client writer loops (optional)
    timeline/
      projector.go
      store.go
      store_memory.go
      store_sqlite.go

  transport/
    httpapi/
      handler.go              // mounts routes to sub-mux
      chat.go
      ws.go
      profiles.go
      timeline.go
      debug.go                // optional
    ui/
      handler.go              // static assets only, optional

  protocol/                   // optional but recommended
    sem_envelope.go           // helpers for {sem:true,event:{...}}
    constants.go              // event type constants
```

**Important:** make `service` the reusable core. `transport/httpapi` is just one way to expose it.

---

## 10) A migration plan that won’t break everything at once

### Step 1: Fix mounting (high impact, low risk)

Change `Router.Mount` to use `http.StripPrefix`, and add a redirect for the no-trailing-slash base path.

This makes embedding immediately viable.

### Step 2: Split UI from API (still low risk)

* Add `NewAPIHandler(...)` that registers only `/chat`, `/ws`, `/timeline`, `/profiles`
* Add `NewUIHandler(...)` separately
* Keep your existing `NewRouter` but implement it by composing these two

### Step 3: Extract ConversationManager

Move `getOrCreateConv`, `addConn/removeConn`, and stream wiring out of router into `ConversationManager`.

Router becomes thin:

* resolve request
* call manager/service
* return JSON / upgrade websocket

### Step 4: Move run/queue logic out of router.go

`startRunForPrompt`, `finishRun`, `tryDrainQueue` become methods on a `Runner` owned by the conversation or service.

### Step 5: Stream ordering improvement (borrow from Mento)

Adopt the “derive seq from Redis stream ID” behavior.
This doesn’t change your UI contract other than making `seq` more meaningful.

### Step 6: Add eviction loop

This prevents memory growth with random conv IDs and makes the package safe to run long-lived inside other apps.

---

## 11) What your NPM webchat UI should depend on (backend-wise)

To keep the backend reusable across different host apps, the UI package should only assume:

* A base URL
* These endpoints (relative to base):

  * `POST /v1/chat`
  * `GET /v1/ws`
  * `GET /v1/timeline`
  * `GET /v1/profiles` (optional; only if UI needs it)

And the websocket SEM envelope format.

Everything else (cookies, debug endpoints, internal fields) should be optional.

---

## 12) The “don’t regret it later” backend checklist

* ✅ API and WS endpoints are versioned under `/v1`
* ✅ Can mount under any base path via `StripPrefix`
* ✅ UI serving is optional and separate from API
* ✅ Conversation lifecycle is in a manager (not in HTTP handlers)
* ✅ Eviction loop exists (no memory leak with random conv ids)
* ✅ Stream ordering uses Redis stream IDs when available (Mento insight)
* ✅ Request policy/auth is one interface, not scattered logic
* ❌ Don’t make 20 config options; expose a small `Options` + a few interfaces

---

If you want, I can take what exists in `pinocchio/pkg/webchat` and map it 1:1 into the proposed “Service + APIHandler + UIHandler” modules (i.e., a refactor blueprint listing which existing functions/types move where and what the new minimal exported API would be).


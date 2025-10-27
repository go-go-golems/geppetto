package structuredsink

import (
    "context"
    "regexp"
    "sort"
    "strings"
    "sync"

    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/google/uuid"
    "github.com/pkg/errors"
    "github.com/rs/zerolog/log"
    "gopkg.in/yaml.v3"
)

// Options control the filtering sink behavior.
type Options struct {
    EmitRawDeltas       bool
    EmitParsedSnapshots bool
    MaxCaptureBytes     int
    AcceptFenceLangs    []string
    OnMalformed         string // "ignore" | "forward-raw" | "error-events"
    Debug               bool   // if true, emit zerolog debug traces
}

func (o *Options) withDefaults() Options {
    ret := *o
    if ret.AcceptFenceLangs == nil || len(ret.AcceptFenceLangs) == 0 {
        ret.AcceptFenceLangs = []string{"yaml", "yml"}
    }
    if ret.OnMalformed == "" {
        ret.OnMalformed = "error-events"
    }
    return ret
}

// Extractor defines a typed extractor registered for a specific <$name:$dataType> pair.
type Extractor interface {
    Name() string
    DataType() string
    NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession
}

// ExtractorSession receives streaming callbacks and returns typed events to publish.
type ExtractorSession interface {
    OnStart(ctx context.Context) []events.Event
    OnDelta(ctx context.Context, raw string) []events.Event
    OnUpdate(ctx context.Context, snapshot map[string]any, parseErr error) []events.Event
    OnCompleted(ctx context.Context, final map[string]any, success bool, err error) []events.Event
}

// FilteringSink wraps a downstream EventSink and filters structured data blocks
// marked with <$name:$dtype> ```yaml ... ``` </$name:$dtype> from text streams,
// while emitting per-extractor typed events via the same downstream sink.
type FilteringSink struct {
    next       events.EventSink
    opts       Options
    exByKey    map[string]Extractor // key: name+"\x00"+dtype
    mu         sync.Mutex
    byStreamID map[uuid.UUID]*streamState
    baseCtx    context.Context
}

func NewFilteringSink(next events.EventSink, opts Options, extractors ...Extractor) *FilteringSink {
    o := opts.withDefaults()
    ex := make(map[string]Extractor)
    for _, e := range extractors {
        key := extractorKey(e.Name(), e.DataType())
        ex[key] = e
    }
    return &FilteringSink{
        next:       next,
        opts:       o,
        exByKey:    ex,
        byStreamID: make(map[uuid.UUID]*streamState),
        baseCtx:    context.Background(),
    }
}

// NewFilteringSinkWithContext is like NewFilteringSink but allows specifying a base context
// used to derive per-stream and per-item contexts that get cancelled on completion.
func NewFilteringSinkWithContext(ctx context.Context, next events.EventSink, opts Options, extractors ...Extractor) *FilteringSink {
    if ctx == nil {
        ctx = context.Background()
    }
    o := opts.withDefaults()
    ex := make(map[string]Extractor)
    for _, e := range extractors {
        key := extractorKey(e.Name(), e.DataType())
        ex[key] = e
    }
    return &FilteringSink{
        next:       next,
        opts:       o,
        exByKey:    ex,
        byStreamID: make(map[uuid.UUID]*streamState),
        baseCtx:    ctx,
    }
}

var _ events.EventSink = (*FilteringSink)(nil)

func extractorKey(name, dtype string) string { return name + "\x00" + dtype }

// Parser state per message stream
type streamState struct {
    id        uuid.UUID
    // contexts: stream-scoped and current item-scoped
    ctx       context.Context
    cancel    context.CancelFunc
    itemCtx   context.Context
    itemCancel context.CancelFunc
    // track full raw text seen so far (to avoid double-appending on final)
    rawSeen   strings.Builder
    // output buffer state for completion consistency
    filteredCompletion strings.Builder
    // small carry for boundary detection across partials
    carry string

    // capture state
    capturing bool
    name      string
    dtype     string
    seq       int
    session   ExtractorSession
    yamlBuf   strings.Builder

    // sub-state buffers for partial pattern matches
    openTagBuf   strings.Builder
    fenceBuf     strings.Builder
    closeTagBuf  strings.Builder
    inFence      bool
    fenceOpened  bool
    fenceLangOK  bool
    awaitingCloseTag bool
}

// PublishEvent implements events.EventSink. It intercepts partial/final text events
// and forwards filtered variants while publishing extractor-defined typed events.
func (f *FilteringSink) PublishEvent(ev events.Event) error {
    switch ev.Type() {
    case events.EventTypePartialCompletion:
        return f.handlePartial(ev)
    case events.EventTypeFinal:
        return f.handleFinal(ev)
    default:
        // pass-through
        return f.next.PublishEvent(ev)
    }
}

func (f *FilteringSink) getState(meta events.EventMetadata) *streamState {
    f.mu.Lock()
    defer f.mu.Unlock()
    st, ok := f.byStreamID[meta.ID]
    if !ok {
        st = &streamState{id: meta.ID}
        st.ctx, st.cancel = context.WithCancel(f.baseCtx)
        f.byStreamID[meta.ID] = st
    }
    return st
}

func (f *FilteringSink) deleteState(meta events.EventMetadata) {
    f.mu.Lock()
    defer f.mu.Unlock()
    if st, ok := f.byStreamID[meta.ID]; ok {
        if st.itemCancel != nil { st.itemCancel(); st.itemCancel = nil }
        if st.cancel != nil { st.cancel() }
        delete(f.byStreamID, meta.ID)
    }
}

func (f *FilteringSink) publishAll(ctx context.Context, meta events.EventMetadata, list []events.Event) error {
    for _, e := range list {
        if e == nil { continue }
        // Ensure metadata is set if missing
        impl, ok := e.(*events.EventImpl)
        if ok {
            // keep Type_ and payload; only set meta if zero UUID
            if impl.Metadata_.ID == uuid.Nil {
                impl.Metadata_.ID = meta.ID
            }
            if impl.Metadata_.RunID == "" { impl.Metadata_.RunID = meta.RunID }
            if impl.Metadata_.TurnID == "" { impl.Metadata_.TurnID = meta.TurnID }
        }
        if err := f.next.PublishEvent(e); err != nil {
            return err
        }
    }
    return nil
}

// --- streaming handlers ---

func (f *FilteringSink) handlePartial(ev events.Event) error {
    var (
        delta string
        meta  events.EventMetadata
        ok    bool
    )
    // Prefer typed partials
    if pcTyped, isTyped := ev.(*events.EventPartialCompletion); isTyped {
        delta = pcTyped.Delta
        meta = pcTyped.Metadata()
        ok = true
    } else if impl, isImpl := ev.(*events.EventImpl); isImpl {
        // Try decode payload-based
        pc, decOK := impl.ToPartialCompletion()
        if decOK {
            delta = pc.Delta
            meta = pc.Metadata()
            ok = true
        }
    }
    if !ok {
        // pass-through on unknown
        return f.next.PublishEvent(ev)
    }

    st := f.getState(meta)
    if f.opts.Debug { log.Debug().Str("stream", meta.ID.String()).Str("event", "partial").Str("delta", delta).Msg("incoming") }
    st.rawSeen.WriteString(delta)

    filteredDelta, typedEvents := f.scanAndFilter(meta, st, delta)

    // Maintain filtered completion consistency
    st.filteredCompletion.WriteString(filteredDelta)

    // Forward filtered partial only if there is any delta (including empty is okay to preserve timing)
    fwd := &events.EventPartialCompletion{
        EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta},
        Delta:     filteredDelta,
        Completion: st.filteredCompletion.String(),
    }
    if err := f.next.PublishEvent(fwd); err != nil {
        return err
    }
    // Publish extractor-typed events
    return f.publishAll(st.ctx, meta, typedEvents)
}

func (f *FilteringSink) handleFinal(ev events.Event) error {
    // Handle typed EventFinal directly first
    if fe, ok := ev.(*events.EventFinal); ok {
        meta := fe.Metadata()
        st := f.getState(meta)
        // Process only the raw tail we haven't seen via partials
        full := fe.Text
        prefix := st.rawSeen.String()
        delta := full
        if strings.HasPrefix(full, prefix) {
            delta = full[len(prefix):]
        }
        if f.opts.Debug { log.Debug().Str("stream", meta.ID.String()).Str("event", "final").Int("raw_seen_len", len(prefix)).Int("full_len", len(full)).Int("tail_len", len(delta)).Msg("incoming") }
        filtered, typed := f.scanAndFilter(meta, st, delta)
        st.filteredCompletion.WriteString(filtered)
        _ = f.publishAll(st.ctx, meta, typed)
        out := events.NewFinalEvent(meta, st.filteredCompletion.String())
        f.deleteState(meta)
        return f.next.PublishEvent(out)
    }

    // If we have a raw EventImpl, try legacy EventText extraction
    if fin, ok := ev.(*events.EventImpl); ok {
        meta := fin.Metadata()
        st := f.getState(meta)
        tf, isText := fin.ToText()
        if isText {
            full := tf.Text
            prefix := st.rawSeen.String()
            delta := full
            if strings.HasPrefix(full, prefix) {
                delta = full[len(prefix):]
            }
            if f.opts.Debug { log.Debug().Str("stream", meta.ID.String()).Str("event", "final-text").Int("raw_seen_len", len(prefix)).Int("full_len", len(full)).Int("tail_len", len(delta)).Msg("incoming") }
            filtered, typed := f.scanAndFilter(meta, st, delta)
            st.filteredCompletion.WriteString(filtered)
            _ = f.publishAll(st.ctx, meta, typed)
            out := events.NewFinalEvent(meta, st.filteredCompletion.String())
            f.deleteState(meta)
            return f.next.PublishEvent(out)
        }
    }

    // Unknown final—pass-through
    return f.next.PublishEvent(ev)
}

// scanAndFilter processes incoming delta, updating state and returning
// - filteredDelta to forward to UI
// - typed events to publish based on extractor sessions
func (f *FilteringSink) scanAndFilter(meta events.EventMetadata, st *streamState, delta string) (string, []events.Event) {
    // We'll stream-process characters while maintaining small pattern buffers.
    var out strings.Builder
    var typed []events.Event

    for i := 0; i < len(delta); i++ {
        ch := delta[i]

        if !st.capturing {
            // Accumulate potential open tag if a '<' appeared previously or now
            if st.openTagBuf.Len() > 0 || ch == '<' {
                st.openTagBuf.WriteByte(ch)
                // Decide if we have a full valid tag
                if tryParseOpenTag(st, st.openTagBuf.String()) {
                    // fully parsed and valid; enter capture state
                    st.capturing = true
                    st.seq++
                    st.yamlBuf.Reset()
                    st.fenceBuf.Reset()
                    st.closeTagBuf.Reset()
                    st.inFence = false
                    st.fenceOpened = false
                    st.fenceLangOK = false
                    st.awaitingCloseTag = false

                    // Emit OnStart
                    if ex := f.exByKey[extractorKey(st.name, st.dtype)]; ex != nil {
                        // derive per-item context from the stream context
                        if st.itemCancel != nil { st.itemCancel(); st.itemCancel = nil }
                        st.itemCtx, st.itemCancel = context.WithCancel(st.ctx)
                        st.session = ex.NewSession(st.itemCtx, meta, itemID(meta.ID, st.seq))
                        typed = append(typed, st.session.OnStart(st.itemCtx)...)
                    } else {
                        // Unknown extractor: treat as not capturing (flush buffer)
                        out.WriteString(st.openTagBuf.String())
                        st.openTagBuf.Reset()
                        st.capturing = false
                    }
                    continue
                }
                // If it becomes clear this is not a tag, flush buffered text
                if st.openTagBuf.Len() >= 2 && !strings.HasPrefix(st.openTagBuf.String(), "<$") {
                    out.WriteString(st.openTagBuf.String())
                    st.openTagBuf.Reset()
                }
                continue
            }

            // Normal text (no tag buffering in progress)
            out.WriteByte(ch)
            continue
        }

        // Capturing branch
        if !st.fenceOpened {
            st.fenceBuf.WriteByte(ch)
            if tryDetectFenceOpen(&st.fenceBuf, f.opts.AcceptFenceLangs) {
                st.fenceOpened = true
                st.inFence = true
                st.fenceLangOK = true
            } else if st.fenceBuf.Len() > 6 && !strings.Contains(st.fenceBuf.String(), "```") {
                // No fence realistically; malformed—flush and reset by policy
                flushMalformed(f, meta, st, &out, &typed)
            }
            continue
        }

        if st.inFence {
            // Collect YAML until closing fence
            st.yamlBuf.WriteByte(ch)
            if strings.HasSuffix(st.yamlBuf.String(), "```") {
                // remove the trailing fence from yamlBuf
                s := st.yamlBuf.String()
                if len(s) >= 3 { s = s[:len(s)-3] }
                st.yamlBuf.Reset()
                st.yamlBuf.WriteString(s)
                // emit accumulated delta without the closing fence
                if f.opts.EmitRawDeltas && st.session != nil {
                    typed = append(typed, st.session.OnDelta(st.itemCtx, s)...)
                }
                // parse snapshot on close as well
                snapshot, perr := parseYAML(s)
                if st.session != nil {
                    if f.opts.EmitParsedSnapshots {
                        typed = append(typed, st.session.OnUpdate(st.itemCtx, snapshot, perr)...)
                    }
                    // Completed will be emitted once close tag is confirmed; switch state
                }
                st.inFence = false
                st.awaitingCloseTag = true
                st.closeTagBuf.Reset()
            } else {
                // mid-yaml: optionally emit delta/snapshot
                if f.opts.EmitRawDeltas && st.session != nil {
                    typed = append(typed, st.session.OnDelta(st.itemCtx, string(ch))...)
                }
                if st.session != nil && f.opts.EmitParsedSnapshots {
                    snapshot, perr := parseYAML(st.yamlBuf.String())
                    typed = append(typed, st.session.OnUpdate(st.itemCtx, snapshot, perr)...)
                }
            }
            continue
        }

        if st.awaitingCloseTag {
            st.closeTagBuf.WriteByte(ch)
            if tryParseCloseTag(st, st.closeTagBuf.String()) {
                // finalize
                finalSnap, perr := parseYAML(st.yamlBuf.String())
                if st.session != nil {
                    typed = append(typed, st.session.OnCompleted(st.itemCtx, finalSnap, perr == nil, perr)...)
                }
                // reset state to idle
                st.capturing = false
                st.name, st.dtype = "", ""
                st.session = nil
                if st.itemCancel != nil { st.itemCancel(); st.itemCancel = nil }
                st.yamlBuf.Reset()
                st.openTagBuf.Reset()
                st.fenceBuf.Reset()
                st.closeTagBuf.Reset()
                st.inFence = false
                st.fenceOpened = false
                st.awaitingCloseTag = false
            }
            continue
        }
    }

    // At the end of delta, flush any non-capturing buffers
    if !st.capturing && st.openTagBuf.Len() > 0 {
        out.WriteString(st.openTagBuf.String())
        st.openTagBuf.Reset()
    }

    return out.String(), coalesce(typed)
}

func flushMalformed(f *FilteringSink, meta events.EventMetadata, st *streamState, out *strings.Builder, typed *[]events.Event) {
    switch f.opts.OnMalformed {
    case "ignore":
        // drop everything captured so far
    case "forward-raw":
        // write back the buffered pieces
        out.WriteString(st.openTagBuf.String())
        out.WriteString(st.fenceBuf.String())
        out.WriteString(st.yamlBuf.String())
        out.WriteString(st.closeTagBuf.String())
    case "error-events":
        if st.session != nil {
            *typed = append(*typed, st.session.OnCompleted(st.itemCtx, nil, false, errors.New("malformed structured block"))...)
        }
    }
    // reset state
    st.capturing = false
    st.name, st.dtype = "", ""
    st.session = nil
    if st.itemCancel != nil { st.itemCancel(); st.itemCancel = nil }
    st.yamlBuf.Reset()
    st.openTagBuf.Reset()
    st.fenceBuf.Reset()
    st.closeTagBuf.Reset()
    st.inFence = false
    st.fenceOpened = false
    st.awaitingCloseTag = false
}

func itemID(id uuid.UUID, seq int) string {
    return id.String() + ":" + itoa(seq)
}

// itoa converts int to string via local strconv without importing fmt. Keep simple.
func itoa(i int) string { return strconv(i) }

// simple int to string (without importing fmt), deterministic
func strconv(i int) string {
    if i == 0 { return "0" }
    neg := false
    if i < 0 { neg = true; i = -i }
    var b [20]byte
    pos := len(b)
    for i > 0 {
        pos--
        b[pos] = byte('0' + i%10)
        i /= 10
    }
    if neg { pos--; b[pos] = '-' }
    return string(b[pos:])
}

var (
    reOpen = regexp.MustCompile(`^<\$([a-zA-Z0-9_-]+):([a-zA-Z0-9._-]+)>$`)
)

func tryParseOpenTag(st *streamState, s string) bool {
    if !strings.HasPrefix(s, "<$") {
        return false
    }
    if strings.HasSuffix(s, ">") {
        m := reOpen.FindStringSubmatch(s)
        if len(m) == 3 {
            st.name = m[1]
            st.dtype = m[2]
            st.openTagBuf.Reset()
            return true
        }
        // invalid tag
        return false
    }
    return false
}

func tryDetectFenceOpen(buf *strings.Builder, langs []string) bool {
    s := buf.String()
    // look for ```lang\n or ```\n (lang optional, must be accepted if present)
    idx := strings.Index(s, "```")
    if idx < 0 { return false }
    rest := s[idx+3:]
    // require newline to finalize header line to avoid capturing header as YAML
    lineEnd := strings.IndexByte(rest, '\n')
    if lineEnd < 0 { return false }
    header := rest[:lineEnd]
    header = strings.TrimSpace(header)
    if header == "" { return true }
    // check language
    normalized := strings.ToLower(header)
    sort.Strings(langs)
    i := sort.SearchStrings(langs, normalized)
    return i < len(langs) && langs[i] == normalized
}

func tryParseCloseTag(st *streamState, s string) bool {
    // expecting </$name:dtype>
    if !strings.HasSuffix(s, ">") { return false }
    exp := "</$" + st.name + ":" + st.dtype + ">"
    if strings.HasSuffix(s, exp) {
        st.closeTagBuf.Reset()
        return true
    }
    return false
}

func parseYAML(s string) (map[string]any, error) {
    var v map[string]any
    if strings.TrimSpace(s) == "" { return nil, errors.New("empty") }
    err := yaml.Unmarshal([]byte(s), &v)
    if err != nil { return nil, err }
    return v, nil
}

func coalesce(list []events.Event) []events.Event {
    if len(list) == 0 { return nil }
    return list
}



package structuredsink

import (
    "context"
    "fmt"
    "strings"
    "testing"
    "time"

    "github.com/go-go-golems/geppetto/pkg/events"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type recordingSink struct {
    list []events.Event
}

func (r *recordingSink) PublishEvent(ev events.Event) error {
    r.list = append(r.list, ev)
    return nil
}

func TestFilteringSink_PassThrough_NoStructured(t *testing.T) {
    rec := &recordingSink{}
    sink := NewFilteringSink(rec, Options{})

    id := uuid.New()
    meta := events.EventMetadata{ID: id, RunID: "run-1", TurnID: "turn-1"}

    // Two partials and a final without any structured tags
    in := []events.Event{
        events.NewPartialCompletionEvent(meta, "Hello, ", "Hello, "),
        events.NewPartialCompletionEvent(meta, "world!", "Hello, world!"),
        events.NewFinalEvent(meta, "Hello, world!"),
    }

    for _, ev := range in {
        require.NoError(t, sink.PublishEvent(ev))
    }

    // Expect same number of forwarded events
    require.Equal(t, len(in), len(rec.list))

    // Check partials preserved
    p1, ok := rec.list[0].(*events.EventPartialCompletion)
    require.True(t, ok)
    assert.Equal(t, "Hello, ", p1.Delta)
    assert.Equal(t, "Hello, ", p1.Completion)

    p2, ok := rec.list[1].(*events.EventPartialCompletion)
    require.True(t, ok)
    assert.Equal(t, "world!", p2.Delta)
    assert.Equal(t, "Hello, world!", p2.Completion)

    f, ok := rec.list[2].(*events.EventFinal)
    require.True(t, ok)
    assert.Equal(t, "Hello, world!", f.Text)
}

func TestFilteringSink_ContextLifecycle(t *testing.T) {
    rec := &recordingSink{}
    baseCtx, baseCancel := context.WithCancel(context.Background())
    defer baseCancel()
    sink := NewFilteringSinkWithContext(baseCtx, rec, Options{})

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    // First partial creates stream state and stream context
    require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "x", "x")))

    // Access internal state for this stream
    st := sink.getState(meta)
    require.NotNil(t, st)
    require.NotNil(t, st.ctx)

    // Context should not be cancelled yet
    select {
    case <-st.ctx.Done():
        t.Fatal("stream context cancelled too early")
    default:
    }

    // Final should cancel and remove state
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, "x")))

    // st.ctx must be cancelled shortly
    select {
    case <-st.ctx.Done():
        // ok
    case <-time.After(500 * time.Millisecond):
        t.Fatal("expected stream context to be cancelled after final")
    }

    // State removed from map
    sink.mu.Lock()
    _, ok := sink.byStreamID[id]
    sink.mu.Unlock()
    require.False(t, ok, "stream state should be deleted after final")
}

// --- test extractor emitting typed events via EventLog messages ---

type testExtractor struct{ name, dtype string }

func (te *testExtractor) Name() string    { return te.name }
func (te *testExtractor) DataType() string { return te.dtype }
func (te *testExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession {
    return &testSession{ctx: ctx, itemID: itemID}
}

type testSession struct{
    ctx context.Context
    itemID string
}

func (ts *testSession) OnStart(ctx context.Context) []events.Event {
    return []events.Event{events.NewLogEvent(events.EventMetadata{}, "info", "start:"+ts.itemID, nil)}
}
func (ts *testSession) OnDelta(ctx context.Context, raw string) []events.Event {
    return []events.Event{events.NewLogEvent(events.EventMetadata{}, "info", "delta:"+raw, nil)}
}
func (ts *testSession) OnUpdate(ctx context.Context, snapshot map[string]any, parseErr error) []events.Event {
    status := "ok"
    if parseErr != nil { status = "err" }
    return []events.Event{events.NewLogEvent(events.EventMetadata{}, "info", "update:"+status, nil)}
}
func (ts *testSession) OnCompleted(ctx context.Context, final map[string]any, success bool, err error) []events.Event {
    status := "success"
    if !success || err != nil { status = "fail" }
    return []events.Event{events.NewLogEvent(events.EventMetadata{}, "info", "completed:"+status, nil)}
}

func collectLogMessages(list []events.Event) []string {
    msgs := make([]string, 0)
    for _, ev := range list {
        if lg, ok := ev.(*events.EventLog); ok {
            msgs = append(msgs, lg.Message)
        }
    }
    return msgs
}

// ---- citations test extractor and typed events ----

type CitationItem struct {
    Title   string
    Authors []string
}

type EventCitationDelta struct {
    events.EventImpl
    ItemID string `json:"item_id"`
    Delta  string `json:"delta"`
}

type EventCitationUpdate struct {
    events.EventImpl
    ItemID  string         `json:"item_id"`
    Entries []CitationItem `json:"entries,omitempty"`
    Error   string         `json:"error,omitempty"`
}

type citationsExtractor struct{ name, dtype string }

func (ce *citationsExtractor) Name() string     { return ce.name }
func (ce *citationsExtractor) DataType() string { return ce.dtype }
func (ce *citationsExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession {
    return &citationsSession{ctx: ctx, itemID: itemID}
}

type citationsSession struct {
    ctx    context.Context
    itemID string
}

func (cs *citationsSession) OnStart(ctx context.Context) []events.Event {
    return nil
}

func (cs *citationsSession) OnDelta(ctx context.Context, raw string) []events.Event {
    return []events.Event{&EventCitationDelta{EventImpl: events.EventImpl{Type_: "citations-delta"}, ItemID: cs.itemID, Delta: raw}}
}

func (cs *citationsSession) OnUpdate(ctx context.Context, snapshot map[string]any, parseErr error) []events.Event {
    var entries []CitationItem
    hasInvalid := false
    if snapshot != nil {
        v, ok := snapshot["citations"]
        if !ok {
            // treat as invalid schema if citations key missing
            hasInvalid = true
        } else {
            list, ok := v.([]any)
            if !ok {
                hasInvalid = true
            } else {
                for _, it := range list {
                    m, ok := it.(map[string]any)
                    if !ok { hasInvalid = true; continue }
                    t, ok := m["title"].(string)
                    if !ok || t == "" { hasInvalid = true; continue }
                    rawAuthors, ok := m["authors"].([]any)
                    if !ok { hasInvalid = true; continue }
                    var authors []string
                    validAuthors := true
                    for _, au := range rawAuthors {
                        s, ok := au.(string)
                        if !ok || s == "" { validAuthors = false; break }
                        authors = append(authors, s)
                    }
                    if !validAuthors { hasInvalid = true; continue }
                    entries = append(entries, CitationItem{Title: t, Authors: authors})
                }
            }
        }
    }
    var errStr string
    if parseErr != nil { errStr = parseErr.Error() }
    if errStr == "" && hasInvalid { errStr = "invalid citations schema" }
    return []events.Event{&EventCitationUpdate{EventImpl: events.EventImpl{Type_: "citations-update"}, ItemID: cs.itemID, Entries: entries, Error: errStr}}
}

func (cs *citationsSession) OnCompleted(ctx context.Context, final map[string]any, success bool, err error) []events.Event {
    // Emit a final update snapshot
    return cs.OnUpdate(ctx, final, err)
}

func collectCitationUpdates(list []events.Event) []*EventCitationUpdate {
    var ups []*EventCitationUpdate
    for _, ev := range list {
        if u, ok := ev.(*EventCitationUpdate); ok {
            ups = append(ups, u)
        }
    }
    return ups
}

func collectCitationDeltas(list []events.Event) []*EventCitationDelta {
    var ds []*EventCitationDelta
    for _, ev := range list {
        if d, ok := ev.(*EventCitationDelta); ok { ds = append(ds, d) }
    }
    return ds
}

func TestFilteringSink_Citations_Incremental(t *testing.T) {
    rec := &recordingSink{}
    ex := &citationsExtractor{name: "citations", dtype: "v1"}
    sink := NewFilteringSink(rec, Options{EmitRawDeltas: true, EmitParsedSnapshots: true, Debug: false}, ex)

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    parts := []string{
        "prefix ",
        "<$citations:v1>",
        "```yaml\n",
        "citations:\n",
        "  - title: The Book\n",
        "    authors:\n",
        "      - Alice\n",
        "  - title: Another\n",
        "    authors:\n",
        "      - Bob\n",
        "```\n",
        "</$citations:v1>",
        " suffix",
    }
    completion := ""
    for _, p := range parts {
        completion += p
        require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, p, completion)))
    }
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, completion)))

    // Final text should have filtered block
    last := rec.list[len(rec.list)-1]
    fe, ok := last.(*events.EventFinal)
    require.True(t, ok)
    assert.Equal(t, "prefix  suffix", fe.Text)

    // Check updates show growth from 0->1->2 entries (we only assert max >= 2)
    ups := collectCitationUpdates(rec.list)
    require.NotEmpty(t, ups)
    maxLen := 0
    for _, u := range ups { if l := len(u.Entries); l > maxLen { maxLen = l } }
    assert.GreaterOrEqual(t, maxLen, 2)

    // Ensure at least one delta contains 'citations:' and a title
    ds := collectCitationDeltas(rec.list)
    found := false
    for _, d := range ds { if strings.Contains(d.Delta, "citations:") || strings.Contains(d.Delta, "title:") { found = true; break } }
    assert.True(t, found, "expected citation deltas to include YAML content")
}

func updatesForItem(list []events.Event, itemID string) []*EventCitationUpdate {
    var ret []*EventCitationUpdate
    for _, ev := range list {
        if u, ok := ev.(*EventCitationUpdate); ok {
            if u.ItemID == itemID { ret = append(ret, u) }
        }
    }
    return ret
}

func TestFilteringSink_Citations_Final_ValidateEntries(t *testing.T) {
    rec := &recordingSink{}
    ex := &citationsExtractor{name: "citations", dtype: "v1"}
    sink := NewFilteringSink(rec, Options{EmitRawDeltas: true, EmitParsedSnapshots: true}, ex)

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    yaml := "citations:\n  - title: Paper A\n    authors:\n      - Alice\n      - Bob\n  - title: Paper B\n    authors:\n      - Carol\n"
    final := fmt.Sprintf("X <$%s:%s>```yaml\n%s```\n</$%s:%s> Y", ex.name, ex.dtype, yaml, ex.name, ex.dtype)
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, final)))

    itemID := id.String()+":1"
    ups := updatesForItem(rec.list, itemID)
    require.NotEmpty(t, ups)
    last := ups[len(ups)-1]
    require.Len(t, last.Entries, 2)
    assert.Equal(t, "Paper A", last.Entries[0].Title)
    assert.Equal(t, []string{"Alice", "Bob"}, last.Entries[0].Authors)
    assert.Equal(t, "Paper B", last.Entries[1].Title)
    assert.Equal(t, []string{"Carol"}, last.Entries[1].Authors)
    assert.Equal(t, "", last.Error)
}

func TestFilteringSink_Citations_MalformedItems_DroppedAndFlagged(t *testing.T) {
    rec := &recordingSink{}
    ex := &citationsExtractor{name: "citations", dtype: "v1"}
    sink := NewFilteringSink(rec, Options{EmitRawDeltas: true, EmitParsedSnapshots: true}, ex)

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    yaml := "citations:\n  - title: Good One\n    authors:\n      - Alice\n  - title: Bad One\n    authors: Bob\n  - title: Bad Two\n    authors:\n      - 123\n"
    final := fmt.Sprintf("<$/%s:%s>", "invalid", "noop") // decoy not matching extractor
    _ = final
    text := fmt.Sprintf("pre <$%s:%s>```yaml\n%s```\n</$%s:%s> post", ex.name, ex.dtype, yaml, ex.name, ex.dtype)
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, text)))

    itemID := id.String()+":1"
    ups := updatesForItem(rec.list, itemID)
    require.NotEmpty(t, ups)
    last := ups[len(ups)-1]
    // Only the first valid entry remains
    require.Len(t, last.Entries, 1)
    assert.Equal(t, "Good One", last.Entries[0].Title)
    assert.Equal(t, []string{"Alice"}, last.Entries[0].Authors)
    // Error flagged due to malformed items
    assert.NotEqual(t, "", last.Error)
}


func TestFilteringSink_Structured_Final_Basic(t *testing.T) {
    rec := &recordingSink{}
    ex := &testExtractor{name: "x", dtype: "v1"}
    sink := NewFilteringSink(rec, Options{EmitRawDeltas: true, EmitParsedSnapshots: true}, ex)

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    // Send a preamble partial first
    require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "Intro ", "Intro ")))

    yaml := "a: 1\nb: 2\n"
    finalText := fmt.Sprintf("Intro <$%s:%s>```yaml\n%s```\n</$%s:%s> outro", ex.name, ex.dtype, yaml, ex.name, ex.dtype)

    // Publish final containing structured block
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, finalText)))

    // Expect: partial (Intro ), typed events (start, delta, update, completed), final with filtered text
    require.GreaterOrEqual(t, len(rec.list), 2)

    // Last must be final with filtered completion
    last := rec.list[len(rec.list)-1]
    fe, ok := last.(*events.EventFinal)
    require.True(t, ok)
    assert.Equal(t, "Intro  outro", fe.Text)

    // Gather log messages
    msgs := collectLogMessages(rec.list)
    // Ensure our extractor emitted its lifecycle messages
    assert.Contains(t, msgs, "start:"+id.String()+":1")
    assert.Contains(t, msgs, "update:ok")
    assert.Contains(t, msgs, "completed:success")
    // Delta contains yaml
    foundDelta := false
    for _, m := range msgs { if strings.HasPrefix(m, "delta:") && strings.Contains(m, "a: 1") { foundDelta = true; break } }
    assert.True(t, foundDelta, "expected a delta with YAML content")
}

func TestFilteringSink_Structured_PartialStream_Fragmented(t *testing.T) {
    rec := &recordingSink{}
    ex := &testExtractor{name: "x", dtype: "v1"}
    sink := NewFilteringSink(rec, Options{EmitRawDeltas: true, EmitParsedSnapshots: true}, ex)

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    // Stream fragmented parts across partials
    parts := []string{
        "Hello ",
        "<$x:v1>",
        "```yaml\n",
        "a: 1\n",
        "```\n",
        "</$x:v1>",
        " world",
    }
    completion := ""
    for _, p := range parts {
        completion += p
        require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, p, completion)))
    }
    // Finish
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, completion)))

    // Collect forwarded partials and final
    var deltas []string
    for _, ev := range rec.list {
        if pc, ok := ev.(*events.EventPartialCompletion); ok {
            deltas = append(deltas, pc.Delta)
        }
    }
    // The visible text should be "Hello  world" (structured removed)
    // We expect initial "Hello " delta, then empty during structured, then trailing " world"
    require.GreaterOrEqual(t, len(deltas), 2)
    assert.Equal(t, "Hello ", deltas[0])
    assert.Equal(t, " world", deltas[len(deltas)-1])

    last := rec.list[len(rec.list)-1]
    fe, ok := last.(*events.EventFinal)
    require.True(t, ok)
    assert.Equal(t, "Hello  world", fe.Text)

    // Ensure extractor emitted lifecycle
    msgs := collectLogMessages(rec.list)
    assert.Contains(t, msgs, "start:"+id.String()+":1")
    assert.Contains(t, msgs, "completed:success")
}

func TestFilteringSink_Structured_TwoBlocks_Final(t *testing.T) {
    rec := &recordingSink{}
    ex := &testExtractor{name: "x", dtype: "v1"}
    sink := NewFilteringSink(rec, Options{EmitRawDeltas: true, EmitParsedSnapshots: true}, ex)

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    block := func(kv string) string {
        return "<$x:v1>```yaml\n"+kv+"\n```\n</$x:v1>"
    }
    finalText := "pre "+block("a: 1")+" mid "+block("b: 2")+" post"
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, finalText)))

    // Final should filter both blocks
    last := rec.list[len(rec.list)-1]
    fe, ok := last.(*events.EventFinal)
    require.True(t, ok)
    assert.Equal(t, "pre  mid  post", fe.Text)

    // Extract start messages and assert seq endings :1 and :2
    msgs := collectLogMessages(rec.list)
    starts := make([]string, 0)
    for _, m := range msgs { if strings.HasPrefix(m, "start:") { starts = append(starts, m) } }
    require.Len(t, starts, 2)
    assert.Equal(t, "start:"+id.String()+":1", starts[0])
    assert.Equal(t, "start:"+id.String()+":2", starts[1])
}

func TestFilteringSink_Malformed_ForwardRaw(t *testing.T) {
    rec := &recordingSink{}
    ex := &testExtractor{name: "x", dtype: "v1"}
    sink := NewFilteringSink(rec, Options{OnMalformed: "forward-raw"}, ex)

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    finalText := "before <$x:v1> no-fence content then text </$x:v1> after"
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, finalText)))

    last := rec.list[len(rec.list)-1]
    fe, ok := last.(*events.EventFinal)
    require.True(t, ok)
    // With forward-raw, open tag was consumed and cannot be reconstituted, so it won't appear.
    assert.Equal(t, "before  no-fence content then text </$x:v1> after", fe.Text)

    // No completed:fail expected
    msgs := collectLogMessages(rec.list)
    for _, m := range msgs { if m == "completed:fail" { t.Fatalf("unexpected fail completion event") } }
}

func TestFilteringSink_Malformed_ErrorEvents(t *testing.T) {
    rec := &recordingSink{}
    ex := &testExtractor{name: "x", dtype: "v1"}
    sink := NewFilteringSink(rec, Options{OnMalformed: "error-events"}, ex)

    id := uuid.New()
    meta := events.EventMetadata{ID: id}

    finalText := "before <$x:v1> no-fence content then text </$x:v1> after"
    require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, finalText)))

    last := rec.list[len(rec.list)-1]
    fe, ok := last.(*events.EventFinal)
    require.True(t, ok)
    // The malformed captured region (starting after the tag) is dropped; closing tag leaks as plain text
    assert.Contains(t, fe.Text, "before ")
    assert.Contains(t, fe.Text, "</$x:v1> after")
    assert.NotContains(t, fe.Text, "no-fence content")

    msgs := collectLogMessages(rec.list)
    assert.Contains(t, msgs, "completed:fail")
}



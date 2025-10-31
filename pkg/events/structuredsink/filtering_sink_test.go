package structuredsink

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eventCollector is a simple sink that records all events forwarded downstream
type eventCollector struct{ list []events.Event }

func (c *eventCollector) PublishEvent(ev events.Event) error {
	c.list = append(c.list, ev)
	return nil
}

// testExtractor captures session lifecycle and chunks for assertions
type testExtractor struct {
	pkg, typ, ver string
	last          *testSession
	sessions      []*testSession
	mu            sync.Mutex
}

func (e *testExtractor) TagPackage() string { return e.pkg }
func (e *testExtractor) TagType() string    { return e.typ }
func (e *testExtractor) TagVersion() string { return e.ver }
func (e *testExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession {
	s := &testSession{ctx: ctx, itemID: itemID}
	e.mu.Lock()
	e.last = s
	e.sessions = append(e.sessions, s)
	e.mu.Unlock()
	return s
}

type testSession struct {
	ctx       context.Context
	itemID    string
	started   bool
	rawChunks []string
	completed bool
	finalRaw  string
	success   bool
}

func (s *testSession) OnStart(ctx context.Context) []events.Event {
	s.started = true
	return nil
}

func (s *testSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
	s.rawChunks = append(s.rawChunks, string(chunk))
	return nil
}

func (s *testSession) OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event {
	s.completed = true
	s.finalRaw = string(raw)
	s.success = success
	return nil
}

func newMeta() events.EventMetadata {
	return events.EventMetadata{ID: uuid.New(), RunID: "run", TurnID: "turn"}
}

func collectTextParts(list []events.Event) ([]string, string) {
	var partials []string
	var final string
	for _, ev := range list {
		switch v := ev.(type) {
		case *events.EventPartialCompletion:
			partials = append(partials, v.Delta)
		case *events.EventFinal:
			final = v.Text
		case *events.EventImpl:
			if pc, ok := v.ToPartialCompletion(); ok {
				partials = append(partials, pc.Delta)
				continue
			}
			if tf, ok := v.ToText(); ok {
				final = tf.Text
				continue
			}
		}
	}
	return partials, final
}

// feedParts is a helper to assemble streams from parts
func feedParts(t *testing.T, sink *FilteringSink, meta events.EventMetadata, parts []string) string {
	completion := ""
	for _, p := range parts {
		completion += p
		require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, p, completion)))
	}
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, completion)))
	return completion
}

func TestFilteringSink_CloseTagSinglePartial(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	full := "Hello <core:x:v1>abc</core:x:v1> world"

	// send as a single partial and then final
	_ = sink.PublishEvent(&events.EventPartialCompletion{EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta}, Delta: full})
	_ = sink.PublishEvent(events.NewFinalEvent(meta, full))

	// assert extractor saw correct lifecycle
	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	// With single partial, our lag buffer may keep all until close seen, so rawChunks may be empty; we rely on finalRaw
	assert.True(t, ex.last.completed)
	assert.True(t, ex.last.success)
	assert.Equal(t, "abc", ex.last.finalRaw)

	// assert filtered text
	partials, final := collectTextParts(col.list)
	require.NotEmpty(t, partials)
	assert.Contains(t, final, "Hello ")
	assert.Contains(t, final, " world")
	assert.NotContains(t, final, "<core:x:v1>")
	assert.NotContains(t, final, "</core:x:v1>")
	assert.NotContains(t, final, "abc")
}

func TestFilteringSink_CloseTagSplitAcrossPartials(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	p1 := "A<core:x:v1>abc</"
	p2 := "core:x:v1> Z"
	full := p1 + p2

	_ = sink.PublishEvent(&events.EventPartialCompletion{EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta}, Delta: p1})
	_ = sink.PublishEvent(&events.EventPartialCompletion{EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta}, Delta: p2})
	_ = sink.PublishEvent(events.NewFinalEvent(meta, full))

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.True(t, ex.last.success)
	assert.Equal(t, "abc", ex.last.finalRaw)
	// ensure no leakage of close tag bytes into chunks
	for _, c := range ex.last.rawChunks {
		assert.NotContains(t, c, "</$x:v1>")
		assert.NotContains(t, c, "</")
		assert.NotContains(t, c, "$x:v1>")
	}

	partials, final := collectTextParts(col.list)
	// expected filtered text contains only outside content
	assert.Equal(t, []string{"A", " Z"}, partials)
	assert.Equal(t, "A Z", final)
}

func TestFilteringSink_CloseTagBoundaryBeforeGt(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	p1 := "prefix <core:x:v1>abc</core:x:v1"
	p2 := "> suffix"
	full := p1 + p2

	_ = sink.PublishEvent(&events.EventPartialCompletion{EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta}, Delta: p1})
	_ = sink.PublishEvent(&events.EventPartialCompletion{EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta}, Delta: p2})
	_ = sink.PublishEvent(events.NewFinalEvent(meta, full))

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.completed)
	assert.True(t, ex.last.success)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix  suffix", final)
}

func TestFilteringSink_MalformedAtFinal_DefaultErrorEvents(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	p1 := "M <core:x:v1>abc"

	_ = sink.PublishEvent(&events.EventPartialCompletion{EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta}, Delta: p1})
	_ = sink.PublishEvent(events.NewFinalEvent(meta, p1))

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.completed)
	assert.False(t, ex.last.success)
	// final raw should include all captured bytes (payload + any withheld lag bytes)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	// default policy should not reinsert malformed block text
	assert.Equal(t, "M ", final)
}

func TestFilteringSink_UnknownExtractor_FlushesAsText(t *testing.T) {
	col := &eventCollector{}
	// register extractor for a different tag so this one is unknown
	ex := &testExtractor{pkg: "core", typ: "y", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	full := "X <core:unknown:v1>abc</core:unknown:v1> Y"

	_ = sink.PublishEvent(&events.EventPartialCompletion{EventImpl: events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta}, Delta: full})
	_ = sink.PublishEvent(events.NewFinalEvent(meta, full))

	// No extractor session should be created
	assert.Nil(t, ex.last)

	// All text should be forwarded untouched
	_, final := collectTextParts(col.list)
	assert.Equal(t, full, final)
}

// Test 1: Open-tag split matrix - complete set of boundaries
func TestFilteringSink_OpenTagSplit_BeforeDollar(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <", "core:x:v1>abc</core:x:v1> suffix"})

	require.NotNil(t, ex.last, "OnStart should fire exactly once once tag completes")
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.True(t, ex.last.success)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.NotContains(t, final, "<core:x:v1>", "No open-tag bytes should appear in forwarded text")
	assert.Equal(t, "prefix  suffix", final, "Filtered final should equal outside text only")
}

func TestFilteringSink_OpenTagSplit_AfterDollar(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:", "x:v1>abc</core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix  suffix", final)
}

func TestFilteringSink_OpenTagSplit_AfterName(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x", ":v1>abc</core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix  suffix", final)
}

func TestFilteringSink_OpenTagSplit_AfterColon(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:", "v1>abc</core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix  suffix", final)
}

func TestFilteringSink_OpenTagSplit_AfterDtype(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:v1", ">abc</core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix  suffix", final)
}

func TestFilteringSink_OpenTagSplit_MultipleFragments(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <", "core", ":", "x", ":", "v1", ">abc</core:x:v1> suffix"})

	require.NotNil(t, ex.last, "OnStart should fire exactly once once tag completes")
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.NotContains(t, final, "<core:x:v1>", "No open-tag bytes should appear in forwarded text")
	assert.Equal(t, "prefix  suffix", final)
}

// Test 2: Close-tag near-misses
func TestFilteringSink_CloseTagNearMiss_ExtraChar(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:v1>abc</core:x:v1!>middle</core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc</core:x:v1!>middle", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix  suffix", final)
}

func TestFilteringSink_CloseTagNearMiss_ExtraGt(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:v1>abc</core:x:v1>>middle</core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	// Close occurs on the first '>' completing the exact close tag; payload is just "abc"
	assert.Equal(t, "abc", ex.last.finalRaw)
	assert.True(t, ex.last.success)

	_, final := collectTextParts(col.list)
	// The extra '>' and the following text are outside the structured block
	assert.Equal(t, "prefix >middle</core:x:v1> suffix", final)
}

func TestFilteringSink_CloseTagNearMiss_SimilarPrefix(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:v1>abc</core:x:v1test></core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc</core:x:v1test>", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix  suffix", final)
}

// Test 3: Case sensitivity mismatch
func TestFilteringSink_CaseSensitivityMismatch(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "X", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:X:v1>abc</core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	// No close on mismatch; at final, OnCompleted(success=false, err) is called per policy
	assert.False(t, ex.last.success, "Should fail due to case mismatch")

	_, final := collectTextParts(col.list)
	// Default policy drops the captured region up to end-of-stream
	assert.Equal(t, "prefix ", final)
}

// Test 4: Empty payload
func TestFilteringSink_EmptyPayload(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:v1></core:x:v1> suffix"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.True(t, ex.last.success)
	assert.Equal(t, "", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix  suffix", final)
	assert.NotContains(t, final, "<core:x:v1>")
}

// Test 5: Back-to-back blocks without spacing
func TestFilteringSink_BackToBackBlocks(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"<core:x:v1>a</core:x:v1><core:x:v1>b</core:x:v1>"})

	require.Len(t, ex.sessions, 2)
	assert.True(t, ex.sessions[0].started)
	assert.True(t, ex.sessions[0].completed)
	assert.Equal(t, "a", ex.sessions[0].finalRaw)
	assert.Equal(t, meta.ID.String()+":1", ex.sessions[0].itemID)

	assert.True(t, ex.sessions[1].started)
	assert.True(t, ex.sessions[1].completed)
	assert.Equal(t, "b", ex.sessions[1].finalRaw)
	assert.Equal(t, meta.ID.String()+":2", ex.sessions[1].itemID)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "", final)
}

// Test 6: Interleaved blocks for different extractors
func TestFilteringSink_InterleavedDifferentExtractors(t *testing.T) {
	col := &eventCollector{}
	exA := &testExtractor{pkg: "core", typ: "a", ver: "v1"}
	exB := &testExtractor{pkg: "core", typ: "b", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, exA, exB)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"before <core:a:v1>payloadA</core:a:v1> mid <core:b:v1>payloadB</core:b:v1> after"})

	require.NotNil(t, exA.last)
	assert.True(t, exA.last.started)
	assert.True(t, exA.last.completed)
	assert.Equal(t, "payloadA", exA.last.finalRaw)

	require.NotNil(t, exB.last)
	assert.True(t, exB.last.started)
	assert.True(t, exB.last.completed)
	assert.Equal(t, "payloadB", exB.last.finalRaw)

	// Verify no cross-talk
	assert.NotContains(t, exA.last.finalRaw, "payloadB")
	assert.NotContains(t, exB.last.finalRaw, "payloadA")

	_, final := collectTextParts(col.list)
	assert.Equal(t, "before  mid  after", final)
}

// Test 7: Unknown extractor - split tag variant
func TestFilteringSink_UnknownExtractor_SplitTag(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "y", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"X <", "core:unknown:v1>abc</core:unknown:v1> Y"})

	assert.Nil(t, ex.last)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "X <core:unknown:v1>abc</core:unknown:v1> Y", final)
}

func TestFilteringSink_UnknownExtractor_SplitCloseTag(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "y", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"X <core:unknown:v1>abc</", "core:unknown:v1> Y"})

	assert.Nil(t, ex.last)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "X <core:unknown:v1>abc</core:unknown:v1> Y", final)
}

// Test 8: Malformed policies
func TestFilteringSink_MalformedPolicy_ErrorEvents(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{Malformed: MalformedErrorEvents}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"before <core:x:v1>payload"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.completed)
	assert.False(t, ex.last.success)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "before ", final)
	assert.NotContains(t, final, "payload")
}

func TestFilteringSink_MalformedPolicy_ForwardRaw(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{Malformed: MalformedReconstructText}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"before <core:x:v1>payload"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.completed)
	assert.False(t, ex.last.success)

	_, final := collectTextParts(col.list)
	assert.Contains(t, final, "before")
	assert.Contains(t, final, "<core:x:v1>")
	assert.Contains(t, final, "payload")
}

func TestFilteringSink_MalformedPolicy_Ignore(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{Malformed: MalformedIgnore}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"before <core:x:v1>payload"})

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.completed)
	assert.False(t, ex.last.success)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "before ", final)
	assert.NotContains(t, final, "payload")
}

// Test 9: Final-only inputs
func TestFilteringSink_FinalOnly_ValidBlock(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	full := "before <core:x:v1>abc</core:x:v1> after"
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, full)))

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.started)
	assert.True(t, ex.last.completed)
	assert.True(t, ex.last.success)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "before  after", final)
}

func TestFilteringSink_FinalOnly_Malformed(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	full := "before <core:x:v1>abc"
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, full)))

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.completed)
	assert.False(t, ex.last.success)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "before ", final)
}

func TestFilteringSink_FinalOnly_UnknownExtractor(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "y", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	full := "before <core:unknown:v1>abc</core:unknown:v1> after"
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, full)))

	assert.Nil(t, ex.last)

	_, final := collectTextParts(col.list)
	assert.Equal(t, full, final)
}

// Test 10: Metadata propagation for typed events
func TestFilteringSink_MetadataPropagation(t *testing.T) {
	col := &eventCollector{}
	metaExtractor := &metadataTestExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, metaExtractor)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:v1>abc</core:x:v1> suffix"})

	require.NotNil(t, metaExtractor.lastSession)
	assert.Len(t, metaExtractor.lastSession.emittedEvents, 1)

	ev := metaExtractor.lastSession.emittedEvents[0]
	impl, ok := ev.(*events.EventImpl)
	require.True(t, ok)
	assert.Equal(t, meta.ID, impl.Metadata_.ID)
	assert.Equal(t, meta.RunID, impl.Metadata_.RunID)
	assert.Equal(t, meta.TurnID, impl.Metadata_.TurnID)
}

type metadataTestExtractor struct {
	pkg, typ, ver string
	lastSession   *metadataTestSession
}

func (e *metadataTestExtractor) TagPackage() string { return e.pkg }
func (e *metadataTestExtractor) TagType() string    { return e.typ }
func (e *metadataTestExtractor) TagVersion() string { return e.ver }
func (e *metadataTestExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession {
	s := &metadataTestSession{ctx: ctx, itemID: itemID}
	e.lastSession = s
	return s
}

type metadataTestSession struct {
	ctx           context.Context
	itemID        string
	emittedEvents []events.Event
}

func (s *metadataTestSession) OnStart(ctx context.Context) []events.Event {
	// Return event with zero metadata
	ev := &events.EventImpl{
		Type_:     events.EventTypeLog,
		Metadata_: events.EventMetadata{}, // zero metadata
	}
	s.emittedEvents = append(s.emittedEvents, ev)
	return []events.Event{ev}
}

func (s *metadataTestSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
	return nil
}

func (s *metadataTestSession) OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event {
	return nil
}

// Test 11: Item context lifecycle (cancellation)
func TestFilteringSink_ItemContextCancellation(t *testing.T) {
	col := &eventCollector{}
	ctxExtractor := &contextTestExtractor{name: "core", dtype: "x"}
	sink := NewFilteringSinkWithContext(context.Background(), col, Options{}, ctxExtractor)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:v1>abc</core:x:v1> suffix"})

	require.NotNil(t, ctxExtractor.lastSession)
	assert.NotNil(t, ctxExtractor.lastSession.ctx)

	// Context should be canceled after completion
	select {
	case <-ctxExtractor.lastSession.ctx.Done():
		// Expected - context should be canceled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should have been canceled within 100ms")
	}
}

func TestFilteringSink_ItemContextCancellation_Malformed(t *testing.T) {
	col := &eventCollector{}
	ctxExtractor := &contextTestExtractor{name: "core", dtype: "x"}
	sink := NewFilteringSinkWithContext(context.Background(), col, Options{}, ctxExtractor)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"prefix <core:x:v1>abc"})

	require.NotNil(t, ctxExtractor.lastSession)
	assert.NotNil(t, ctxExtractor.lastSession.ctx)

	// Context should be canceled after malformed handling
	select {
	case <-ctxExtractor.lastSession.ctx.Done():
		// Expected - context should be canceled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should have been canceled within 100ms")
	}
}

type contextTestExtractor struct {
	name, dtype string
	lastSession *contextTestSession
}

func (e *contextTestExtractor) TagPackage() string { return e.name }
func (e *contextTestExtractor) TagType() string    { return e.dtype }
func (e *contextTestExtractor) TagVersion() string { return "v1" }
func (e *contextTestExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession {
	s := &contextTestSession{ctx: ctx, itemID: itemID}
	e.lastSession = s
	return s
}

type contextTestSession struct {
	ctx    context.Context
	itemID string
}

func (s *contextTestSession) OnStart(ctx context.Context) []events.Event {
	return nil
}

func (s *contextTestSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
	return nil
}

func (s *contextTestSession) OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event {
	return nil
}

// Test 12: Multiple streams interleaved
func TestFilteringSink_MultipleStreamsInterleaved(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	metaA := newMeta()
	metaB := newMeta()

	// Alternate partials between streams
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(metaA, "A1 <core:x:v1>", "A1 <core:x:v1>")))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(metaB, "B1 <core:x:v1>", "B1 <core:x:v1>")))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(metaA, "payloadA</core:x:v1> A2", "A1 <core:x:v1>payloadA</core:x:v1> A2")))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(metaB, "payloadB</core:x:v1> B2", "B1 <core:x:v1>payloadB</core:x:v1> B2")))

	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(metaA, "A1 <core:x:v1>payloadA</core:x:v1> A2")))
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(metaB, "B1 <core:x:v1>payloadB</core:x:v1> B2")))

	// Should have two sessions (one per stream)
	require.Len(t, ex.sessions, 2)

	// Verify stream A
	assert.Equal(t, metaA.ID.String()+":1", ex.sessions[0].itemID)
	assert.Equal(t, "payloadA", ex.sessions[0].finalRaw)

	// Verify stream B
	assert.Equal(t, metaB.ID.String()+":1", ex.sessions[1].itemID)
	assert.Equal(t, "payloadB", ex.sessions[1].finalRaw)

	// Verify filtered outputs are separate
	partials, _ := collectTextParts(col.list)
	assert.Contains(t, partials, "A1 ")
	assert.Contains(t, partials, " A2")
	assert.Contains(t, partials, "B1 ")
	assert.Contains(t, partials, " B2")
}

// Test 13: Zero-length deltas stability
func TestFilteringSink_ZeroLengthDeltas_Outside(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "", "")))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "prefix ", "prefix ")))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "", "prefix ")))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "<core:x:v1>abc</core:x:v1>", "prefix <core:x:v1>abc</core:x:v1>")))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "", "prefix <core:x:v1>abc</core:x:v1>")))
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, "prefix <core:x:v1>abc</core:x:v1>")))

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "prefix ", final)
}

func TestFilteringSink_ZeroLengthDeltas_InsideCapture(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{}, ex)

	meta := newMeta()
	completion := ""
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "<core:x:v1>", "<core:x:v1>")))
	completion = "<core:x:v1>"
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "", completion)))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "abc", completion+"abc")))
	completion = completion + "abc"
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "", completion)))
	require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, "</core:x:v1>", completion+"</core:x:v1>")))
	completion = completion + "</core:x:v1>"
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, completion)))

	require.NotNil(t, ex.last)
	assert.True(t, ex.last.completed)
	assert.Equal(t, "abc", ex.last.finalRaw)

	_, final := collectTextParts(col.list)
	assert.Equal(t, "", final)
}

// Test 14: MaxCaptureBytes (future-ready - skipped if not implemented)
func TestFilteringSink_MaxCaptureBytes_SkipIfNotImplemented(t *testing.T) {
	col := &eventCollector{}
	ex := &testExtractor{pkg: "core", typ: "x", ver: "v1"}
	sink := NewFilteringSink(col, Options{MaxCaptureBytes: 5}, ex)

	meta := newMeta()
	feedParts(t, sink, meta, []string{"<core:x:v1>123456789</core:x:v1>"})

	require.NotNil(t, ex.last)
	// Currently MaxCaptureBytes is not enforced, so this should succeed
	// Once implemented, update this test to verify failure behavior
	assert.True(t, ex.last.completed)
	assert.Equal(t, "123456789", ex.last.finalRaw)
}

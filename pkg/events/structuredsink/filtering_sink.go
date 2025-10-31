package structuredsink

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Options control the filtering sink behavior.
type Options struct {
	MaxCaptureBytes int
	Malformed       MalformedPolicy
	Debug           bool // if true, emit zerolog debug traces
}

func (o *Options) withDefaults() Options {
	ret := *o
	if ret.Malformed == 0 {
		ret.Malformed = MalformedErrorEvents
	}
	return ret
}

// MalformedPolicy clarifies behavior on unclosed/malformed structured blocks.
type MalformedPolicy int

const (
	MalformedErrorEvents     MalformedPolicy = iota // call OnCompleted(false), do not reinsert text
	MalformedReconstructText                        // reinsert reconstructed block into filtered text and call OnCompleted(false)
	MalformedIgnore                                 // drop captured payload and call OnCompleted(false)
)

func (o Options) resolveMalformedPolicy() MalformedPolicy {
	switch o.Malformed {
	case MalformedIgnore, MalformedReconstructText, MalformedErrorEvents:
		return o.Malformed
	default:
		return MalformedErrorEvents
	}
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
	OnRaw(ctx context.Context, chunk []byte) []events.Event
	OnCompleted(ctx context.Context, raw []byte, success bool, err error) []events.Event
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
	id uuid.UUID
	// contexts: stream-scoped and current item-scoped
	ctx        context.Context
	cancel     context.CancelFunc
	itemCtx    context.Context
	itemCancel context.CancelFunc
	// track full raw text seen so far (to avoid double-appending on final)
	rawSeen strings.Builder
	// output buffer state for completion consistency
	filteredCompletion strings.Builder

	// capture state
	state         parserState
	name          string
	dtype         string
	seq           int
	session       ExtractorSession
	payloadBuf    strings.Builder
	expectedClose string
	lagBuf        tagLagBuffer

	// sub-state buffers for partial pattern matches
	openTagBuf strings.Builder
}

type parserState int

const (
	stateIdle parserState = iota
	stateCapturing
)

// tagLagBuffer holds the last N bytes (N = closeTagLen-1) to reliably detect close tag
// without leaking partial close-tag bytes to payload.
type tagLagBuffer struct {
	buf      []byte
	capacity int
}

func (b *tagLagBuffer) reset(capacity int) {
	if cap(b.buf) < capacity {
		b.buf = make([]byte, 0, capacity)
	} else {
		b.buf = b.buf[:0]
	}
	b.capacity = capacity
}

// PublishEvent implements events.EventSink. It intercepts partial/final text events
// and forwards filtered variants while publishing extractor-defined typed events.
func (f *FilteringSink) PublishEvent(ev events.Event) error {
	t := ev.Type()
	if t == events.EventTypePartialCompletion {
		return f.handlePartial(ev)
	}
	if t == events.EventTypeFinal {
		return f.handleFinal(ev)
	}
	return f.next.PublishEvent(ev)
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
		if st.itemCancel != nil {
			st.itemCancel()
			st.itemCancel = nil
		}
		if st.cancel != nil {
			st.cancel()
		}
		delete(f.byStreamID, meta.ID)
	}
}

func (f *FilteringSink) publishAll(ctx context.Context, meta events.EventMetadata, list []events.Event) error {
	for _, e := range list {
		if e == nil {
			continue
		}
		// Ensure metadata is set if missing
		impl, ok := e.(*events.EventImpl)
		if ok {
			// keep Type_ and payload; only set meta if zero UUID
			if impl.Metadata_.ID == uuid.Nil {
				impl.Metadata_.ID = meta.ID
			}
			if impl.Metadata_.RunID == "" {
				impl.Metadata_.RunID = meta.RunID
			}
			if impl.Metadata_.TurnID == "" {
				impl.Metadata_.TurnID = meta.TurnID
			}
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
	// Typed partials only
	if pcTyped, isTyped := ev.(*events.EventPartialCompletion); isTyped {
		delta = pcTyped.Delta
		meta = pcTyped.Metadata()
		ok = true
	}
	if !ok {
		// pass-through on unknown
		return f.next.PublishEvent(ev)
	}

	st := f.getState(meta)
	if f.opts.Debug {
		log.Debug().Str("stream", meta.ID.String()).Str("event", "partial").Str("delta", delta).Msg("incoming")
	}
	st.rawSeen.WriteString(delta)

	filteredDelta, typedEvents := f.scanAndFilter(meta, st, delta)

	// Maintain filtered completion consistency
	st.filteredCompletion.WriteString(filteredDelta)

	// Forward filtered partial only if there is any delta (including empty is okay to preserve timing)
	fwd := &events.EventPartialCompletion{
		EventImpl:  events.EventImpl{Type_: events.EventTypePartialCompletion, Metadata_: meta},
		Delta:      filteredDelta,
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
		if f.opts.Debug {
			log.Debug().Str("stream", meta.ID.String()).Str("event", "final").Int("raw_seen_len", len(prefix)).Int("full_len", len(full)).Int("tail_len", len(delta)).Msg("incoming")
		}
		filtered, typed := f.scanAndFilter(meta, st, delta)
		st.filteredCompletion.WriteString(filtered)
		_ = f.publishAll(st.ctx, meta, typed)
		// Handle unfinished capture at final
		if st.state == stateCapturing {
			var malformedOut strings.Builder
			flushMalformed(f, meta, st, &malformedOut, &typed)
			st.filteredCompletion.WriteString(malformedOut.String())
			_ = f.publishAll(st.ctx, meta, typed)
		}
		out := events.NewFinalEvent(meta, st.filteredCompletion.String())
		f.deleteState(meta)
		return f.next.PublishEvent(out)
	}

	// Unknown final—pass-through
	return f.next.PublishEvent(ev)
}

// scanAndFilter processes incoming delta, updating state and returning
// - filteredDelta to forward to UI
// - typed events to publish based on extractor sessions
func (f *FilteringSink) scanAndFilter(meta events.EventMetadata, st *streamState, delta string) (string, []events.Event) {
	var out strings.Builder
	var typed []events.Event
	var capDelta strings.Builder

	for i := 0; i < len(delta); i++ {
		ch := delta[i]

		if st.state != stateCapturing {
			// Accumulate potential open tag if a '<' appeared previously or now
			if st.openTagBuf.Len() > 0 || ch == '<' {
				st.openTagBuf.WriteByte(ch)
				// Decide if we have a full valid tag
				tagText := st.openTagBuf.String()
				if tryParseOpenTag(st, tagText) {
					// fully parsed and valid; enter capture state
					st.state = stateCapturing
					st.seq++
					st.payloadBuf.Reset()
					st.expectedClose = "</$" + st.name + ":" + st.dtype + ">"
					st.lagBuf.reset(len(st.expectedClose) - 1)

					if f.opts.Debug {
						log.Debug().Str("stream", meta.ID.String()).Str("name", st.name).Str("dtype", st.dtype).Msg("filtering-sink: open tag detected")
					}

					// Emit OnStart
					if ex := f.exByKey[extractorKey(st.name, st.dtype)]; ex != nil {
						// derive per-item context from the stream context
						if st.itemCancel != nil {
							st.itemCancel()
							st.itemCancel = nil
						}
						st.itemCtx, st.itemCancel = context.WithCancel(st.ctx)
						st.session = ex.NewSession(st.itemCtx, meta, itemID(meta.ID, st.seq))
						if f.opts.Debug {
							log.Debug().Str("stream", meta.ID.String()).Str("name", st.name).Str("dtype", st.dtype).Msg("filtering-sink: extractor session started")
						}
						typed = append(typed, st.session.OnStart(st.itemCtx)...)
					} else {
						if f.opts.Debug {
							log.Debug().Str("stream", meta.ID.String()).Str("name", st.name).Str("dtype", st.dtype).Msg("filtering-sink: no extractor found for tag; flushing as text")
						}
						// Unknown extractor: treat as not capturing (flush buffer)
						out.WriteString(tagText)
						st.openTagBuf.Reset()
						st.state = stateIdle
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

		// Capturing branch with lag buffer to detect close tag without leakage
		closeLen := len(st.expectedClose)
		// If we have exactly closeLen-1 bytes in lag and new ch would complete close tag, detect and finalize
		if len(st.lagBuf.buf)+1 == closeLen {
			// compare lag bytes with prefix and ch with last char
			if strings.HasPrefix(st.expectedClose, string(st.lagBuf.buf)) && st.expectedClose[closeLen-1] == ch {
				// flush remaining in-delta emitted bytes first
				if st.session != nil && capDelta.Len() > 0 {
					typed = append(typed, st.session.OnRaw(st.itemCtx, []byte(capDelta.String()))...)
					capDelta.Reset()
				}
				// finalize item with payload only; lagBuf holds the close-tag prefix, drop it
				finalRaw := []byte(st.payloadBuf.String())
				if st.session != nil {
					if f.opts.Debug {
						log.Debug().Str("stream", meta.ID.String()).Str("name", st.name).Str("dtype", st.dtype).Int("final_len", len(finalRaw)).Msg("filtering-sink: close tag detected; completing session")
					}
					typed = append(typed, st.session.OnCompleted(st.itemCtx, finalRaw, true, nil)...)
				}
				// reset state to idle
				st.state = stateIdle
				st.name, st.dtype = "", ""
				st.session = nil
				if st.itemCancel != nil {
					st.itemCancel()
					st.itemCancel = nil
				}
				st.payloadBuf.Reset()
				st.openTagBuf.Reset()
				st.lagBuf.reset(0)
				st.expectedClose = ""
				continue
			}
		}

		// Not closing yet: emit oldest byte if lag is full, then push ch
		if len(st.lagBuf.buf) == st.lagBuf.capacity && st.lagBuf.capacity > 0 {
			b := st.lagBuf.buf[0]
			st.lagBuf.buf = st.lagBuf.buf[1:]
			capDelta.WriteByte(b)
			st.payloadBuf.WriteByte(b)
		}
		st.lagBuf.buf = append(st.lagBuf.buf, ch)
	}

	// End of delta: if capturing, flush the accumulated per-delta payload to extractor
	if st.state == stateCapturing && st.session != nil && capDelta.Len() > 0 {
		typed = append(typed, st.session.OnRaw(st.itemCtx, []byte(capDelta.String()))...)
	}

	// If not capturing, decide whether to flush any openTagBuf remnants.
	// Preserve potential structured tag prefixes like '<$' across deltas to allow split tags.
	// Additionally preserve a lone '<' so that an open tag split exactly at '<' continues across partials.
	if st.state != stateCapturing && st.openTagBuf.Len() > 0 {
		s := st.openTagBuf.String()
		if !strings.HasPrefix(s, "<$") && s != "<" {
			out.WriteString(s)
			st.openTagBuf.Reset()
		}
	}

	return out.String(), coalesce(typed)
}

func flushMalformed(f *FilteringSink, meta events.EventMetadata, st *streamState, out *strings.Builder, typed *[]events.Event) {
	policy := f.opts.resolveMalformedPolicy()
	switch policy {
	case MalformedIgnore:
		// drop everything captured so far
	case MalformedReconstructText:
		// reconstruct a best-effort raw block (not byte-identical)
		out.WriteString("<$" + st.name + ":" + st.dtype + ">")
		out.WriteString(st.payloadBuf.String())
		if len(st.lagBuf.buf) > 0 {
			out.WriteString(string(st.lagBuf.buf))
		}
	case MalformedErrorEvents:
		// fall through to emit error event below
	}
	if st.session != nil {
		// include lagBuf bytes that were held back from payload
		finalRaw := append([]byte(st.payloadBuf.String()), st.lagBuf.buf...)
		*typed = append(*typed, st.session.OnCompleted(st.itemCtx, finalRaw, false, errors.New("malformed structured block"))...)
	}
	// reset state
	st.state = stateIdle
	st.name, st.dtype = "", ""
	st.session = nil
	if st.itemCancel != nil {
		st.itemCancel()
		st.itemCancel = nil
	}
	st.payloadBuf.Reset()
	st.openTagBuf.Reset()
	st.lagBuf.reset(0)
	st.expectedClose = ""
}

func itemID(id uuid.UUID, seq int) string {
	return id.String() + ":" + strconv.Itoa(seq)
}

func tryParseOpenTag(st *streamState, s string) bool {
	if !strings.HasPrefix(s, "<$") {
		return false
	}
	if !strings.HasSuffix(s, ">") {
		return false
	}
	// strip prefix/suffix
	body := s[2 : len(s)-1]
	// split on first ':'
	idx := strings.IndexByte(body, ':')
	if idx <= 0 || idx >= len(body)-1 {
		return false
	}
	name := body[:idx]
	dtype := body[idx+1:]
	if !isValidName(name) || !isValidDType(dtype) {
		return false
	}
	st.name = name
	st.dtype = dtype
	st.openTagBuf.Reset()
	return true
}

func isValidName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func isValidDType(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' {
			continue
		}
		return false
	}
	return true
}

// no fence detection in v2 (handled by extractor)

// (no close-tag parser helper; close tag detection is handled inline in scanAndFilter)

// no sink-side parsing in v2 (handled by extractor)

func coalesce(list []events.Event) []events.Event {
	if len(list) == 0 {
		return nil
	}
	return list
}

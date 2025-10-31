package structuredsink

import (
	"context"
	"math/rand"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// minimal collector for fuzz/property tests
type fuzzCollector struct{ list []events.Event }

func (c *fuzzCollector) PublishEvent(ev events.Event) error {
	c.list = append(c.list, ev)
	return nil
}

// minimal extractor that captures final raw payload
type fuzzExtractor struct {
	name, dtype string
	last        *fuzzSession
}

func (e *fuzzExtractor) Name() string     { return e.name }
func (e *fuzzExtractor) DataType() string { return e.dtype }
func (e *fuzzExtractor) NewSession(ctx context.Context, meta events.EventMetadata, itemID string) ExtractorSession {
	s := &fuzzSession{}
	e.last = s
	return s
}

type fuzzSession struct {
	final string
	done  bool
}

func (s *fuzzSession) OnStart(ctx context.Context) []events.Event         { return nil }
func (s *fuzzSession) OnRaw(ctx context.Context, b []byte) []events.Event { return nil }
func (s *fuzzSession) OnCompleted(ctx context.Context, raw []byte, ok bool, err error) []events.Event {
	s.final, s.done = string(raw), true
	return nil
}

func newMetaFuzz() events.EventMetadata {
	return events.EventMetadata{ID: uuid.New(), RunID: "run", TurnID: "turn"}
}

// splitIntoParts randomly cuts s into n segments (n>=1) deterministically from seed
func splitIntoParts(s string, seed int64, maxParts int) []string {
	if maxParts < 1 {
		maxParts = 1
	}
	r := rand.New(rand.NewSource(seed))
	// choose number of parts between 1 and maxParts
	np := 1 + r.Intn(maxParts)
	// generate np-1 cut indices in (0,len)
	L := len(s)
	cuts := make([]int, 0, np-1)
	for len(cuts) < np-1 {
		if L == 0 {
			break
		}
		x := 1 + r.Intn(L-1)
		ok := true
		for _, c := range cuts {
			if c == x {
				ok = false
				break
			}
		}
		if ok {
			cuts = append(cuts, x)
		}
	}
	if len(cuts) == 0 {
		return []string{s}
	}
	// sort cuts
	for i := 0; i < len(cuts); i++ {
		for j := i + 1; j < len(cuts); j++ {
			if cuts[j] < cuts[i] {
				cuts[i], cuts[j] = cuts[j], cuts[i]
			}
		}
	}
	parts := make([]string, 0, len(cuts)+1)
	prev := 0
	for _, c := range cuts {
		parts = append(parts, s[prev:c])
		prev = c
	}
	parts = append(parts, s[prev:])
	return parts
}

func feedPartsFuzz(t *testing.T, sink *FilteringSink, meta events.EventMetadata, parts []string) {
	completion := ""
	for _, p := range parts {
		completion += p
		require.NoError(t, sink.PublishEvent(events.NewPartialCompletionEvent(meta, p, completion)))
	}
	require.NoError(t, sink.PublishEvent(events.NewFinalEvent(meta, completion)))
}

// Property: For a known extractor, arbitrary segmentations of a valid tagged block
// must yield the same final payload and filtered text.
func TestFilteringSink_RandomSegmentations_Known(t *testing.T) {
	const prefix = "P "
	const payload = "PAY"
	const suffix = " S"
	full := prefix + "<$x:v1>" + payload + "</$x:v1>" + suffix

	for seed := int64(0); seed < 200; seed++ {
		col := &fuzzCollector{}
		ex := &fuzzExtractor{name: "x", dtype: "v1"}
		sink := NewFilteringSink(col, Options{}, ex)
		meta := newMetaFuzz()
		parts := splitIntoParts(full, seed, 10)
		feedPartsFuzz(t, sink, meta, parts)

		require.NotNil(t, ex.last)
		require.True(t, ex.last.done)
		require.Equal(t, payload, ex.last.final)

		// Extract filtered final from collector
		_, final := collectTextParts(col.list)
		require.Equal(t, prefix+suffix, final)
	}
}

// Property: Unknown extractors must forward text verbatim across arbitrary segmentations.
func TestFilteringSink_RandomSegmentations_Unknown(t *testing.T) {
	const full = "X <$unknown:v1>ABC</$unknown:v1> Y"
	for seed := int64(0); seed < 200; seed++ {
		col := &fuzzCollector{}
		ex := &fuzzExtractor{name: "x", dtype: "v1"} // register different tag
		sink := NewFilteringSink(col, Options{}, ex)
		meta := newMetaFuzz()
		parts := splitIntoParts(full, seed, 10)
		feedPartsFuzz(t, sink, meta, parts)

		// No session created for unknown tag
		require.Nil(t, ex.last)
		_, final := collectTextParts(col.list)
		require.Equal(t, full, final)
	}
}

// Go fuzz: tries random segmentations for known extractor case
func FuzzFilteringSink_TagSplit_Known(f *testing.F) {
	const prefix = "P "
	const payload = "PAY"
	const suffix = " S"
	full := prefix + "<$x:v1>" + payload + "</$x:v1>" + suffix
	f.Add(int64(1))
	f.Add(int64(42))
	f.Add(int64(99999))
	f.Fuzz(func(t *testing.T, seed int64) {
		col := &fuzzCollector{}
		ex := &fuzzExtractor{name: "x", dtype: "v1"}
		sink := NewFilteringSink(col, Options{}, ex)
		meta := newMetaFuzz()
		parts := splitIntoParts(full, seed, 10)
		feedPartsFuzz(t, sink, meta, parts)
		require.NotNil(t, ex.last)
		require.True(t, ex.last.done)
		require.Equal(t, payload, ex.last.final)
		_, final := collectTextParts(col.list)
		require.Equal(t, prefix+suffix, final)
	})
}

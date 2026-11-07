package systemone

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-go-golems/geppetto/pkg/decision"
)

// TestLive_DecideAgainstLunaRoute runs the guide's §1.3 request against the
// real gateway. It is skipped unless GEPPETTO_LIVE_SYSTEMONE=1:
//
//	GEPPETTO_LIVE_SYSTEMONE=1 LUNAROUTE_API_KEY=<key> \
//	GEPPETTO_SYSTEMONE_MODEL=djev go test ./pkg/decision/systemone -run TestLive -v -count=1
func TestLive_DecideAgainstLunaRoute(t *testing.T) {
	if os.Getenv("GEPPETTO_LIVE_SYSTEMONE") != "1" {
		t.Skip("set GEPPETTO_LIVE_SYSTEMONE=1 to run")
	}
	key := os.Getenv("LUNAROUTE_API_KEY")
	if key == "" {
		t.Skip("set LUNAROUTE_API_KEY")
	}
	model := os.Getenv("GEPPETTO_SYSTEMONE_MODEL")
	if model == "" {
		model = "djev"
	}
	p, err := New(Options{APIKey: key, Model: model})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := p.Decide(ctx, decision.Request{
		State: map[string]any{"query": "how does the rebalance law preserve product placements?"},
		Questions: map[string]decision.Question{
			"route": {Type: decision.Choice, Instructions: "Which retrieval strategy fits this query best?", Options: []decision.Option{
				{Name: "keyword", Description: "Exact identifiers, names, error strings"},
				{Name: "semantic", Description: "Conceptual question needing paraphrase matching"},
				{Name: "none", Description: "Chit-chat or no retrieval needed"}}},
			"needs_retrieval": {Type: decision.Noul, Instructions: "Does answering require looking up documents?"},
			"complexity":      {Type: decision.Score, Instructions: "How complex is the question?", Levels: []string{"trivial lookup", "single fact", "explanation of one mechanism", "multi-document synthesis"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("model=%s request_id=%s duration=%dms usage=%+v", r.Model, r.RequestID, *r.DurationMs, *r.Usage)
	for id, a := range r.Answers {
		t.Logf("%s: %+v", id, a)
	}
}

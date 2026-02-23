package main

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gepaopt "github.com/go-go-golems/geppetto/pkg/optimizer/gepa"
)

func TestEvalReportQueriesAndFormats(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "runs.sqlite")

	if err := seedReportFixture(dbPath); err != nil {
		t.Fatalf("seed fixture failed: %v", err)
	}

	// Query-level checks.
	db, err := openSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	runRows, err := queryRunRows(db, 20)
	if err != nil {
		t.Fatalf("queryRunRows failed: %v", err)
	}
	if len(runRows) != 2 {
		t.Fatalf("expected 2 run rows, got %d", len(runRows))
	}

	pluginRows, err := queryPluginSummaryRows(db, 20)
	if err != nil {
		t.Fatalf("queryPluginSummaryRows failed: %v", err)
	}
	if len(pluginRows) == 0 {
		t.Fatalf("expected plugin summary rows")
	}

	// JSON output path.
	jsonOut, err := captureStdout(func() error {
		return runEvalReport(&evalReportOptions{
			DBPath:    dbPath,
			LimitRuns: 20,
			Format:    "json",
		})
	})
	if err != nil {
		t.Fatalf("runEvalReport json failed: %v", err)
	}
	if !strings.Contains(jsonOut, "\"runs\"") {
		t.Fatalf("expected json output to contain runs, got: %s", jsonOut)
	}

	// Table output path.
	tableOut, err := captureStdout(func() error {
		return runEvalReport(&evalReportOptions{
			DBPath:    dbPath,
			LimitRuns: 20,
			Format:    "table",
		})
	})
	if err != nil {
		t.Fatalf("runEvalReport table failed: %v", err)
	}
	if !strings.Contains(tableOut, "Recent GEPA runs") {
		t.Fatalf("expected table output header, got: %s", tableOut)
	}
}

func seedReportFixture(dbPath string) error {
	// Optimize run.
	optRec, err := newRunRecorder(runRecorderConfig{
		DBPath:      dbPath,
		Mode:        "optimize",
		PluginID:    "example.plugin",
		PluginName:  "Example Plugin",
		Profile:     "default",
		DatasetSize: 3,
		Objective:   "maximize score",
		MaxEvals:    8,
		BatchSize:   2,
		SeedPrompt:  "seed",
	})
	if err != nil {
		return err
	}
	best := gepaopt.Candidate{"prompt": "best"}
	res := &gepaopt.Result{
		BestCandidate: best,
		BestStats: gepaopt.CandidateStats{
			MeanScore:      0.75,
			MeanObjectives: gepaopt.ObjectiveScores{"score": 0.75},
			N:              3,
		},
		CallsUsed: 6,
		Candidates: []gepaopt.CandidateEntry{
			{
				ID:        0,
				ParentID:  -1,
				Hash:      hashCandidate(gepaopt.Candidate{"prompt": "seed"}),
				Candidate: gepaopt.Candidate{"prompt": "seed"},
				GlobalStats: gepaopt.CandidateStats{
					MeanScore:      0.40,
					MeanObjectives: gepaopt.ObjectiveScores{"score": 0.40},
					N:              3,
				},
				EvalsCached: 3,
			},
			{
				ID:        1,
				ParentID:  0,
				Hash:      hashCandidate(best),
				Candidate: best,
				GlobalStats: gepaopt.CandidateStats{
					MeanScore:      0.75,
					MeanObjectives: gepaopt.ObjectiveScores{"score": 0.75},
					N:              3,
				},
				EvalsCached: 3,
			},
		},
	}
	if err := optRec.RecordOptimizeResult(res); err != nil {
		return err
	}
	if err := optRec.Close(true, nil); err != nil {
		return err
	}

	// Eval run.
	evalRec, err := newRunRecorder(runRecorderConfig{
		DBPath:      dbPath,
		Mode:        "eval",
		PluginID:    "example.plugin",
		PluginName:  "Example Plugin",
		Profile:     "default",
		DatasetSize: 2,
		SeedPrompt:  "best",
	})
	if err != nil {
		return err
	}
	stats := gepaopt.CandidateStats{
		MeanScore:      0.5,
		MeanObjectives: gepaopt.ObjectiveScores{"score": 0.5},
		N:              2,
	}
	evals := []gepaopt.ExampleEval{
		{
			ExampleIndex: 0,
			Result: gepaopt.EvalResult{
				Score:      1,
				Objectives: gepaopt.ObjectiveScores{"score": 1},
				Output:     map[string]any{"text": "ok"},
				Trace:      map[string]any{"step": 1},
				Raw:        map[string]any{"raw": true},
			},
		},
		{
			ExampleIndex: 1,
			Result: gepaopt.EvalResult{
				Score:      0,
				Objectives: gepaopt.ObjectiveScores{"score": 0},
				Output:     map[string]any{"text": "bad"},
				Trace:      map[string]any{"step": 2},
				Raw:        map[string]any{"raw": false},
			},
		},
	}
	if err := evalRec.RecordEvalResult(gepaopt.Candidate{"prompt": "best"}, stats, evals); err != nil {
		return err
	}
	return evalRec.Close(true, nil)
}

func openSQLite(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", path)
}

func captureStdout(fn func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	runErr := fn()

	_ = w.Close()
	os.Stdout = oldStdout

	blob, readErr := io.ReadAll(r)
	_ = r.Close()
	if runErr != nil {
		return "", runErr
	}
	if readErr != nil {
		return "", readErr
	}
	return string(blob), nil
}

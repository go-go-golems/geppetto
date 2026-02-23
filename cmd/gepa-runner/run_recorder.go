package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gepaopt "github.com/go-go-golems/geppetto/pkg/optimizer/gepa"
	_ "github.com/mattn/go-sqlite3"
)

const (
	gepaRunsTable             = "gepa_runs"
	gepaCandidateMetricsTable = "gepa_candidate_metrics"
	gepaEvalExamplesTable     = "gepa_eval_examples"
)

type runRecorderConfig struct {
	DBPath      string
	Mode        string
	PluginID    string
	PluginName  string
	Profile     string
	DatasetSize int
	Objective   string
	MaxEvals    int
	BatchSize   int
	SeedPrompt  string
}

type runRecord struct {
	RunID             string
	Mode              string
	Status            string
	StartedAtMs       int64
	FinishedAtMs      int64
	DurationMs        int64
	PluginID          string
	PluginName        string
	Profile           string
	DatasetSize       int
	Objective         string
	MaxEvals          int
	BatchSize         int
	CallsUsed         int
	BestMeanScore     *float64
	BestN             *int
	MeanScore         *float64
	MeanN             *int
	CandidateCount    int
	BestCandidateHash string
	SeedPromptSHA256  string
	ErrorMessage      string
	CreatedAtMs       int64
}

type candidateMetricRow struct {
	RunID              string
	CandidateID        int
	ParentID           int
	CandidateHash      string
	MeanScore          float64
	N                  int
	MeanObjectivesJSON string
	EvalsCached        int
	ReflectionRaw      string
	CandidateJSON      string
	IsBest             int
}

type evalExampleRow struct {
	RunID          string
	CandidateHash  string
	ExampleIndex   int
	Score          float64
	ObjectivesJSON string
	Feedback       string
	EvaluatorNotes string
	OutputJSON     string
	TraceJSON      string
	RawJSON        string
}

type runRecorder struct {
	db *sql.DB

	run        runRecord
	candidates []candidateMetricRow
	evals      []evalExampleRow
}

func newRunRecorder(cfg runRecorderConfig) (*runRecorder, error) {
	dbPath := strings.TrimSpace(cfg.DBPath)
	if dbPath == "" {
		return nil, fmt.Errorf("run recorder: db path is empty")
	}
	mode := strings.TrimSpace(cfg.Mode)
	if mode != "optimize" && mode != "eval" {
		return nil, fmt.Errorf("run recorder: invalid mode %q", mode)
	}
	if err := ensureParentDir(dbPath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	if err := ensureRecorderTables(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	nowMs := time.Now().UnixMilli()
	rec := &runRecorder{
		db: db,
		run: runRecord{
			RunID:            generateRunID(mode),
			Mode:             mode,
			Status:           "running",
			StartedAtMs:      nowMs,
			PluginID:         strings.TrimSpace(cfg.PluginID),
			PluginName:       strings.TrimSpace(cfg.PluginName),
			Profile:          strings.TrimSpace(cfg.Profile),
			DatasetSize:      cfg.DatasetSize,
			Objective:        strings.TrimSpace(cfg.Objective),
			MaxEvals:         cfg.MaxEvals,
			BatchSize:        cfg.BatchSize,
			SeedPromptSHA256: hashString(strings.TrimSpace(cfg.SeedPrompt)),
			CreatedAtMs:      nowMs,
		},
		candidates: make([]candidateMetricRow, 0, 16),
		evals:      make([]evalExampleRow, 0, 64),
	}
	return rec, nil
}

func (r *runRecorder) RecordOptimizeResult(res *gepaopt.Result) error {
	if r == nil || res == nil {
		return nil
	}
	if r.run.Mode != "optimize" {
		return fmt.Errorf("run recorder: optimize result cannot be recorded in mode %q", r.run.Mode)
	}

	bestHash := hashCandidate(res.BestCandidate)
	bestMean := res.BestStats.MeanScore
	bestN := res.BestStats.N

	r.run.CallsUsed = res.CallsUsed
	r.run.BestMeanScore = &bestMean
	r.run.BestN = &bestN
	r.run.CandidateCount = len(res.Candidates)
	r.run.BestCandidateHash = bestHash

	for _, c := range res.Candidates {
		candidateJSON, err := marshalJSONString(c.Candidate)
		if err != nil {
			return err
		}
		objectivesJSON, err := marshalJSONString(c.GlobalStats.MeanObjectives)
		if err != nil {
			return err
		}

		row := candidateMetricRow{
			RunID:              r.run.RunID,
			CandidateID:        c.ID,
			ParentID:           c.ParentID,
			CandidateHash:      c.Hash,
			MeanScore:          c.GlobalStats.MeanScore,
			N:                  c.GlobalStats.N,
			MeanObjectivesJSON: objectivesJSON,
			EvalsCached:        c.EvalsCached,
			ReflectionRaw:      c.ReflectionRaw,
			CandidateJSON:      candidateJSON,
			IsBest:             boolToInt(c.Hash == bestHash),
		}
		r.candidates = append(r.candidates, row)
	}
	return nil
}

func (r *runRecorder) RecordEvalResult(candidate gepaopt.Candidate, stats gepaopt.CandidateStats, evals []gepaopt.ExampleEval) error {
	if r == nil {
		return nil
	}
	if r.run.Mode != "eval" {
		return fmt.Errorf("run recorder: eval result cannot be recorded in mode %q", r.run.Mode)
	}

	meanScore := stats.MeanScore
	meanN := stats.N
	candidateHash := hashCandidate(candidate)

	r.run.MeanScore = &meanScore
	r.run.MeanN = &meanN
	r.run.CallsUsed = len(evals)
	r.run.CandidateCount = 1
	r.run.BestCandidateHash = candidateHash

	for _, ev := range evals {
		objectivesJSON, err := marshalJSONString(ev.Result.Objectives)
		if err != nil {
			return err
		}
		outputJSON, err := marshalJSONString(ev.Result.Output)
		if err != nil {
			return err
		}
		traceJSON, err := marshalJSONString(ev.Result.Trace)
		if err != nil {
			return err
		}
		rawJSON, err := marshalJSONString(ev.Result.Raw)
		if err != nil {
			return err
		}
		row := evalExampleRow{
			RunID:          r.run.RunID,
			CandidateHash:  candidateHash,
			ExampleIndex:   ev.ExampleIndex,
			Score:          ev.Result.Score,
			ObjectivesJSON: objectivesJSON,
			Feedback:       truncateString(stringOrEmpty(ev.Result.Feedback), 2000),
			EvaluatorNotes: truncateString(ev.Result.EvaluatorNotes, 2000),
			OutputJSON:     outputJSON,
			TraceJSON:      traceJSON,
			RawJSON:        rawJSON,
		}
		r.evals = append(r.evals, row)
	}
	return nil
}

func (r *runRecorder) Close(success bool, runErr error) error {
	if r == nil || r.db == nil {
		return nil
	}
	defer func() {
		_ = r.db.Close()
	}()

	nowMs := time.Now().UnixMilli()
	r.run.FinishedAtMs = nowMs
	r.run.DurationMs = maxInt64(0, nowMs-r.run.StartedAtMs)
	if success && runErr == nil {
		r.run.Status = "completed"
	} else {
		r.run.Status = "failed"
		if runErr != nil {
			r.run.ErrorMessage = truncateString(runErr.Error(), 4000)
		}
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := r.insertRun(tx); err != nil {
		return err
	}
	if err := r.insertCandidates(tx); err != nil {
		return err
	}
	if err := r.insertEvalRows(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *runRecorder) RunID() string {
	if r == nil {
		return ""
	}
	return r.run.RunID
}

func (r *runRecorder) insertRun(tx *sql.Tx) error {
	_, err := tx.Exec(`
INSERT OR REPLACE INTO gepa_runs (
  run_id,
  mode,
  status,
  started_at_ms,
  finished_at_ms,
  duration_ms,
  plugin_id,
  plugin_name,
  profile,
  dataset_size,
  objective,
  max_evals,
  batch_size,
  calls_used,
  best_mean_score,
  best_n,
  mean_score,
  mean_n,
  candidate_count,
  best_candidate_hash,
  seed_prompt_sha256,
  error,
  created_at_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.run.RunID,
		r.run.Mode,
		r.run.Status,
		r.run.StartedAtMs,
		r.run.FinishedAtMs,
		r.run.DurationMs,
		nullableString(r.run.PluginID),
		nullableString(r.run.PluginName),
		nullableString(r.run.Profile),
		r.run.DatasetSize,
		nullableString(r.run.Objective),
		nullableInt(r.run.MaxEvals),
		nullableInt(r.run.BatchSize),
		nullableInt(r.run.CallsUsed),
		nullableFloat(r.run.BestMeanScore),
		nullableIntPtr(r.run.BestN),
		nullableFloat(r.run.MeanScore),
		nullableIntPtr(r.run.MeanN),
		nullableInt(r.run.CandidateCount),
		nullableString(r.run.BestCandidateHash),
		nullableString(r.run.SeedPromptSHA256),
		nullableString(r.run.ErrorMessage),
		r.run.CreatedAtMs,
	)
	return err
}

func (r *runRecorder) insertCandidates(tx *sql.Tx) error {
	if len(r.candidates) == 0 {
		return nil
	}
	stmt, err := tx.Prepare(`
INSERT OR REPLACE INTO gepa_candidate_metrics (
  run_id,
  candidate_id,
  parent_id,
  candidate_hash,
  mean_score,
  n,
  mean_objectives_json,
  evals_cached,
  reflection_raw,
  candidate_json,
  is_best
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, c := range r.candidates {
		if _, err := stmt.Exec(
			c.RunID,
			c.CandidateID,
			c.ParentID,
			c.CandidateHash,
			c.MeanScore,
			c.N,
			c.MeanObjectivesJSON,
			c.EvalsCached,
			nullableString(c.ReflectionRaw),
			c.CandidateJSON,
			c.IsBest,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *runRecorder) insertEvalRows(tx *sql.Tx) error {
	if len(r.evals) == 0 {
		return nil
	}
	stmt, err := tx.Prepare(`
INSERT OR REPLACE INTO gepa_eval_examples (
  run_id,
  candidate_hash,
  example_index,
  score,
  objectives_json,
  feedback,
  evaluator_notes,
  output_json,
  trace_json,
  raw_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, ev := range r.evals {
		if _, err := stmt.Exec(
			ev.RunID,
			ev.CandidateHash,
			ev.ExampleIndex,
			ev.Score,
			ev.ObjectivesJSON,
			nullableString(ev.Feedback),
			nullableString(ev.EvaluatorNotes),
			ev.OutputJSON,
			ev.TraceJSON,
			ev.RawJSON,
		); err != nil {
			return err
		}
	}
	return nil
}

func ensureRecorderTables(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("run recorder: db is nil")
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS gepa_runs (
  run_id TEXT PRIMARY KEY,
  mode TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at_ms INTEGER NOT NULL,
  finished_at_ms INTEGER NOT NULL,
  duration_ms INTEGER NOT NULL,
  plugin_id TEXT,
  plugin_name TEXT,
  profile TEXT,
  dataset_size INTEGER NOT NULL DEFAULT 0,
  objective TEXT,
  max_evals INTEGER,
  batch_size INTEGER,
  calls_used INTEGER,
  best_mean_score REAL,
  best_n INTEGER,
  mean_score REAL,
  mean_n INTEGER,
  candidate_count INTEGER,
  best_candidate_hash TEXT,
  seed_prompt_sha256 TEXT,
  error TEXT,
  created_at_ms INTEGER NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS gepa_candidate_metrics (
  run_id TEXT NOT NULL,
  candidate_id INTEGER NOT NULL,
  parent_id INTEGER NOT NULL,
  candidate_hash TEXT NOT NULL,
  mean_score REAL NOT NULL,
  n INTEGER NOT NULL,
  mean_objectives_json TEXT NOT NULL,
  evals_cached INTEGER NOT NULL,
  reflection_raw TEXT,
  candidate_json TEXT NOT NULL,
  is_best INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (run_id, candidate_id)
)`,
		`CREATE TABLE IF NOT EXISTS gepa_eval_examples (
  run_id TEXT NOT NULL,
  candidate_hash TEXT NOT NULL,
  example_index INTEGER NOT NULL,
  score REAL NOT NULL,
  objectives_json TEXT NOT NULL,
  feedback TEXT,
  evaluator_notes TEXT,
  output_json TEXT NOT NULL,
  trace_json TEXT NOT NULL,
  raw_json TEXT NOT NULL,
  PRIMARY KEY (run_id, candidate_hash, example_index)
)`,
		`CREATE INDEX IF NOT EXISTS idx_gepa_runs_started_at ON gepa_runs (started_at_ms DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_gepa_runs_plugin ON gepa_runs (plugin_id, started_at_ms DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_gepa_candidates_run_hash ON gepa_candidate_metrics (run_id, candidate_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_gepa_eval_examples_run ON gepa_eval_examples (run_id, candidate_hash)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func ensureParentDir(path string) error {
	parent := filepath.Dir(path)
	if parent == "" || parent == "." {
		return nil
	}
	return os.MkdirAll(parent, 0o755)
}

func generateRunID(mode string) string {
	return fmt.Sprintf("gepa-%s-%d", mode, time.Now().UnixNano())
}

func hashCandidate(candidate gepaopt.Candidate) string {
	payload, err := json.Marshal(candidate)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func hashString(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func marshalJSONString(v any) (string, error) {
	if v == nil {
		return "null", nil
	}
	blob, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(blob), nil
}

func nullableString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullableInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullableIntPtr(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func stringOrEmpty(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

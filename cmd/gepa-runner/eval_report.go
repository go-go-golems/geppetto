package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

type evalReportOptions struct {
	DBPath    string
	LimitRuns int
	Format    string
}

type runReportRow struct {
	RunID          string   `json:"run_id"`
	Mode           string   `json:"mode"`
	Status         string   `json:"status"`
	StartedAt      string   `json:"started_at"`
	DurationMs     int64    `json:"duration_ms"`
	PluginID       string   `json:"plugin_id"`
	PluginName     string   `json:"plugin_name"`
	DatasetSize    int64    `json:"dataset_size"`
	CallsUsed      int64    `json:"calls_used"`
	CandidateCount int64    `json:"candidate_count"`
	Score          *float64 `json:"score,omitempty"`
	Error          string   `json:"error,omitempty"`
}

type pluginSummaryRow struct {
	PluginID       string   `json:"plugin_id"`
	Mode           string   `json:"mode"`
	RunCount       int64    `json:"run_count"`
	CompletedCount int64    `json:"completed_count"`
	FailedCount    int64    `json:"failed_count"`
	AvgDurationMs  *float64 `json:"avg_duration_ms,omitempty"`
	AvgScore       *float64 `json:"avg_score,omitempty"`
}

func newEvalReportCommand() *cobra.Command {
	opts := &evalReportOptions{
		DBPath:    ".gepa-runner/runs.sqlite",
		LimitRuns: 20,
		Format:    "table",
	}

	cmd := &cobra.Command{
		Use:   "eval-report",
		Short: "Report GEPA run metrics from SQLite recorder tables",
		Long: `Reads GEPA recorder tables:
- gepa_runs
- gepa_candidate_metrics
- gepa_eval_examples

and prints a compact report for recent runs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEvalReport(opts)
		},
	}

	cmd.Flags().StringVar(&opts.DBPath, "db", opts.DBPath, "Path to recorder SQLite database.")
	cmd.Flags().IntVar(&opts.LimitRuns, "limit-runs", opts.LimitRuns, "Max recent runs to include.")
	cmd.Flags().StringVar(&opts.Format, "format", opts.Format, "Output format: table or json.")

	return cmd
}

func runEvalReport(opts *evalReportOptions) error {
	if opts == nil {
		return fmt.Errorf("eval report options are nil")
	}
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = "table"
	}
	if format != "table" && format != "json" {
		return fmt.Errorf("invalid --format %q, expected table or json", opts.Format)
	}
	if opts.LimitRuns < 1 {
		opts.LimitRuns = 20
	}

	db, err := sql.Open("sqlite3", opts.DBPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	runRows, err := queryRunRows(db, opts.LimitRuns)
	if err != nil {
		return err
	}
	pluginRows, err := queryPluginSummaryRows(db, opts.LimitRuns)
	if err != nil {
		return err
	}

	if format == "json" {
		out := map[string]any{
			"db":         opts.DBPath,
			"limit_runs": opts.LimitRuns,
			"runs":       runRows,
			"plugins":    pluginRows,
		}
		blob, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		_, _ = os.Stdout.Write(blob)
		_, _ = os.Stdout.Write([]byte("\n"))
		return nil
	}

	printRunsTable(runRows)
	fmt.Println()
	printPluginSummaryTable(pluginRows)
	return nil
}

func queryRunRows(db *sql.DB, limitRuns int) ([]runReportRow, error) {
	q := fmt.Sprintf(`
SELECT
  run_id,
  mode,
  status,
  started_at_ms,
  duration_ms,
  plugin_id,
  plugin_name,
  dataset_size,
  calls_used,
  candidate_count,
  COALESCE(best_mean_score, mean_score) AS score,
  error
FROM %s
ORDER BY started_at_ms DESC
LIMIT ?
`, gepaRunsTable)

	rows, err := db.Query(q, limitRuns)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]runReportRow, 0, limitRuns)
	for rows.Next() {
		var runID string
		var mode string
		var status string
		var startedAtMs sql.NullInt64
		var durationMs sql.NullInt64
		var pluginID sql.NullString
		var pluginName sql.NullString
		var datasetSize sql.NullInt64
		var callsUsed sql.NullInt64
		var candidateCount sql.NullInt64
		var score sql.NullFloat64
		var lastError sql.NullString

		if err := rows.Scan(
			&runID,
			&mode,
			&status,
			&startedAtMs,
			&durationMs,
			&pluginID,
			&pluginName,
			&datasetSize,
			&callsUsed,
			&candidateCount,
			&score,
			&lastError,
		); err != nil {
			return nil, err
		}

		var scorePtr *float64
		if score.Valid {
			v := score.Float64
			scorePtr = &v
		}
		started := ""
		if startedAtMs.Valid && startedAtMs.Int64 > 0 {
			started = time.UnixMilli(startedAtMs.Int64).Format(time.RFC3339)
		}

		out = append(out, runReportRow{
			RunID:          runID,
			Mode:           mode,
			Status:         status,
			StartedAt:      started,
			DurationMs:     nullInt64Value(durationMs),
			PluginID:       nullStringValueWithDefault(pluginID, "(unknown)"),
			PluginName:     nullStringValue(pluginName),
			DatasetSize:    nullInt64Value(datasetSize),
			CallsUsed:      nullInt64Value(callsUsed),
			CandidateCount: nullInt64Value(candidateCount),
			Score:          scorePtr,
			Error:          nullStringValue(lastError),
		})
	}

	return out, rows.Err()
}

func queryPluginSummaryRows(db *sql.DB, limitRuns int) ([]pluginSummaryRow, error) {
	q := fmt.Sprintf(`
SELECT
  COALESCE(NULLIF(plugin_id, ''), '(unknown)') AS plugin_id,
  mode,
  COUNT(*) AS run_count,
  SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS completed_count,
  SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed_count,
  AVG(CAST(duration_ms AS REAL)) AS avg_duration_ms,
  AVG(COALESCE(best_mean_score, mean_score)) AS avg_score
FROM %s
WHERE run_id IN (
  SELECT run_id
  FROM %s
  ORDER BY started_at_ms DESC
  LIMIT ?
)
GROUP BY COALESCE(NULLIF(plugin_id, ''), '(unknown)'), mode
ORDER BY run_count DESC, plugin_id ASC, mode ASC
`, gepaRunsTable, gepaRunsTable)

	rows, err := db.Query(q, limitRuns)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]pluginSummaryRow, 0, 8)
	for rows.Next() {
		var pluginID string
		var mode string
		var runCount sql.NullInt64
		var completed sql.NullInt64
		var failed sql.NullInt64
		var avgDuration sql.NullFloat64
		var avgScore sql.NullFloat64

		if err := rows.Scan(
			&pluginID,
			&mode,
			&runCount,
			&completed,
			&failed,
			&avgDuration,
			&avgScore,
		); err != nil {
			return nil, err
		}

		var avgDurationPtr *float64
		if avgDuration.Valid {
			v := avgDuration.Float64
			avgDurationPtr = &v
		}
		var avgScorePtr *float64
		if avgScore.Valid {
			v := avgScore.Float64
			avgScorePtr = &v
		}

		out = append(out, pluginSummaryRow{
			PluginID:       pluginID,
			Mode:           mode,
			RunCount:       nullInt64Value(runCount),
			CompletedCount: nullInt64Value(completed),
			FailedCount:    nullInt64Value(failed),
			AvgDurationMs:  avgDurationPtr,
			AvgScore:       avgScorePtr,
		})
	}
	return out, rows.Err()
}

func printRunsTable(rows []runReportRow) {
	fmt.Println("Recent GEPA runs")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "RUN ID\tMODE\tSTATUS\tSTARTED\tDURATION_MS\tDATASET\tCALLS\tCANDIDATES\tSCORE\tPLUGIN")
	for _, row := range rows {
		score := "-"
		if row.Score != nil {
			score = fmt.Sprintf("%.6f", *row.Score)
		}
		plugin := row.PluginID
		if strings.TrimSpace(row.PluginName) != "" {
			plugin = fmt.Sprintf("%s (%s)", row.PluginName, row.PluginID)
		}
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%s\t%s\n",
			row.RunID,
			row.Mode,
			row.Status,
			row.StartedAt,
			row.DurationMs,
			row.DatasetSize,
			row.CallsUsed,
			row.CandidateCount,
			score,
			plugin,
		)
	}
	_ = w.Flush()
}

func printPluginSummaryTable(rows []pluginSummaryRow) {
	fmt.Println("Plugin summary (recent runs)")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PLUGIN\tMODE\tRUNS\tCOMPLETED\tFAILED\tAVG_DURATION_MS\tAVG_SCORE")
	for _, row := range rows {
		avgDuration := "-"
		if row.AvgDurationMs != nil {
			avgDuration = fmt.Sprintf("%.1f", *row.AvgDurationMs)
		}
		avgScore := "-"
		if row.AvgScore != nil {
			avgScore = fmt.Sprintf("%.6f", *row.AvgScore)
		}
		_, _ = fmt.Fprintf(
			w,
			"%s\t%s\t%d\t%d\t%d\t%s\t%s\n",
			row.PluginID,
			row.Mode,
			row.RunCount,
			row.CompletedCount,
			row.FailedCount,
			avgDuration,
			avgScore,
		)
	}
	_ = w.Flush()
}

func nullInt64Value(v sql.NullInt64) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}

func nullStringValue(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func nullStringValueWithDefault(v sql.NullString, defaultValue string) string {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return defaultValue
	}
	return v.String
}

package sqlitetool

import (
    "context"
    "database/sql"
    "fmt"
    "strings"
    "time"

    rootmw "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/turns"
    "github.com/invopop/jsonschema"
    _ "github.com/mattn/go-sqlite3"
    "github.com/rs/zerolog/log"
)

const (
    DataKeySQLiteDSN     = "sqlite_dsn"
    DataKeySQLitePrompts = "sqlite_prompts" // optional []string of system snippets
)

// DBLike abstracts *sql.DB (and compatible types) for this middleware.
type DBLike interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Config struct {
    DSN              string        // fallback DSN if Turn.Data not set
    DB               DBLike        // optional pre-opened DB (won't be closed by middleware)
    Name             string        // optional instance name; affects tool name
    ReadOnly         bool          // if true, enforce read-only (default true)
    MaxRows          int           // limit row returns
    ExecutionTimeout time.Duration // timeout for query execution
}

func DefaultConfig() Config { return Config{ReadOnly: true, MaxRows: 200, ExecutionTimeout: 20 * time.Second} }

// NewMiddleware loads schema/prompts from SQLite and advertises a sql_query tool. It also executes queries.
func NewMiddleware(cfg Config) rootmw.Middleware {
    return func(next rootmw.HandlerFunc) rootmw.HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            if t == nil {
                return next(ctx, t)
            }
            if t.Data == nil {
                t.Data = map[string]any{}
            }

            // Compute tool name per instance
            toolName := deriveToolName(cfg.Name)

            dsn := cfg.DSN
            if v, ok := t.Data[DataKeySQLiteDSN].(string); ok && v != "" {
                dsn = v
            }
            if dsn == "" {
                return next(ctx, t)
            }

            // Resolve DB
            var (
                db  DBLike
                err error
            )
            if cfg.DB != nil {
                db = cfg.DB
            } else {
                // Open DB (short-lived)
                opened, oerr := sql.Open("sqlite3", dsn)
                if oerr != nil {
                    log.Warn().Err(oerr).Msg("sqlitetool: failed to open sqlite")
                    return next(ctx, t)
                }
                defer opened.Close()
                db = opened
            }

            // Enforce read-only if configured (driver-level DSN may also be set by caller)
            if cfg.ReadOnly {
                if _, pErr := db.ExecContext(ctx, "PRAGMA query_only=ON"); pErr != nil {
                    log.Debug().Err(pErr).Msg("sqlitetool: failed to set PRAGMA query_only=ON")
                }
            }

            // Read schema as SQL
            schema, err := DumpSchemaSQL(ctx, db)
            if err != nil {
                log.Warn().Err(err).Msg("sqlitetool: failed to dump schema")
            } else if strings.TrimSpace(schema) != "" {
                header := "SQLite schema:\n\n"
                if cfg.Name != "" { header = fmt.Sprintf("SQLite schema [%s]:\n\n", cfg.Name) }
                turns.AppendBlock(t, turns.NewSystemTextBlock(header+schema))
                // Add access mode note
                mode := "read-only"
                if !cfg.ReadOnly { mode = "read-write" }
                turns.AppendBlock(t, turns.NewSystemTextBlock(fmt.Sprintf("SQLite access mode: %s. Use the %s tool accordingly.", mode, toolName)))
            }

            // Load optional prompts from _prompts table
            prompts, err := LoadPrompts(ctx, db)
            if err != nil {
                log.Debug().Err(err).Msg("sqlitetool: no prompts loaded")
            }
            if v, ok := t.Data[DataKeySQLitePrompts].([]string); ok {
                prompts = append(prompts, v...)
            }
            for _, p := range prompts {
                turns.AppendBlock(t, turns.NewSystemTextBlock(p))
            }

            // Ensure registry has sql_query tool definition (instance-scoped name)
            if regAny, ok := t.Data[turns.DataKeyToolRegistry]; ok && regAny != nil {
                if reg, ok := regAny.(tools.ToolRegistry); ok && reg != nil {
                    schemaObj := &jsonschema.Schema{Type: "object"}
                    props := jsonschema.NewProperties()
                    props.Set("sql", &jsonschema.Schema{Type: "string"})
                    schemaObj.Properties = props
                    schemaObj.Required = []string{"sql"}
                    desc := "Execute a read-only SQL query against the attached SQLite database."
                    if !cfg.ReadOnly { desc = "Execute a SQL query or statement against the attached SQLite database." }
                    _ = reg.RegisterTool(toolName, tools.ToolDefinition{
                        Name:        toolName,
                        Description: desc,
                        Parameters:  schemaObj,
                        Tags:        []string{"sqlite", "sql"},
                        Version:     "1.0",
                    })
                }
            }

            // Execute engine step to allow model to emit tool_calls
            updated, err := next(ctx, t)
            if err != nil {
                return updated, err
            }

            // Execute any pending sql_query calls directly (bypassing generic toolbox)
            // This mirrors tool middleware but inlined for the single tool.
            calls := extractPendingSQLQueries(updated, toolName)
            if len(calls) == 0 {
                return updated, nil
            }
            for _, c := range calls {
                resStr := ""
                q := strings.TrimSpace(c.SQL)
                if q != "" {
                    resStr = runSQLWithLimit(ctx, db, q, cfg.ReadOnly, cfg.MaxRows, cfg.ExecutionTimeout)
                }
                turns.AppendBlock(updated, turns.NewToolUseBlock(c.ID, resStr))
            }
            return updated, nil
        }
    }
}

// DumpSchemaSQL returns SQL CREATE statements for all user tables (excluding _prompts)
func DumpSchemaSQL(ctx context.Context, db DBLike) (string, error) {
    rows, err := db.QueryContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != '_prompts' ORDER BY name`)
    if err != nil { return "", err }
    defer rows.Close()
    var parts []string
    for rows.Next() {
        var sqlStr string
        if err := rows.Scan(&sqlStr); err != nil { return "", err }
        if strings.TrimSpace(sqlStr) != "" { parts = append(parts, sqlStr+";") }
    }
    return strings.Join(parts, "\n\n"), nil
}

// LoadPrompts loads optional prompts from a `_prompts` table with columns (prompt TEXT)
func LoadPrompts(ctx context.Context, db DBLike) ([]string, error) {
    _, err := db.ExecContext(ctx, `SELECT 1 FROM _prompts LIMIT 1`)
    if err != nil { return nil, err }
    rows, err := db.QueryContext(ctx, `SELECT prompt FROM _prompts`)
    if err != nil { return nil, err }
    defer rows.Close()
    var prompts []string
    for rows.Next() {
        var p string
        if err := rows.Scan(&p); err != nil { return nil, err }
        prompts = append(prompts, p)
    }
    return prompts, nil
}

type sqlCall struct{ ID, SQL string }

func deriveToolName(name string) string {
    if n := strings.TrimSpace(name); n != "" {
        // normalize to safe suffix
        n = strings.ToLower(n)
        n = strings.ReplaceAll(n, " ", "_")
        return "sql_query_" + n
    }
    return "sql_query"
}

func extractPendingSQLQueries(t *turns.Turn, toolName string) []sqlCall {
    used := map[string]bool{}
    for _, b := range t.Blocks {
        if b.Kind == turns.BlockKindToolUse {
            if id, _ := b.Payload[turns.PayloadKeyID].(string); id != "" { used[id] = true }
        }
    }
    var ret []sqlCall
    for _, b := range t.Blocks {
        if b.Kind != turns.BlockKindToolCall { continue }
        id, _ := b.Payload[turns.PayloadKeyID].(string)
        if id == "" || used[id] { continue }
        name, _ := b.Payload[turns.PayloadKeyName].(string)
        if name != toolName { continue }
        sqlStr := ""
        if args, ok := b.Payload[turns.PayloadKeyArgs].(map[string]any); ok {
            if s, ok := args["sql"].(string); ok { sqlStr = s }
        }
        ret = append(ret, sqlCall{ID: id, SQL: sqlStr})
    }
    return ret
}

func runQueryWithLimit(ctx context.Context, db DBLike, sqlStr string, maxRows int, timeout time.Duration) string {
    cctx := ctx
    cancel := func(){}
    if timeout > 0 { cctx, cancel = context.WithTimeout(ctx, timeout) }
    defer cancel()
    rows, err := db.QueryContext(cctx, sqlStr)
    if err != nil { return fmt.Sprintf("error: %v", err) }
    defer rows.Close()
    cols, err := rows.Columns()
    if err != nil { return fmt.Sprintf("error: %v", err) }
    var out []string
    out = append(out, strings.Join(cols, " | "))
    count := 0
    for rows.Next() {
        if maxRows > 0 && count >= maxRows { break }
        vals := make([]any, len(cols))
        ptrs := make([]any, len(cols))
        for i := range vals { ptrs[i] = &vals[i] }
        if err := rows.Scan(ptrs...); err != nil { return fmt.Sprintf("error: %v", err) }
        var parts []string
        for _, v := range vals {
            parts = append(parts, fmt.Sprintf("%v", v))
        }
        out = append(out, strings.Join(parts, " | "))
        count++
    }
    return strings.Join(out, "\n")
}

// runSQLWithLimit enforces read-only vs read-write behavior and returns a human-readable result string.
func runSQLWithLimit(ctx context.Context, db DBLike, sqlStr string, readOnly bool, maxRows int, timeout time.Duration) string {
    // crude allowlist for read-only: allow SELECT/WITH/EXPLAIN and read-only PRAGMA
    upper := strings.ToUpper(strings.TrimSpace(sqlStr))
    if readOnly {
        if strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "WITH ") || strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "PRAGMA ") {
            return runQueryWithLimit(ctx, db, sqlStr, maxRows, timeout)
        }
        return "error: read-only mode; only SELECT/WITH/EXPLAIN/PRAGMA allowed"
    }
    // read-write: for SELECT-like, query; otherwise exec
    if strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "WITH ") || strings.HasPrefix(upper, "EXPLAIN") {
        return runQueryWithLimit(ctx, db, sqlStr, maxRows, timeout)
    }
    // Exec path
    cctx := ctx
    cancel := func(){}
    if timeout > 0 { cctx, cancel = context.WithTimeout(ctx, timeout) }
    defer cancel()
    res, err := db.ExecContext(cctx, sqlStr)
    if err != nil { return fmt.Sprintf("error: %v", err) }
    n, _ := res.RowsAffected()
    return fmt.Sprintf("rows_affected=%d", n)
}



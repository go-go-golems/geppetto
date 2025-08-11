package sqlitetool

import (
    "context"
    "database/sql"
    "fmt"
    "strings"
    "time"

    rootmw "github.com/go-go-golems/geppetto/pkg/inference/middleware"
    "github.com/go-go-golems/geppetto/pkg/inference/tools"
    "github.com/go-go-golems/geppetto/pkg/events"
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
    MaxRows          int           // limit row returns
    ExecutionTimeout time.Duration // timeout for query execution
    // Additional output safety limits (applied to the textual result rendering)
    MaxOutputLines   int           // maximum number of output lines including header (default: 50)
    MaxOutputBytes   int           // maximum number of output bytes (default: 1024)
}

func DefaultConfig() Config { return Config{MaxRows: 200, ExecutionTimeout: 20 * time.Second, MaxOutputLines: 50, MaxOutputBytes: 1024} }

// NewMiddleware loads schema/prompts from SQLite and advertises a sql_query tool. It also executes queries.
func NewMiddleware(cfg Config) rootmw.Middleware {
    // Pre-compute tool description once at middleware creation
    toolDesc := "Execute a read-only SQL query against the attached SQLite database."
    func() {
        var (
            db  DBLike
            err error
            opened *sql.DB
        )
        // Prefer provided DB, else open from DSN if available
        if cfg.DB != nil {
            db = cfg.DB
        } else if cfg.DSN != "" {
            opened, err = sql.Open("sqlite3", cfg.DSN)
            if err != nil {
                log.Debug().Err(err).Msg("sqlitetool: could not open DB during init; using generic description")
                return
            }
            db = opened
            defer opened.Close()
        } else {
            return
        }

        ctx := context.Background()
        schema, err := DumpSchemaSQL(ctx, db)
        if err != nil {
            log.Debug().Err(err).Msg("sqlitetool: failed to dump schema during init")
        }
        prompts, err := LoadPrompts(ctx, db)
        if err != nil {
            // benign; no prompts table
        }

        var b strings.Builder
        b.WriteString(toolDesc)
        if strings.TrimSpace(schema) != "" {
            b.WriteString("\n\nSchema:\n\n")
            b.WriteString(schema)
        }
        if len(prompts) > 0 {
            b.WriteString("\n\nPrompts:\n")
            for _, p := range prompts {
                if s := strings.TrimSpace(p); s != "" {
                    b.WriteString("\n- ")
                    b.WriteString(s)
                }
            }
            b.WriteString("\n")
        }
        toolDesc = b.String()
    }()

    return func(next rootmw.HandlerFunc) rootmw.HandlerFunc {
        return func(ctx context.Context, t *turns.Turn) (*turns.Turn, error) {
            if t == nil {
                return next(ctx, t)
            }
            if t.Data == nil {
                t.Data = map[string]any{}
            }

            // Determine if the tool should be available for this turn; check DSN presence
            dsn := cfg.DSN
            if v, ok := t.Data[DataKeySQLiteDSN].(string); ok && v != "" {
                dsn = v
            }
            if dsn == "" && cfg.DB == nil {
                // No database configured for this turn
                return next(ctx, t)
            }

            // Ensure registry has sql_query tool definition
            if regAny, ok := t.Data[turns.DataKeyToolRegistry]; ok && regAny != nil {
                if reg, ok := regAny.(tools.ToolRegistry); ok && reg != nil {
                    schemaObj := &jsonschema.Schema{Type: "object"}
                    props := jsonschema.NewProperties()
                    props.Set("sql", &jsonschema.Schema{Type: "string"})
                    schemaObj.Properties = props
                    schemaObj.Required = []string{"sql"}
                    _ = reg.RegisterTool("sql_query", tools.ToolDefinition{
                        Name:        "sql_query",
                        Description: toolDesc,
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
            calls := extractPendingSQLQueries(updated)
            if len(calls) == 0 {
                return updated, nil
            }
            // Resolve DB for execution
            var execDB DBLike
            if cfg.DB != nil {
                execDB = cfg.DB
            } else {
                opened, oerr := sql.Open("sqlite3", dsn)
                if oerr != nil {
                    log.Warn().Err(oerr).Msg("sqlitetool: failed to open sqlite for execution")
                    return updated, nil
                }
                defer opened.Close()
                execDB = opened
            }
            for _, c := range calls {
                resStr := ""
                q := strings.TrimSpace(c.SQL)
                if q != "" {
                    // Emit execution-start event
                    events.PublishEventToContext(ctx, events.NewToolCallExecuteEvent(
                        events.EventMetadata{RunID: updated.RunID, TurnID: updated.ID},
                        events.ToolCall{ID: c.ID, Name: "sql_query", Input: fmt.Sprintf("{\"sql\":%q}", q)},
                    ))

                    // Ensure sensible defaults if zero
                    maxLines := cfg.MaxOutputLines
                    if maxLines <= 0 { maxLines = 50 }
                    maxBytes := cfg.MaxOutputBytes
                    if maxBytes <= 0 { maxBytes = 1024 }
                    resStr = runQueryWithLimit(ctx, execDB, q, cfg.MaxRows, maxLines, maxBytes, cfg.ExecutionTimeout)
                }
                turns.AppendBlock(updated, turns.NewToolUseBlock(c.ID, resStr))
                // Emit execution-result event
                events.PublishEventToContext(ctx, events.NewToolCallExecutionResultEvent(
                    events.EventMetadata{RunID: updated.RunID, TurnID: updated.ID},
                    events.ToolResult{ID: c.ID, Result: resStr},
                ))
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

func extractPendingSQLQueries(t *turns.Turn) []sqlCall {
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
        if name != "sql_query" { continue }
        sqlStr := ""
        if args, ok := b.Payload[turns.PayloadKeyArgs].(map[string]any); ok {
            if s, ok := args["sql"].(string); ok { sqlStr = s }
        }
        ret = append(ret, sqlCall{ID: id, SQL: sqlStr})
    }
    return ret
}

func runQueryWithLimit(ctx context.Context, db DBLike, sqlStr string, maxRows int, maxLines int, maxBytes int, timeout time.Duration) string {
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
    totalBytes := len(out[0])
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
        line := strings.Join(parts, " | ")
        out = append(out, line)
        totalBytes += len(line) + 1 // include newline
        count++
        if maxLines > 0 && len(out) >= maxLines { break }
        if maxBytes > 0 && totalBytes >= maxBytes { break }
    }
    result := strings.Join(out, "\n")
    truncated := false
    if (maxLines > 0 && len(out) >= maxLines) || (maxBytes > 0 && len(result) >= maxBytes) {
        truncated = true
    }
    if truncated {
        // Compute approximate KB cutoff message
        kb := (len(result) + 1023) / 1024
        if kb == 0 { kb = 1 }
        result = result + fmt.Sprintf("\n... additional data cutoff (%d kB)", kb)
    }
    return result
}



package scopeddb

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
)

type QueryOptions struct {
	MaxRows        int
	MaxColumns     int
	MaxCellChars   int
	Timeout        time.Duration
	RequireOrderBy bool
}

type QueryInput struct {
	SQL    string   `json:"sql" jsonschema:"description=Single SELECT or WITH query only against allowed scope tables/views. Prefer ? placeholders plus params instead of inline literal values.,required"`
	Params []string `json:"params,omitempty" jsonschema:"description=Optional bind parameters as strings. Prefer these over inline literal values in SQL."`
}

type QueryOutput struct {
	Columns   []string         `json:"columns"`
	Rows      []map[string]any `json:"rows"`
	Count     int              `json:"count"`
	Truncated bool             `json:"truncated,omitempty"`
	Error     string           `json:"error,omitempty"`
}

type QueryRunner struct {
	db             *sql.DB
	allowedObjects map[string]struct{}
	opts           QueryOptions
}

func DefaultQueryOptions() QueryOptions {
	return QueryOptions{
		MaxRows:        100,
		MaxColumns:     64,
		MaxCellChars:   2000,
		Timeout:        5 * time.Second,
		RequireOrderBy: false,
	}
}

func WithDefaultQueryOptions(opts QueryOptions) QueryOptions {
	ret := opts
	def := DefaultQueryOptions()
	if ret.MaxRows <= 0 {
		ret.MaxRows = def.MaxRows
	}
	if ret.MaxColumns <= 0 {
		ret.MaxColumns = def.MaxColumns
	}
	if ret.MaxCellChars <= 0 {
		ret.MaxCellChars = def.MaxCellChars
	}
	if ret.Timeout <= 0 {
		ret.Timeout = def.Timeout
	}
	return ret
}

func NormalizeCell(v any, maxChars int) any {
	switch typed := v.(type) {
	case nil:
		return nil
	case []byte:
		s := string(typed)
		if maxChars > 0 && len(s) > maxChars {
			return s[:maxChars]
		}
		return s
	case string:
		if maxChars > 0 && len(typed) > maxChars {
			return typed[:maxChars]
		}
		return typed
	default:
		return typed
	}
}

func NewQueryRunner(db *sql.DB, allowedObjects map[string]struct{}, opts QueryOptions) (*QueryRunner, error) {
	if db == nil {
		return nil, fmt.Errorf("tool db is nil")
	}
	return &QueryRunner{
		db:             db,
		allowedObjects: normalizeAllowedObjects(allowedObjects),
		opts:           WithDefaultQueryOptions(opts),
	}, nil
}

func (r *QueryRunner) Run(ctx context.Context, in QueryInput) (QueryOutput, error) {
	if r == nil || r.db == nil {
		return QueryOutput{Error: "tool db is not initialized"}, nil
	}
	sqlText := TrimOptionalTrailingSemicolon(in.SQL)
	if err := validateQuery(sqlText, r.allowedObjects, r.opts); err != nil {
		return QueryOutput{Error: err.Error()}, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	qctx, cancel := context.WithTimeout(ctx, r.opts.Timeout)
	defer cancel()

	conn, err := r.db.Conn(qctx)
	if err != nil {
		return QueryOutput{Error: err.Error()}, nil
	}
	defer func() { _ = conn.Close() }()

	if err := setSQLiteAuthorizer(conn, newToolDBAuthorizer(r.allowedObjects)); err != nil {
		return QueryOutput{Error: err.Error()}, nil
	}
	defer func() {
		_ = setSQLiteAuthorizer(conn, nil)
	}()

	if err := ensureReadonlyPreparedQuery(conn, sqlText); err != nil {
		return QueryOutput{Error: err.Error()}, nil
	}

	args := make([]any, 0, len(in.Params))
	for _, param := range in.Params {
		args = append(args, param)
	}
	rows, err := conn.QueryContext(qctx, sqlText, args...)
	if err != nil {
		return QueryOutput{Error: err.Error()}, nil
	}
	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	if err != nil {
		return QueryOutput{Error: err.Error()}, nil
	}
	if len(cols) > r.opts.MaxColumns {
		return QueryOutput{Error: fmt.Sprintf("query returns %d columns; max is %d", len(cols), r.opts.MaxColumns)}, nil
	}

	out := QueryOutput{
		Columns: cols,
		Rows:    make([]map[string]any, 0, minInt(r.opts.MaxRows, 64)),
	}

	for rows.Next() {
		if out.Count >= r.opts.MaxRows {
			out.Truncated = true
			break
		}
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return QueryOutput{Error: err.Error()}, nil
		}

		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = NormalizeCell(values[i], r.opts.MaxCellChars)
		}
		out.Rows = append(out.Rows, row)
		out.Count++
	}
	if err := rows.Err(); err != nil {
		return QueryOutput{Error: err.Error()}, nil
	}

	return out, nil
}

func validateQuery(sqlText string, allowedObjects map[string]struct{}, opts QueryOptions) error {
	sqlText = TrimOptionalTrailingSemicolon(sqlText)
	if sqlText == "" {
		return fmt.Errorf("sql is required")
	}
	sanitizedText := TrimOptionalTrailingSemicolon(stripSQLLiteralsAndComments(sqlText))
	if strings.Contains(sanitizedText, ";") {
		return fmt.Errorf("multiple statements are not allowed")
	}
	lower := strings.ToLower(strings.TrimSpace(sanitizedText))
	if !strings.HasPrefix(lower, "select ") && !strings.HasPrefix(lower, "with ") {
		return fmt.Errorf("only SELECT queries are allowed")
	}
	if opts.RequireOrderBy && strings.Contains(lower, " from ") && !strings.Contains(lower, " order by ") {
		return fmt.Errorf("query must include ORDER BY for deterministic row ordering")
	}
	if len(allowedObjects) == 0 {
		return nil
	}
	for _, ref := range referencedObjects(sanitizedText) {
		if _, ok := allowedObjects[NormalizeObjectName(ref)]; !ok {
			return fmt.Errorf("query references disallowed table/view %q", ref)
		}
	}
	return nil
}

var fromJoinObjectRe = regexp.MustCompile(`(?i)\b(from|join)\s+([a-zA-Z_][a-zA-Z0-9_\.]*)`)

func referencedObjects(sqlText string) []string {
	matches := fromJoinObjectRe.FindAllStringSubmatch(sqlText, -1)
	cteAliases := referencedCTEAliases(sqlText)
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		obj := normalizeExtractedObject(match[2])
		if obj == "" {
			continue
		}
		if _, ok := cteAliases[NormalizeObjectName(obj)]; ok {
			continue
		}
		if _, ok := seen[obj]; ok {
			continue
		}
		seen[obj] = struct{}{}
		out = append(out, obj)
	}
	return out
}

func normalizeExtractedObject(v string) string {
	t := strings.TrimSpace(v)
	t = strings.Trim(t, "`\"[]")
	if t == "" {
		return ""
	}
	if dot := strings.LastIndex(t, "."); dot >= 0 && dot+1 < len(t) {
		t = t[dot+1:]
	}
	return strings.TrimSpace(t)
}

func referencedCTEAliases(sqlText string) map[string]struct{} {
	lower := strings.ToLower(strings.TrimSpace(sqlText))
	if !strings.HasPrefix(lower, "with ") {
		return nil
	}

	i := len("with")
	for i < len(sqlText) && isSQLSpace(sqlText[i]) {
		i++
	}
	if strings.HasPrefix(strings.ToLower(sqlText[i:]), "recursive") {
		i += len("recursive")
	}

	ret := map[string]struct{}{}
	for i < len(sqlText) {
		for i < len(sqlText) && isSQLSpace(sqlText[i]) {
			i++
		}

		start := i
		for i < len(sqlText) && isSQLIdentifierChar(sqlText[i], i == start) {
			i++
		}
		if start == i {
			return ret
		}

		name := NormalizeObjectName(sqlText[start:i])
		if name != "" {
			ret[name] = struct{}{}
		}

		for i < len(sqlText) && isSQLSpace(sqlText[i]) {
			i++
		}
		if i < len(sqlText) && sqlText[i] == '(' {
			end, ok := consumeBalancedParens(sqlText, i)
			if !ok {
				return ret
			}
			i = end
		}

		for i < len(sqlText) && isSQLSpace(sqlText[i]) {
			i++
		}
		if !strings.HasPrefix(strings.ToLower(sqlText[i:]), "as") {
			return ret
		}
		i += len("as")
		for i < len(sqlText) && isSQLSpace(sqlText[i]) {
			i++
		}
		if i >= len(sqlText) || sqlText[i] != '(' {
			return ret
		}

		end, ok := consumeBalancedParens(sqlText, i)
		if !ok {
			return ret
		}
		i = end

		for i < len(sqlText) && isSQLSpace(sqlText[i]) {
			i++
		}
		if i >= len(sqlText) || sqlText[i] != ',' {
			return ret
		}
		i++
	}

	return ret
}

func isSQLSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

func isSQLIdentifierChar(b byte, first bool) bool {
	if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b == '_' {
		return true
	}
	if !first && b >= '0' && b <= '9' {
		return true
	}
	return false
}

func consumeBalancedParens(sqlText string, start int) (int, bool) {
	if start >= len(sqlText) || sqlText[start] != '(' {
		return start, false
	}
	depth := 0
	for i := start; i < len(sqlText); i++ {
		switch sqlText[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1, true
			}
		}
	}
	return len(sqlText), false
}

func stripSQLLiteralsAndComments(sqlText string) string {
	if sqlText == "" {
		return ""
	}

	const (
		stateNormal = iota
		stateSingleQuote
		stateDoubleQuote
		stateBacktickQuote
		stateBracketQuote
		stateLineComment
		stateBlockComment
	)

	var b strings.Builder
	b.Grow(len(sqlText))

	state := stateNormal
	for i := 0; i < len(sqlText); i++ {
		ch := sqlText[i]
		next := byte(0)
		if i+1 < len(sqlText) {
			next = sqlText[i+1]
		}

		switch state {
		case stateNormal:
			switch {
			case ch == '\'':
				state = stateSingleQuote
				b.WriteByte(' ')
			case ch == '"':
				state = stateDoubleQuote
				b.WriteByte(' ')
			case ch == '`':
				state = stateBacktickQuote
				b.WriteByte(' ')
			case ch == '[':
				state = stateBracketQuote
				b.WriteByte(' ')
			case ch == '-' && next == '-':
				state = stateLineComment
				b.WriteString("  ")
				i++
			case ch == '/' && next == '*':
				state = stateBlockComment
				b.WriteString("  ")
				i++
			default:
				b.WriteByte(ch)
			}
		case stateSingleQuote:
			if ch == '\'' && next == '\'' {
				b.WriteString("  ")
				i++
				continue
			}
			if ch == '\'' {
				state = stateNormal
			}
			b.WriteByte(' ')
		case stateDoubleQuote:
			if ch == '"' && next == '"' {
				b.WriteString("  ")
				i++
				continue
			}
			if ch == '"' {
				state = stateNormal
			}
			b.WriteByte(' ')
		case stateBacktickQuote:
			if ch == '`' {
				state = stateNormal
			}
			b.WriteByte(' ')
		case stateBracketQuote:
			if ch == ']' {
				state = stateNormal
			}
			b.WriteByte(' ')
		case stateLineComment:
			if ch == '\n' {
				state = stateNormal
				b.WriteByte('\n')
				continue
			}
			b.WriteByte(' ')
		case stateBlockComment:
			if ch == '*' && next == '/' {
				state = stateNormal
				b.WriteString("  ")
				i++
				continue
			}
			if ch == '\n' {
				b.WriteByte('\n')
				continue
			}
			b.WriteByte(' ')
		}
	}

	return b.String()
}

func setSQLiteAuthorizer(conn *sql.Conn, callback func(int, string, string, string) int) error {
	if conn == nil {
		return fmt.Errorf("sql connection is nil")
	}
	return conn.Raw(func(driverConn any) error {
		sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("unexpected sqlite driver connection type %T", driverConn)
		}
		sqliteConn.RegisterAuthorizer(callback)
		return nil
	})
}

func ensureReadonlyPreparedQuery(conn *sql.Conn, sqlText string) error {
	if conn == nil {
		return fmt.Errorf("sql connection is nil")
	}
	return conn.Raw(func(driverConn any) error {
		sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("unexpected sqlite driver connection type %T", driverConn)
		}

		stmtDriver, err := sqliteConn.Prepare(sqlText)
		if err != nil {
			return err
		}
		defer func() { _ = stmtDriver.Close() }()

		stmt, ok := stmtDriver.(*sqlite3.SQLiteStmt)
		if !ok {
			return fmt.Errorf("unexpected sqlite statement type %T", stmtDriver)
		}
		if !stmt.Readonly() {
			return fmt.Errorf("only read-only SELECT queries are allowed")
		}
		return nil
	})
}

func newToolDBAuthorizer(allowedObjects map[string]struct{}) func(int, string, string, string) int {
	return func(op int, arg1, arg2, arg3 string) int {
		switch op {
		case sqlite3.SQLITE_SELECT:
			return sqlite3.SQLITE_OK
		case sqlite3.SQLITE_READ:
			// Object allow-listing happens in validateQuery by inspecting the SQL text.
			// The go-sqlite3 authorizer callback does not expose SQLite's source object
			// (for example the view name that triggered a base-table read), so denying
			// reads here would incorrectly reject queries against allowed views.
			return sqlite3.SQLITE_OK
		case sqlite3.SQLITE_INSERT,
			sqlite3.SQLITE_UPDATE,
			sqlite3.SQLITE_DELETE,
			sqlite3.SQLITE_PRAGMA,
			sqlite3.SQLITE_ATTACH,
			sqlite3.SQLITE_DETACH,
			sqlite3.SQLITE_TRANSACTION,
			sqlite3.SQLITE_CREATE_INDEX,
			sqlite3.SQLITE_CREATE_TABLE,
			sqlite3.SQLITE_CREATE_TEMP_INDEX,
			sqlite3.SQLITE_CREATE_TEMP_TABLE,
			sqlite3.SQLITE_CREATE_TEMP_TRIGGER,
			sqlite3.SQLITE_CREATE_TEMP_VIEW,
			sqlite3.SQLITE_CREATE_TRIGGER,
			sqlite3.SQLITE_CREATE_VIEW,
			sqlite3.SQLITE_CREATE_VTABLE,
			sqlite3.SQLITE_DROP_INDEX,
			sqlite3.SQLITE_DROP_TABLE,
			sqlite3.SQLITE_DROP_TEMP_INDEX,
			sqlite3.SQLITE_DROP_TEMP_TABLE,
			sqlite3.SQLITE_DROP_TEMP_TRIGGER,
			sqlite3.SQLITE_DROP_TEMP_VIEW,
			sqlite3.SQLITE_DROP_TRIGGER,
			sqlite3.SQLITE_DROP_VIEW,
			sqlite3.SQLITE_DROP_VTABLE:
			return sqlite3.SQLITE_DENY
		default:
			return sqlite3.SQLITE_OK
		}
	}
}

func normalizeAllowedObjects(allowedObjects map[string]struct{}) map[string]struct{} {
	if len(allowedObjects) == 0 {
		return nil
	}
	ret := make(map[string]struct{}, len(allowedObjects))
	for value := range allowedObjects {
		normalized := NormalizeObjectName(value)
		if normalized == "" {
			continue
		}
		ret[normalized] = struct{}{}
	}
	return ret
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

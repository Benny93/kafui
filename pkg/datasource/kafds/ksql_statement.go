package kafds

import (
	"regexp"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
)

// ksqlStatementKind routes a validated statement to the correct ksqlDB endpoint.
type ksqlStatementKind int

const (
	ksqlKindQuery     ksqlStatementKind = iota // SELECT ... -> /query
	ksqlKindStatement                          // everything else -> /ksql
)

// knownKsqlKeywords are the statement-initial keywords ksqlDB accepts (besides
// SELECT which is classified as a query). Anything else is rejected as invalid.
var knownKsqlKeywords = map[string]bool{
	"CREATE":    true,
	"INSERT":    true,
	"DROP":      true,
	"TERMINATE": true,
	"SHOW":      true,
	"LIST":      true,
	"DESCRIBE":  true,
	"EXPLAIN":   true,
	"SET":       true,
	"UNSET":     true,
	"ALTER":     true,
	"ASSERT":    true,
	"PAUSE":     true,
	"RESUME":    true,
}

// unsupportedKsqlKeywords are recognizable but intentionally unsupported here.
var unsupportedKsqlKeywords = map[string]bool{
	"PRINT":    true,
	"DEFINE":   true,
	"UNDEFINE": true,
}

var (
	ksqlLineComment  = regexp.MustCompile(`(?m)--.*$`)
	ksqlBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// classifyKsqlStatement trims comments/whitespace and validates a single ksqlDB
// statement, returning its routing kind. On any validation failure it returns an
// error-flagged KsqlResultTable (never nil) describing the problem; callers must
// surface that table and must not reach the server. On success the returned
// *KsqlResultTable is nil.
func classifyKsqlStatement(sql string) (ksqlStatementKind, *api.KsqlResultTable) {
	cleaned := ksqlBlockComment.ReplaceAllString(sql, " ")
	cleaned = ksqlLineComment.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return ksqlKindStatement, ksqlValidationError("no valid statement was found")
	}

	// Reject more than one ;-terminated statement. A single trailing ; is fine.
	trimmed := strings.TrimRight(cleaned, "; \t\n\r")
	if strings.Contains(trimmed, ";") {
		return ksqlKindStatement, ksqlValidationError("only a single statement is supported")
	}

	fields := strings.Fields(trimmed)
	keyword := strings.ToUpper(fields[0])

	switch {
	case keyword == "SELECT":
		return ksqlKindQuery, nil
	case unsupportedKsqlKeywords[keyword]:
		return ksqlKindStatement, ksqlValidationError("statement type is unsupported")
	case knownKsqlKeywords[keyword]:
		return ksqlKindStatement, nil
	default:
		return ksqlKindStatement, ksqlValidationError("statement type is unsupported")
	}
}

// ksqlValidationError builds an error-flagged result table for a validation
// failure surfaced to the UI in place of a server round-trip.
func ksqlValidationError(msg string) *api.KsqlResultTable {
	return &api.KsqlResultTable{
		Title:   "Validation error",
		Columns: []string{"Error"},
		Rows:    [][]string{{msg}},
		IsError: true,
	}
}

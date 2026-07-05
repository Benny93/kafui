package kafds

import (
	"context"
	"net/http"

	"github.com/Benny93/kafui/pkg/api"
)

// executeStatement runs a non-SELECT (statement-kind) input against /ksql and
// interprets the typed response into result tables. Connection failures and
// server errors are converted into error tables so they travel down the same
// channel as data (KS-7).
func executeStatement(ctx context.Context, c *ksqlClient, sql string, props map[string]string) []api.KsqlResultTable {
	status, body, err := c.doRaw(ctx, http.MethodPost, "/ksql", ksqlStatementRequest(sql, props))
	if err != nil {
		return []api.KsqlResultTable{ksqlErrorTable(err.Error())}
	}
	return interpretStatementResponse(status, body)
}

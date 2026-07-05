package kafds

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
)

// ksqlListEntry models one element of a /ksql listing response. ksqlDB reports
// streams and tables as an array with a single @type entry carrying the array;
// older servers report a single legacy `format` field instead of key/value
// formats.
type ksqlListEntry struct {
	Type    string `json:"@type"`
	Streams []struct {
		Name        string `json:"name"`
		Topic       string `json:"topic"`
		KeyFormat   string `json:"keyFormat"`
		ValueFormat string `json:"valueFormat"`
		Format      string `json:"format"` // legacy single-format field
	} `json:"streams"`
	Tables []struct {
		Name        string `json:"name"`
		Topic       string `json:"topic"`
		KeyFormat   string `json:"keyFormat"`
		ValueFormat string `json:"valueFormat"`
		Format      string `json:"format"` // legacy single-format field
		IsWindowed  bool   `json:"isWindowed"`
	} `json:"tables"`
}

// ListKsqlStreams posts LIST STREAMS; to /ksql and maps the response. A response
// that is not a recognizable streams listing yields a descriptive error.
func (kp KafkaDataSourceKaf) ListKsqlStreams() ([]api.KsqlStream, error) {
	c, err := kp.ksqlClient()
	if err != nil {
		return nil, err
	}
	var resp []ksqlListEntry
	if err := c.doPost("/ksql", ksqlStatementRequest("LIST STREAMS;", nil), &resp); err != nil {
		return nil, fmt.Errorf("listing ksql streams: %w", err)
	}
	for _, e := range resp {
		if e.Streams == nil && e.Type != "streams" {
			continue
		}
		out := make([]api.KsqlStream, 0, len(e.Streams))
		for _, s := range e.Streams {
			out = append(out, api.KsqlStream{
				Name:        s.Name,
				Topic:       s.Topic,
				KeyFormat:   s.KeyFormat,
				ValueFormat: firstNonEmpty(s.ValueFormat, s.Format),
			})
		}
		return out, nil
	}
	return nil, fmt.Errorf("stream list could not be retrieved")
}

// ListKsqlTables posts LIST TABLES; to /ksql and maps the response, including the
// windowed flag. A response that is not a recognizable tables listing yields a
// descriptive error.
func (kp KafkaDataSourceKaf) ListKsqlTables() ([]api.KsqlTable, error) {
	c, err := kp.ksqlClient()
	if err != nil {
		return nil, err
	}
	var resp []ksqlListEntry
	if err := c.doPost("/ksql", ksqlStatementRequest("LIST TABLES;", nil), &resp); err != nil {
		return nil, fmt.Errorf("listing ksql tables: %w", err)
	}
	for _, e := range resp {
		if e.Tables == nil && e.Type != "tables" {
			continue
		}
		out := make([]api.KsqlTable, 0, len(e.Tables))
		for _, t := range e.Tables {
			out = append(out, api.KsqlTable{
				Name:        t.Name,
				Topic:       t.Topic,
				KeyFormat:   t.KeyFormat,
				ValueFormat: firstNonEmpty(t.ValueFormat, t.Format),
				Windowed:    t.IsWindowed,
			})
		}
		return out, nil
	}
	return nil, fmt.Errorf("table list could not be retrieved")
}

// ksqlStatementRequest builds the /ksql request body. An empty/nil property map
// yields an empty streamsProperties object.
func ksqlStatementRequest(sql string, props map[string]string) map[string]interface{} {
	p := map[string]string{}
	for k, v := range props {
		p[k] = v
	}
	return map[string]interface{}{"ksql": sql, "streamsProperties": p}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

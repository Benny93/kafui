package kafds

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
)

const registryContentType = "application/vnd.schemaregistry.v1+json"

// registryError carries a schema-registry non-2xx response. The registry returns
// {"error_code": ..., "message": ...} bodies; ErrorCode is 0 when unparseable.
type registryError struct {
	StatusCode int
	ErrorCode  int
	Message    string
}

func (e *registryError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("schema registry returned %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("schema registry returned %d", e.StatusCode)
}

// registryClient is a small HTTP client for the schema registry supporting all
// verbs, basic auth, connection-level failover across multiple base URLs, and
// typed error mapping. Construct via newRegistryClient.
type registryClient struct {
	baseURLs []string
	lastGood int
	username string
	password string
	http     *http.Client
}

// newRegistryClient builds a client from the active cluster's schema-registry
// configuration. The URL field may be a comma-separated list for failover.
// Returns (nil, nil) when no registry is configured — callers decide whether
// that is an empty listing or a SchemaRegistryNotConfiguredError.
func (kp KafkaDataSourceKaf) newRegistryClient() (*registryClient, error) {
	if currentCluster == nil {
		return nil, nil
	}
	var urls []string
	for _, u := range strings.Split(currentCluster.SchemaRegistryURL, ",") {
		if u = strings.TrimRight(strings.TrimSpace(u), "/"); u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		return nil, nil
	}

	rc := &registryClient{
		baseURLs: urls,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
	if creds := currentCluster.SchemaRegistryCredentials; creds != nil {
		rc.username = creds.Username
		rc.password = creds.Password
	}
	return rc, nil
}

// do executes a request against the registry, trying each configured base URL in
// turn on connection-level failures (SR-3). An HTTP response — even 4xx/5xx —
// stops failover and is returned as a *registryError on non-2xx. On success the
// body is decoded into out when out is non-nil.
func (rc *registryClient) do(method, path string, body, out interface{}) error {
	var reqBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding schema registry request: %w", err)
		}
		reqBody = b
	}

	var connErr error
	n := len(rc.baseURLs)
	for i := 0; i < n; i++ {
		idx := (rc.lastGood + i) % n
		base := rc.baseURLs[idx]

		var rdr io.Reader
		if reqBody != nil {
			rdr = bytes.NewReader(reqBody)
		}
		req, err := http.NewRequest(method, base+path, rdr)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", registryContentType)
		if reqBody != nil {
			req.Header.Set("Content-Type", registryContentType)
		}
		if rc.username != "" {
			req.SetBasicAuth(rc.username, rc.password)
		}

		resp, err := rc.http.Do(req)
		if err != nil {
			// Connection-level failure: remember and try the next URL.
			connErr = err
			continue
		}
		rc.lastGood = idx
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			re := &registryError{StatusCode: resp.StatusCode}
			var parsed struct {
				ErrorCode int    `json:"error_code"`
				Message   string `json:"message"`
			}
			if json.Unmarshal(respBody, &parsed) == nil {
				re.ErrorCode = parsed.ErrorCode
				re.Message = parsed.Message
			}
			if re.Message == "" {
				re.Message = strings.TrimSpace(string(respBody))
			}
			return re
		}
		if out != nil && len(respBody) > 0 {
			return json.Unmarshal(respBody, out)
		}
		return nil
	}
	return fmt.Errorf("no live schema registry instances reachable (%d configured): %w", n, connErr)
}

func (rc *registryClient) doGet(path string, out interface{}) error {
	return rc.do(http.MethodGet, path, nil, out)
}
func (rc *registryClient) doPost(path string, body, out interface{}) error {
	return rc.do(http.MethodPost, path, body, out)
}
func (rc *registryClient) doPut(path string, body, out interface{}) error {
	return rc.do(http.MethodPut, path, body, out)
}
func (rc *registryClient) doDelete(path string, out interface{}) error {
	return rc.do(http.MethodDelete, path, nil, out)
}

// mapRegistryError converts a *registryError into a typed api error using the
// call's subject/version context. Non-registry errors pass through unchanged.
func mapRegistryError(err error, subject string, version int) error {
	if err == nil {
		return nil
	}
	var re *registryError
	if !errors.As(err, &re) {
		return err
	}
	switch {
	case re.ErrorCode == 40402:
		return api.SchemaVersionNotFoundError{Subject: subject, Version: version, Cause: re}
	case re.ErrorCode == 40401 || re.ErrorCode == 40403 || re.StatusCode == http.StatusNotFound:
		return api.SubjectNotFoundError{Subject: subject, Cause: re}
	case re.StatusCode == http.StatusConflict:
		return api.SchemaIncompatibleError{Subject: subject, Message: re.Message, Cause: re}
	case re.StatusCode == http.StatusUnprocessableEntity:
		return api.SchemaValidationError{Message: re.Message, Cause: re}
	default:
		return re
	}
}

// GetSchemas returns all registered schema subjects. It performs only a single
// HTTP request (GET /subjects) so it completes quickly even for large registries.
// Call GetSchemaDetails to lazily load version/ID/type for a subset of subjects.
func (kp KafkaDataSourceKaf) GetSchemas() ([]api.Schema, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return nil, err
	}
	if rc == nil {
		return []api.Schema{}, nil // no registry configured — empty listing
	}

	var subjects []string
	if err := rc.doGet("/subjects", &subjects); err != nil {
		return nil, fmt.Errorf("listing schema registry subjects: %w", err)
	}

	schemas := make([]api.Schema, len(subjects))
	for i, s := range subjects {
		schemas[i] = api.Schema{Subject: s}
	}
	return schemas, nil
}

// GetSchemaDetails fetches the latest version metadata (version, id, schemaType)
// plus the effective compatibility level for the given subjects using a
// 20-worker concurrent pool (SR-6).
func (kp KafkaDataSourceKaf) GetSchemaDetails(subjects []string) ([]api.Schema, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return nil, err
	}
	if rc == nil || len(subjects) == 0 {
		return []api.Schema{}, nil
	}

	// Fetch the global level once for the batch as the per-subject fallback.
	global, _ := kp.getGlobalCompatibility(rc)

	type result struct {
		idx    int
		schema api.Schema
	}

	const workers = 20
	jobs := make(chan int, len(subjects))
	results := make(chan result, len(subjects))

	for w := 0; w < workers; w++ {
		go func() {
			for idx := range jobs {
				subject := subjects[idx]
				var meta struct {
					Subject    string `json:"subject"`
					Version    int    `json:"version"`
					ID         int    `json:"id"`
					SchemaType string `json:"schemaType"`
				}
				if err := rc.doGet("/subjects/"+subject+"/versions/latest", &meta); err != nil {
					shared.Log.Warn("failed to fetch latest version for subject", "subject", subject, "err", err)
					results <- result{idx: idx, schema: api.Schema{Subject: subject}}
					continue
				}
				schemaType := meta.SchemaType
				if schemaType == "" {
					schemaType = "AVRO"
				}
				// Per-subject compatibility, falling back to the global level.
				compat := string(global)
				if level, specific, cerr := kp.getSubjectCompatibility(rc, subject, global); cerr == nil && specific {
					compat = string(level)
				}
				results <- result{idx: idx, schema: api.Schema{
					Subject:       meta.Subject,
					Version:       meta.Version,
					ID:            meta.ID,
					SchemaType:    schemaType,
					Compatibility: compat,
				}}
			}
		}()
	}

	for idx := range subjects {
		jobs <- idx
	}
	close(jobs)

	ordered := make([]api.Schema, len(subjects))
	for range subjects {
		r := <-results
		ordered[r.idx] = r.schema
	}
	return ordered, nil
}

// GetSchemaContent fetches the full schema definition string for the given subject
// and version. Pass version=0 (or any non-positive value) to fetch the latest version.
func (kp KafkaDataSourceKaf) GetSchemaContent(subject string, version int) (string, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return "", err
	}
	if rc == nil {
		return "", api.SchemaRegistryNotConfiguredError{}
	}

	versionStr := "latest"
	if version > 0 {
		versionStr = fmt.Sprintf("%d", version)
	}

	var response struct {
		Subject    string `json:"subject"`
		Version    int    `json:"version"`
		ID         int    `json:"id"`
		SchemaType string `json:"schemaType"`
		Schema     string `json:"schema"`
	}
	path := fmt.Sprintf("/subjects/%s/versions/%s", subject, versionStr)
	if err := rc.doGet(path, &response); err != nil {
		return "", mapRegistryError(err, subject, version)
	}
	return response.Schema, nil
}

// GetSchemaVersions lists all versions of a subject with per-version metadata
// (id, type), leaving the Schema text empty (SR-4). Versions are returned in
// ascending order.
func (kp KafkaDataSourceKaf) GetSchemaVersions(subject string) ([]api.SchemaVersion, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return nil, err
	}
	if rc == nil {
		return nil, api.SchemaRegistryNotConfiguredError{}
	}

	var versionNums []int
	if err := rc.doGet("/subjects/"+subject+"/versions", &versionNums); err != nil {
		return nil, mapRegistryError(err, subject, 0)
	}

	type result struct {
		idx     int
		version api.SchemaVersion
		err     error
	}
	const workers = 20
	jobs := make(chan int, len(versionNums))
	results := make(chan result, len(versionNums))

	for w := 0; w < workers; w++ {
		go func() {
			for idx := range jobs {
				v := versionNums[idx]
				var meta struct {
					Version    int    `json:"version"`
					ID         int    `json:"id"`
					SchemaType string `json:"schemaType"`
				}
				if err := rc.doGet(fmt.Sprintf("/subjects/%s/versions/%d", subject, v), &meta); err != nil {
					results <- result{idx: idx, err: mapRegistryError(err, subject, v)}
					continue
				}
				st := meta.SchemaType
				if st == "" {
					st = "AVRO"
				}
				results <- result{idx: idx, version: api.SchemaVersion{
					Version:    meta.Version,
					ID:         meta.ID,
					SchemaType: st,
				}}
			}
		}()
	}
	for idx := range versionNums {
		jobs <- idx
	}
	close(jobs)

	ordered := make([]api.SchemaVersion, len(versionNums))
	var firstErr error
	for range versionNums {
		r := <-results
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		ordered[r.idx] = r.version
	}
	if firstErr != nil {
		return nil, firstErr
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Version < ordered[j].Version })
	return ordered, nil
}

// GetGlobalCompatibility returns the registry's global compatibility level (SR-5).
func (kp KafkaDataSourceKaf) GetGlobalCompatibility() (api.CompatibilityLevel, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return "", err
	}
	if rc == nil {
		return "", api.SchemaRegistryNotConfiguredError{}
	}
	return kp.getGlobalCompatibility(rc)
}

func (kp KafkaDataSourceKaf) getGlobalCompatibility(rc *registryClient) (api.CompatibilityLevel, error) {
	var cfg struct {
		CompatibilityLevel string `json:"compatibilityLevel"`
	}
	if err := rc.doGet("/config", &cfg); err != nil {
		return "", mapRegistryError(err, "", 0)
	}
	return api.CompatibilityLevel(cfg.CompatibilityLevel), nil
}

// GetSubjectCompatibility returns a subject's effective compatibility level,
// falling back to the global level (isSubjectSpecific=false) when the subject has
// no own setting (SR-5).
func (kp KafkaDataSourceKaf) GetSubjectCompatibility(subject string) (api.CompatibilityLevel, bool, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return "", false, err
	}
	if rc == nil {
		return "", false, api.SchemaRegistryNotConfiguredError{}
	}
	global, gerr := kp.getGlobalCompatibility(rc)
	if gerr != nil {
		return "", false, gerr
	}
	return kp.getSubjectCompatibility(rc, subject, global)
}

func (kp KafkaDataSourceKaf) getSubjectCompatibility(rc *registryClient, subject string, global api.CompatibilityLevel) (api.CompatibilityLevel, bool, error) {
	var cfg struct {
		CompatibilityLevel string `json:"compatibilityLevel"`
	}
	err := rc.doGet("/config/"+subject, &cfg)
	if err != nil {
		var re *registryError
		if errors.As(err, &re) && (re.StatusCode == http.StatusNotFound || re.ErrorCode == 40408) {
			// No subject-specific setting — fall back to the global level.
			return global, false, nil
		}
		return "", false, mapRegistryError(err, subject, 0)
	}
	return api.CompatibilityLevel(cfg.CompatibilityLevel), true, nil
}

// RegisterSchema registers a new schema (new subject or new version) and returns
// the stored record re-fetched from versions/latest (SR-7).
func (kp KafkaDataSourceKaf) RegisterSchema(subject, schemaText, schemaType string) (api.Schema, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return api.Schema{}, err
	}
	if rc == nil {
		return api.Schema{}, api.SchemaRegistryNotConfiguredError{}
	}

	body := map[string]interface{}{"schema": schemaText}
	if t := strings.ToUpper(strings.TrimSpace(schemaType)); t != "" && t != "AVRO" {
		body["schemaType"] = t
	}

	var reg struct {
		ID int `json:"id"`
	}
	if err := rc.doPost("/subjects/"+subject+"/versions", body, &reg); err != nil {
		return api.Schema{}, mapRegistryError(err, subject, 0)
	}

	// Re-fetch the stored record for the version number.
	var meta struct {
		Subject    string `json:"subject"`
		Version    int    `json:"version"`
		ID         int    `json:"id"`
		SchemaType string `json:"schemaType"`
	}
	if err := rc.doGet("/subjects/"+subject+"/versions/latest", &meta); err != nil {
		// Registration succeeded; return what we know.
		return api.Schema{Subject: subject, ID: reg.ID, SchemaType: schemaType}, nil
	}
	st := meta.SchemaType
	if st == "" {
		st = "AVRO"
	}
	return api.Schema{Subject: meta.Subject, Version: meta.Version, ID: meta.ID, SchemaType: st}, nil
}

// CheckSchemaCompatibility tests a candidate schema against the subject's latest
// version without registering it (SR-8).
func (kp KafkaDataSourceKaf) CheckSchemaCompatibility(subject, schemaText, schemaType string) (bool, []string, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return false, nil, err
	}
	if rc == nil {
		return false, nil, api.SchemaRegistryNotConfiguredError{}
	}

	body := map[string]interface{}{"schema": schemaText}
	if t := strings.ToUpper(strings.TrimSpace(schemaType)); t != "" && t != "AVRO" {
		body["schemaType"] = t
	}

	var resp struct {
		IsCompatible bool     `json:"is_compatible"`
		Messages     []string `json:"messages"`
	}
	if err := rc.doPost("/compatibility/subjects/"+subject+"/versions/latest?verbose=true", body, &resp); err != nil {
		return false, nil, mapRegistryError(err, subject, 0)
	}
	return resp.IsCompatible, resp.Messages, nil
}

// DeleteSubject deletes all versions of a subject, returning the deleted version
// numbers. permanent=true performs a hard delete (requires a prior soft delete;
// the registry's 40405 error is surfaced) (SR-9).
func (kp KafkaDataSourceKaf) DeleteSubject(subject string, permanent bool) ([]int, error) {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return nil, err
	}
	if rc == nil {
		return nil, api.SchemaRegistryNotConfiguredError{}
	}

	path := "/subjects/" + subject
	if permanent {
		path += "?permanent=true"
	}
	var deleted []int
	if err := rc.doDelete(path, &deleted); err != nil {
		return nil, mapRegistryError(err, subject, 0)
	}
	return deleted, nil
}

// DeleteSchemaVersion deletes a single version of a subject. version=-1 targets
// the registry keyword "latest". permanent=true performs a hard delete (SR-9).
func (kp KafkaDataSourceKaf) DeleteSchemaVersion(subject string, version int, permanent bool) error {
	rc, err := kp.newRegistryClient()
	if err != nil {
		return err
	}
	if rc == nil {
		return api.SchemaRegistryNotConfiguredError{}
	}

	versionStr := fmt.Sprintf("%d", version)
	if version < 0 {
		versionStr = "latest"
	}
	path := fmt.Sprintf("/subjects/%s/versions/%s", subject, versionStr)
	if permanent {
		path += "?permanent=true"
	}
	if err := rc.doDelete(path, nil); err != nil {
		return mapRegistryError(err, subject, version)
	}
	return nil
}

// SetGlobalCompatibility sets the registry's global compatibility level (SR-10).
func (kp KafkaDataSourceKaf) SetGlobalCompatibility(level api.CompatibilityLevel) error {
	if !level.Valid() {
		return fmt.Errorf("invalid compatibility level: %q", level)
	}
	rc, err := kp.newRegistryClient()
	if err != nil {
		return err
	}
	if rc == nil {
		return api.SchemaRegistryNotConfiguredError{}
	}
	if err := rc.doPut("/config", map[string]string{"compatibility": string(level)}, nil); err != nil {
		return mapRegistryError(err, "", 0)
	}
	return nil
}

// SetSubjectCompatibility sets a subject's compatibility level (SR-10).
func (kp KafkaDataSourceKaf) SetSubjectCompatibility(subject string, level api.CompatibilityLevel) error {
	if !level.Valid() {
		return fmt.Errorf("invalid compatibility level: %q", level)
	}
	rc, err := kp.newRegistryClient()
	if err != nil {
		return err
	}
	if rc == nil {
		return api.SchemaRegistryNotConfiguredError{}
	}
	if err := rc.doPut("/config/"+subject, map[string]string{"compatibility": string(level)}, nil); err != nil {
		return mapRegistryError(err, subject, 0)
	}
	return nil
}

// fetchSchemaInfo fetches subject, version and Avro record name for a schema ID
// by querying the configured schema registry directly.
func (kp KafkaDataSourceKaf) fetchSchemaInfo(schemaIDStr string) (*api.SchemaInfo, error) {
	var schemaID int
	if _, err := fmt.Sscanf(schemaIDStr, "%d", &schemaID); err != nil {
		return nil, fmt.Errorf("invalid schema ID format: %s", schemaIDStr)
	}

	rc, err := kp.newRegistryClient()
	if err != nil || rc == nil {
		// No registry configured — return minimal stub so caller can still show the ID.
		return &api.SchemaInfo{ID: schemaID}, nil
	}

	// Fetch the schema definition JSON.
	var schemaResp struct {
		Schema     string `json:"schema"`
		SchemaType string `json:"schemaType"`
	}
	if err := rc.doGet(fmt.Sprintf("/schemas/ids/%d", schemaID), &schemaResp); err != nil {
		shared.Log.Warn("fetchSchemaInfo: failed to fetch schema by ID", "id", schemaID, "err", err)
		return &api.SchemaInfo{ID: schemaID}, nil
	}

	recordName := extractRecordName(schemaResp.Schema)

	// Try to resolve the subject name for this schema ID.
	// /schemas/ids/{id}/versions returns [{subject, version}, ...] (Confluent SR API).
	subject := ""
	version := 0
	var versions []struct {
		Subject string `json:"subject"`
		Version int    `json:"version"`
	}
	if err := rc.doGet(fmt.Sprintf("/schemas/ids/%d/versions", schemaID), &versions); err == nil && len(versions) > 0 {
		subject = versions[0].Subject
		version = versions[0].Version
	}

	return &api.SchemaInfo{
		ID:         schemaID,
		Schema:     schemaResp.Schema,
		Subject:    subject,
		Version:    version,
		RecordName: recordName,
	}, nil
}

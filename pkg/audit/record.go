// Package audit records kafui operations (state-changing by default, optionally
// all) to a local JSONL file for accountability and review. It is the audit half
// of the enforcement seam: the guarded datasource emits one Record per audited
// operation with its classified result.
package audit

import (
	"errors"
	"os/user"
	"time"

	"github.com/Benny93/kafui/pkg/api"
)

// Result classifies the outcome of an audited operation.
type Result string

const (
	ResultSuccess         Result = "success"
	ResultAccessDenied    Result = "access_denied"
	ResultValidationError Result = "validation_error"
	ResultExecutionError  Result = "execution_error"
	ResultUnknownError    Result = "unknown_error"
)

// Resource is one resource an operation touched.
type Resource struct {
	Type    string   `json:"type"`
	ID      string   `json:"id,omitempty"`
	Alter   bool     `json:"alter"`
	Actions []string `json:"actions"`
}

// Record is one audit-log line.
type Record struct {
	Timestamp string         `json:"timestamp"` // ISO-8601 UTC
	User      string         `json:"user"`
	Cluster   string         `json:"cluster,omitempty"`
	Resources []Resource     `json:"resources"`
	Operation string         `json:"operation"`
	Params    map[string]any `json:"params,omitempty"`
	Result    Result         `json:"result"`
	Error     string         `json:"error,omitempty"`
}

// isAltering reports whether any of the record's resources is an altering op.
func (r Record) isAltering() bool {
	for _, res := range r.Resources {
		if res.Alter {
			return true
		}
	}
	return false
}

// Classify maps an operation error to a Result by unwrapping the typed domain
// errors in pkg/api. A nil error is success.
func Classify(err error) Result {
	if err == nil {
		return ResultSuccess
	}
	var denied api.AccessDeniedError
	var readonly api.ClusterReadOnlyError
	if errors.As(err, &denied) || errors.As(err, &readonly) {
		return ResultAccessDenied
	}
	if isValidation(err) {
		return ResultValidationError
	}
	return ResultExecutionError
}

func isValidation(err error) bool {
	var (
		aclV   api.ACLValidationError
		quotaV api.QuotaValidationError
		topicV api.TopicValidationError
		schemaV api.SchemaValidationError
		cfgV   api.InvalidConfigError
		offV   api.InvalidOffsetResetError
		seekV  api.InvalidSeekError
		rfV    api.InvalidReplicationFactorError
	)
	return errors.As(err, &aclV) || errors.As(err, &quotaV) || errors.As(err, &topicV) ||
		errors.As(err, &schemaV) || errors.As(err, &cfgV) || errors.As(err, &offV) ||
		errors.As(err, &seekV) || errors.As(err, &rfV)
}

// ResolveUser returns the acting local identity: the OS user, falling back to
// "Unknown". (A SASL-username fallback is a future refinement.)
func ResolveUser() string {
	if u, err := user.Current(); err == nil {
		if u.Username != "" {
			return u.Username
		}
	}
	return "Unknown"
}

// nowISO returns the current time as an ISO-8601 UTC string.
func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

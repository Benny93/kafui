// Package aclcsv provides pure CSV serialization/parsing of ACL bindings and a
// declarative sync (diff + apply) between a desired binding set and the cluster.
package aclcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
)

// Header is the fixed CSV column order emitted by Marshal and accepted
// (optionally) by Parse.
var Header = []string{"Principal", "ResourceType", "PatternType", "ResourceName", "Operation", "PermissionType", "Host"}

// CSVParseError describes a malformed CSV line. Column is 1-based and 0 when the
// error applies to the whole line.
type CSVParseError struct {
	Line   int
	Column int
	Reason string
}

func (e CSVParseError) Error() string {
	if e.Column > 0 {
		return fmt.Sprintf("csv line %d, column %d: %s", e.Line, e.Column, e.Reason)
	}
	return fmt.Sprintf("csv line %d: %s", e.Line, e.Reason)
}

// canonical value sets, keyed by their normalized (lower-cased) form. The value
// is the canonical spelling written back out, so parse->marshal is stable.
var (
	resourceTypes = canon("Topic", "Group", "Cluster", "TransactionalID", "DelegationToken")
	patternTypes  = canon("Literal", "Prefixed")
	operations    = canon("Read", "Write", "Create", "Delete", "Alter", "Describe", "ClusterAction", "DescribeConfigs", "AlterConfigs", "IdempotentWrite", "All")
	permissions   = canon("Allow", "Deny")
)

func canon(values ...string) map[string]string {
	m := make(map[string]string, len(values))
	for _, v := range values {
		m[strings.ToLower(v)] = v
	}
	return m
}

// Marshal serializes bindings as CSV with the fixed header. Fields containing
// commas (e.g. SSL DN principals) are quoted by the encoding/csv writer.
func Marshal(entries []api.ACLEntry) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write(Header)
	for _, e := range entries {
		pattern := e.PatternType
		if pattern == "" {
			pattern = "Literal"
		}
		host := e.Host
		if host == "" {
			host = "*"
		}
		_ = w.Write([]string{e.Principal, e.ResourceType, pattern, e.ResourceName, e.Operation, e.Permission, host})
	}
	w.Flush()
	return b.String()
}

// Parse reads CSV text into ACL bindings. It skips blank lines and an optional
// header (case-insensitive, spaces ignored), and rejects lines without exactly
// 7 columns, blank values, or unrecognized resource type / pattern type /
// operation / permission — reporting the offending line (and column).
func Parse(s string) ([]api.ACLEntry, error) {
	r := csv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = -1 // validated manually so the error carries the line
	r.Comment = 0

	var entries []api.ACLEntry
	sawHeader := false
	first := true
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if pe, ok := err.(*csv.ParseError); ok {
				return nil, CSVParseError{Line: pe.Line, Reason: pe.Err.Error()}
			}
			return nil, err
		}
		line, _ := r.FieldPos(0)

		// Optional header on the first data record.
		if first && !sawHeader && isHeader(rec) {
			sawHeader = true
			first = false
			continue
		}
		first = false

		if len(rec) != 7 {
			return nil, CSVParseError{Line: line, Reason: fmt.Sprintf("expected 7 columns, got %d", len(rec))}
		}
		for i, v := range rec {
			if strings.TrimSpace(v) == "" {
				return nil, CSVParseError{Line: line, Column: i + 1, Reason: fmt.Sprintf("%s must not be blank", Header[i])}
			}
		}

		resourceType, err := lookup(resourceTypes, rec[1])
		if err != nil {
			return nil, CSVParseError{Line: line, Column: 2, Reason: fmt.Sprintf("unrecognized resource type %q", rec[1])}
		}
		patternType, err := lookup(patternTypes, rec[2])
		if err != nil {
			return nil, CSVParseError{Line: line, Column: 3, Reason: fmt.Sprintf("unrecognized pattern type %q", rec[2])}
		}
		operation, err := lookup(operations, rec[4])
		if err != nil {
			return nil, CSVParseError{Line: line, Column: 5, Reason: fmt.Sprintf("unrecognized operation %q", rec[4])}
		}
		permission, err := lookup(permissions, rec[5])
		if err != nil {
			return nil, CSVParseError{Line: line, Column: 6, Reason: fmt.Sprintf("unrecognized permission type %q", rec[5])}
		}

		entries = append(entries, api.ACLEntry{
			Principal:    rec[0],
			ResourceType: resourceType,
			PatternType:  patternType,
			ResourceName: rec[3],
			Operation:    operation,
			Permission:   permission,
			Host:         rec[6],
		})
	}
	return entries, nil
}

func lookup(set map[string]string, v string) (string, error) {
	if canonical, ok := set[strings.ToLower(strings.TrimSpace(v))]; ok {
		return canonical, nil
	}
	return "", fmt.Errorf("unrecognized value %q", v)
}

// isHeader reports whether rec is the fixed header row, comparing case- and
// space-insensitively.
func isHeader(rec []string) bool {
	if len(rec) != len(Header) {
		return false
	}
	for i, h := range Header {
		if normalizeHeaderCell(rec[i]) != normalizeHeaderCell(h) {
			return false
		}
	}
	return true
}

func normalizeHeaderCell(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", ""))
}

// SyncPlan is the set difference between a desired binding set and the cluster's
// current bindings, computed on full 7-field binding identity.
type SyncPlan struct {
	ToCreate []api.ACLEntry
	ToDelete []api.ACLEntry
}

// Empty reports whether the plan is a no-op (cluster already in sync).
func (p SyncPlan) Empty() bool {
	return len(p.ToCreate) == 0 && len(p.ToDelete) == 0
}

// SyncACLs fetches the current ACLs and computes the plan to reconcile them
// toward desired. It does not mutate the cluster — call SyncPlan.Apply for that.
func SyncACLs(ds api.KafkaDataSource, desired []api.ACLEntry) (SyncPlan, error) {
	current, err := ds.GetACLs()
	if err != nil {
		return SyncPlan{}, err
	}

	currentKeys := make(map[string]struct{}, len(current))
	for _, e := range current {
		currentKeys[bindingKey(e)] = struct{}{}
	}
	desiredKeys := make(map[string]struct{}, len(desired))
	for _, e := range desired {
		desiredKeys[bindingKey(e)] = struct{}{}
	}

	var plan SyncPlan
	for _, e := range desired {
		if _, ok := currentKeys[bindingKey(e)]; !ok {
			plan.ToCreate = append(plan.ToCreate, e)
		}
	}
	for _, e := range current {
		if _, ok := desiredKeys[bindingKey(e)]; !ok {
			plan.ToDelete = append(plan.ToDelete, e)
		}
	}
	return plan, nil
}

// Apply creates the missing bindings and deletes the extra ones, logging the
// plan. It is a no-op for an empty plan.
func (p SyncPlan) Apply(ds api.KafkaDataSource) error {
	if p.Empty() {
		shared.Log.Info("ACL sync: already in sync, nothing to do")
		return nil
	}
	shared.Log.Info("ACL sync: applying plan", "create", len(p.ToCreate), "delete", len(p.ToDelete))
	for _, e := range p.ToCreate {
		if err := ds.CreateACL(e); err != nil {
			return fmt.Errorf("create ACL %s: %w", bindingKey(e), err)
		}
	}
	for _, e := range p.ToDelete {
		if err := ds.DeleteACL(e); err != nil {
			return fmt.Errorf("delete ACL %s: %w", bindingKey(e), err)
		}
	}
	return nil
}

// bindingKey is the full-identity key of an ACL binding, normalizing the two
// fields that carry implicit defaults (host "*", pattern "Literal").
func bindingKey(e api.ACLEntry) string {
	pattern := e.PatternType
	if pattern == "" {
		pattern = "Literal"
	}
	host := e.Host
	if host == "" {
		host = "*"
	}
	return strings.Join([]string{e.Principal, e.ResourceType, pattern, e.ResourceName, e.Operation, e.Permission, host}, "\x00")
}

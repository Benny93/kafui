package api

// ACL-set expansion helpers (AQ-7). These are pure functions that translate a
// high-level intent ("a consumer for topic X and group Y") into the concrete
// list of ACLEntry bindings needed to satisfy it. The UI (and any datasource)
// creates the expanded entries via CreateACL; no new interface method is needed.

// clusterResourceName is the conventional resource name for the whole cluster.
const clusterResourceName = "kafka-cluster"

func defaultHost(host string) string {
	if host == "" {
		return "*"
	}
	return host
}

// allowEntries builds one allow ACLEntry per operation on a single resource.
func allowEntries(principal, host, resourceType, resourceName, patternType string, ops ...string) []ACLEntry {
	entries := make([]ACLEntry, 0, len(ops))
	for _, op := range ops {
		entries = append(entries, ACLEntry{
			Principal:    principal,
			Host:         host,
			ResourceType: resourceType,
			ResourceName: resourceName,
			PatternType:  patternType,
			Operation:    op,
			Permission:   "Allow",
		})
	}
	return entries
}

// expandResource emits allow bindings for a resource kind that may be specified
// either as an explicit list (Literal) or a single prefix (Prefixed), but not
// both. It errors when both list and prefix are supplied for the same kind.
func expandResource(principal, host, resourceType string, names []string, prefix string, ops []string) ([]ACLEntry, error) {
	if len(names) > 0 && prefix != "" {
		return nil, ACLValidationError{
			Field:  resourceType,
			Reason: "specify either an explicit list or a prefix, not both",
		}
	}
	var entries []ACLEntry
	for _, name := range names {
		entries = append(entries, allowEntries(principal, host, resourceType, name, "Literal", ops...)...)
	}
	if prefix != "" {
		entries = append(entries, allowEntries(principal, host, resourceType, prefix, "Prefixed", ops...)...)
	}
	return entries, nil
}

// ExpandConsumerACLs expands a consumer intent: READ+DESCRIBE on the given
// topics and groups. Lists become Literal bindings; the prefixes become
// Prefixed bindings. It rejects supplying both a list and a prefix for the same
// resource kind.
func ExpandConsumerACLs(principal, host string, topics, groups []string, topicPrefix, groupPrefix string) ([]ACLEntry, error) {
	if err := ValidatePrincipal(principal); err != nil {
		return nil, err
	}
	host = defaultHost(host)
	ops := []string{"Read", "Describe"}

	var entries []ACLEntry
	topicEntries, err := expandResource(principal, host, "Topic", topics, topicPrefix, ops)
	if err != nil {
		return nil, err
	}
	groupEntries, err := expandResource(principal, host, "Group", groups, groupPrefix, ops)
	if err != nil {
		return nil, err
	}
	entries = append(entries, topicEntries...)
	entries = append(entries, groupEntries...)
	return entries, nil
}

// ExpandProducerACLs expands a producer intent: WRITE+DESCRIBE+CREATE on the
// topics, WRITE+DESCRIBE on the transactional id (exact or prefix), and, when
// idempotent, IDEMPOTENT_WRITE on the cluster.
func ExpandProducerACLs(principal, host string, topics []string, topicPrefix, txID, txIDPrefix string, idempotent bool) ([]ACLEntry, error) {
	if err := ValidatePrincipal(principal); err != nil {
		return nil, err
	}
	host = defaultHost(host)

	var entries []ACLEntry
	topicEntries, err := expandResource(principal, host, "Topic", topics, topicPrefix, []string{"Write", "Describe", "Create"})
	if err != nil {
		return nil, err
	}
	entries = append(entries, topicEntries...)

	txEntries, err := expandResource(principal, host, "TransactionalID", nonEmptyList(txID), txIDPrefix, []string{"Write", "Describe"})
	if err != nil {
		return nil, err
	}
	entries = append(entries, txEntries...)

	if idempotent {
		entries = append(entries, allowEntries(principal, host, "Cluster", clusterResourceName, "Literal", "IdempotentWrite")...)
	}
	return entries, nil
}

// ExpandStreamAppACLs expands a Kafka Streams application intent: READ on input
// topics (Literal), WRITE on output topics (Literal), and ALL on Prefixed
// appID for both Topic and Group resource types (internal topics/changelogs).
func ExpandStreamAppACLs(principal, host, appID string, inputTopics, outputTopics []string) ([]ACLEntry, error) {
	if err := ValidatePrincipal(principal); err != nil {
		return nil, err
	}
	if appID == "" {
		return nil, ACLValidationError{Field: "appID", Reason: "must not be empty"}
	}
	host = defaultHost(host)

	var entries []ACLEntry
	for _, t := range inputTopics {
		entries = append(entries, allowEntries(principal, host, "Topic", t, "Literal", "Read")...)
	}
	for _, t := range outputTopics {
		entries = append(entries, allowEntries(principal, host, "Topic", t, "Literal", "Write")...)
	}
	entries = append(entries, allowEntries(principal, host, "Topic", appID, "Prefixed", "All")...)
	entries = append(entries, allowEntries(principal, host, "Group", appID, "Prefixed", "All")...)
	return entries, nil
}

// nonEmptyList returns a single-element slice for a non-empty string, else nil.
// It lets expandResource treat an exact transactional id like a one-item list.
func nonEmptyList(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

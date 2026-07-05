package kafds

import (
	"fmt"
	"sort"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/IBM/sarama"
)

// parseResourceType maps a resource-type string to a sarama enum, rejecting
// unknown values with an ACLValidationError.
func parseResourceType(s string) (sarama.AclResourceType, error) {
	var t sarama.AclResourceType
	if err := t.UnmarshalText([]byte(s)); err != nil {
		return 0, api.ACLValidationError{Field: "resourceType", Reason: fmt.Sprintf("unrecognized value %q", s), Cause: err}
	}
	return t, nil
}

func parsePatternType(s string) (sarama.AclResourcePatternType, error) {
	var t sarama.AclResourcePatternType
	if err := t.UnmarshalText([]byte(s)); err != nil {
		return 0, api.ACLValidationError{Field: "patternType", Reason: fmt.Sprintf("unrecognized value %q", s), Cause: err}
	}
	return t, nil
}

func parseOperation(s string) (sarama.AclOperation, error) {
	var t sarama.AclOperation
	if err := t.UnmarshalText([]byte(s)); err != nil {
		return 0, api.ACLValidationError{Field: "operation", Reason: fmt.Sprintf("unrecognized value %q", s), Cause: err}
	}
	return t, nil
}

func parsePermission(s string) (sarama.AclPermissionType, error) {
	var t sarama.AclPermissionType
	if err := t.UnmarshalText([]byte(s)); err != nil {
		return 0, api.ACLValidationError{Field: "permission", Reason: fmt.Sprintf("unrecognized value %q", s), Cause: err}
	}
	return t, nil
}

// GetACLsFiltered implements api.KafkaDataSource. Empty filter fields match any
// value. Results are stably sorted by principal -> resourceType -> resourceName.
func (kp KafkaDataSourceKaf) GetACLsFiltered(filter api.ACLFilter) ([]api.ACLEntry, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}

	saramaFilter := sarama.AclFilter{
		Version:                   1, // v1 carries the pattern-type filter
		ResourceType:              sarama.AclResourceAny,
		ResourcePatternTypeFilter: sarama.AclPatternAny,
		Operation:                 sarama.AclOperationAny,
		PermissionType:            sarama.AclPermissionAny,
	}
	if filter.ResourceType != "" {
		rt, err := parseResourceType(filter.ResourceType)
		if err != nil {
			return nil, err
		}
		saramaFilter.ResourceType = rt
	}
	if filter.PatternType != "" {
		pt, err := parsePatternType(filter.PatternType)
		if err != nil {
			return nil, err
		}
		saramaFilter.ResourcePatternTypeFilter = pt
	}
	if filter.ResourceName != "" {
		name := filter.ResourceName
		saramaFilter.ResourceName = &name
	}

	resourceAcls, err := admin.ListAcls(saramaFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list ACLs: %w", err)
	}

	shared.Log.Info("GetACLsFiltered: raw entries", "resources", len(resourceAcls))

	var entries []api.ACLEntry
	for _, ra := range resourceAcls {
		for _, acl := range ra.Acls {
			entries = append(entries, api.ACLEntry{
				Principal:    acl.Principal,
				Host:         acl.Host,
				ResourceType: ra.Resource.ResourceType.String(),
				ResourceName: ra.Resource.ResourceName,
				PatternType:  ra.Resource.ResourcePatternType.String(),
				Operation:    acl.Operation.String(),
				Permission:   acl.PermissionType.String(),
			})
		}
	}

	sortACLs(entries)
	return entries, nil
}

func sortACLs(entries []api.ACLEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Principal != entries[j].Principal {
			return entries[i].Principal < entries[j].Principal
		}
		if entries[i].ResourceType != entries[j].ResourceType {
			return entries[i].ResourceType < entries[j].ResourceType
		}
		return entries[i].ResourceName < entries[j].ResourceName
	})
}

// CreateACL implements api.KafkaDataSource. It validates and maps the binding to
// sarama enums, then calls admin.CreateACLs.
func (kp KafkaDataSourceKaf) CreateACL(entry api.ACLEntry) error {
	if entry.PatternType == "" {
		entry.PatternType = "Literal"
	}
	if entry.Host == "" {
		entry.Host = "*"
	}
	if err := api.ValidateACLEntry(entry); err != nil {
		return err
	}

	resource, acl, err := toSaramaBinding(entry)
	if err != nil {
		return err
	}

	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	resourceACLs := &sarama.ResourceAcls{Resource: resource, Acls: []*sarama.Acl{&acl}}
	if err := admin.CreateACLs([]*sarama.ResourceAcls{resourceACLs}); err != nil {
		return fmt.Errorf("failed to create ACL: %w", err)
	}
	return nil
}

// DeleteACL implements api.KafkaDataSource. It builds an exact-match filter from
// the full binding and returns an ACLNotFoundError when nothing matched.
func (kp KafkaDataSourceKaf) DeleteACL(entry api.ACLEntry) error {
	if entry.PatternType == "" {
		entry.PatternType = "Literal"
	}
	if entry.Host == "" {
		entry.Host = "*"
	}
	if err := api.ValidateACLEntry(entry); err != nil {
		return err
	}

	resource, acl, err := toSaramaBinding(entry)
	if err != nil {
		return err
	}

	filter := sarama.AclFilter{
		Version:                   1,
		ResourceType:              resource.ResourceType,
		ResourceName:              &resource.ResourceName,
		ResourcePatternTypeFilter: resource.ResourcePatternType,
		Principal:                 &acl.Principal,
		Host:                      &acl.Host,
		Operation:                 acl.Operation,
		PermissionType:            acl.PermissionType,
	}

	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	matching, err := admin.DeleteACL(filter, false)
	if err != nil {
		return fmt.Errorf("failed to delete ACL: %w", err)
	}
	if len(matching) == 0 {
		return api.ACLNotFoundError{Entry: entry}
	}
	return nil
}

// toSaramaBinding maps a (defaulted, validated) ACLEntry to sarama's Resource +
// Acl, rejecting unrecognized enum strings with an ACLValidationError.
func toSaramaBinding(entry api.ACLEntry) (sarama.Resource, sarama.Acl, error) {
	rt, err := parseResourceType(entry.ResourceType)
	if err != nil {
		return sarama.Resource{}, sarama.Acl{}, err
	}
	pt, err := parsePatternType(entry.PatternType)
	if err != nil {
		return sarama.Resource{}, sarama.Acl{}, err
	}
	op, err := parseOperation(entry.Operation)
	if err != nil {
		return sarama.Resource{}, sarama.Acl{}, err
	}
	perm, err := parsePermission(entry.Permission)
	if err != nil {
		return sarama.Resource{}, sarama.Acl{}, err
	}
	resource := sarama.Resource{
		ResourceType:        rt,
		ResourceName:        entry.ResourceName,
		ResourcePatternType: pt,
	}
	acl := sarama.Acl{
		Principal:      entry.Principal,
		Host:           entry.Host,
		Operation:      op,
		PermissionType: perm,
	}
	return resource, acl, nil
}

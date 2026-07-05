package mock

import (
	"sort"

	"github.com/Benny93/kafui/pkg/api"
)

// --- ACLs (AQ-1 / AQ-6) ---

// seedACLs installs the initial sample ACL set on first access.
func (kp *KafkaDataSourceMock) seedACLs() {
	if kp.aclsInit {
		return
	}
	kp.acls = []api.ACLEntry{
		{Principal: "User:CN=service-account", Host: "*", ResourceType: "Topic", ResourceName: "*", PatternType: "Literal", Operation: "Read", Permission: "Allow"},
		{Principal: "User:CN=service-account", Host: "*", ResourceType: "Group", ResourceName: "*", PatternType: "Literal", Operation: "Read", Permission: "Allow"},
		{Principal: "User:CN=admin", Host: "*", ResourceType: "Cluster", ResourceName: "kafka-cluster", PatternType: "Literal", Operation: "All", Permission: "Allow"},
		{Principal: "User:CN=readonly", Host: "*", ResourceType: "Topic", ResourceName: "orders-", PatternType: "Prefixed", Operation: "Describe", Permission: "Allow"},
	}
	kp.aclsInit = true
}

func normalizeACL(e api.ACLEntry) api.ACLEntry {
	if e.PatternType == "" {
		e.PatternType = "Literal"
	}
	if e.Host == "" {
		e.Host = "*"
	}
	return e
}

func aclEquals(a, b api.ACLEntry) bool {
	return normalizeACL(a) == normalizeACL(b)
}

func matchesACLFilter(e api.ACLEntry, f api.ACLFilter) bool {
	if f.ResourceType != "" && !equalFold(e.ResourceType, f.ResourceType) {
		return false
	}
	if f.ResourceName != "" && e.ResourceName != f.ResourceName {
		return false
	}
	if f.PatternType != "" && !equalFold(e.PatternType, f.PatternType) {
		return false
	}
	return true
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return toLower(a) == toLower(b)
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func sortMockACLs(entries []api.ACLEntry) {
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

// GetACLs implements api.KafkaDataSource (the match-any case).
func (kp *KafkaDataSourceMock) GetACLs() ([]api.ACLEntry, error) {
	return kp.GetACLsFiltered(api.ACLFilter{})
}

// GetACLsFiltered implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) GetACLsFiltered(filter api.ACLFilter) ([]api.ACLEntry, error) {
	kp.aclMu.Lock()
	defer kp.aclMu.Unlock()
	kp.seedACLs()

	var out []api.ACLEntry
	for _, e := range kp.acls {
		if matchesACLFilter(e, filter) {
			out = append(out, e)
		}
	}
	sortMockACLs(out)
	return out, nil
}

// CreateACL implements api.KafkaDataSource. It validates the entry and appends
// it (defaulting host and pattern type).
func (kp *KafkaDataSourceMock) CreateACL(entry api.ACLEntry) error {
	entry = normalizeACL(entry)
	if err := api.ValidateACLEntry(entry); err != nil {
		return err
	}
	kp.aclMu.Lock()
	defer kp.aclMu.Unlock()
	kp.seedACLs()
	kp.acls = append(kp.acls, entry)
	return nil
}

// DeleteACL implements api.KafkaDataSource. It removes the binding matching the
// full definition or returns an ACLNotFoundError.
func (kp *KafkaDataSourceMock) DeleteACL(entry api.ACLEntry) error {
	kp.aclMu.Lock()
	defer kp.aclMu.Unlock()
	kp.seedACLs()

	for i, e := range kp.acls {
		if aclEquals(e, entry) {
			kp.acls = append(kp.acls[:i], kp.acls[i+1:]...)
			return nil
		}
	}
	return api.ACLNotFoundError{Entry: normalizeACL(entry)}
}

// --- Client quotas (AQ-12) ---

// seedQuotas installs the initial sample quota set on first access.
func (kp *KafkaDataSourceMock) seedQuotas() {
	if kp.quotasInit {
		return
	}
	kp.quotas = []api.ClientQuotaEntry{
		{Entity: api.ClientQuotaEntity{User: strPtr("alice")}, Quotas: map[string]float64{"consumer_byte_rate": 2097152, "producer_byte_rate": 1048576}},
		{Entity: api.ClientQuotaEntity{User: strPtr("bob"), ClientID: strPtr("payment-service")}, Quotas: map[string]float64{"producer_byte_rate": 5242880, "request_percentage": 75}},
		{Entity: api.ClientQuotaEntity{User: strPtr("")}, Quotas: map[string]float64{"consumer_byte_rate": 1048576}}, // <default> user
		{Entity: api.ClientQuotaEntity{IP: strPtr("10.0.0.1")}, Quotas: map[string]float64{"connection_creation_rate": 100}},
	}
	kp.quotasInit = true
}

func quotaEntityEquals(a, b api.ClientQuotaEntity) bool {
	return ptrEquals(a.User, b.User) && ptrEquals(a.ClientID, b.ClientID) && ptrEquals(a.IP, b.IP)
}

func ptrEquals(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func sortMockQuotas(entries []api.ClientQuotaEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		for _, get := range []func(api.ClientQuotaEntity) *string{
			func(e api.ClientQuotaEntity) *string { return e.User },
			func(e api.ClientQuotaEntity) *string { return e.ClientID },
			func(e api.ClientQuotaEntity) *string { return e.IP },
		} {
			a, b := get(entries[i].Entity), get(entries[j].Entity)
			if a == nil && b == nil {
				continue
			}
			if a == nil {
				return false
			}
			if b == nil {
				return true
			}
			if *a != *b {
				return *a < *b
			}
		}
		return false
	})
}

// GetClientQuotas implements api.KafkaDataSource.
func (kp *KafkaDataSourceMock) GetClientQuotas() ([]api.ClientQuotaEntry, error) {
	kp.quotaMu.Lock()
	defer kp.quotaMu.Unlock()
	kp.seedQuotas()

	out := make([]api.ClientQuotaEntry, len(kp.quotas))
	for i, e := range kp.quotas {
		copied := make(map[string]float64, len(e.Quotas))
		for k, v := range e.Quotas {
			copied[k] = v
		}
		out[i] = api.ClientQuotaEntry{Entity: e.Entity, Quotas: copied}
	}
	sortMockQuotas(out)
	return out, nil
}

// AlterClientQuotas implements api.KafkaDataSource with replace/delete
// semantics: the submitted map fully replaces the entity's properties; an empty
// or nil map deletes the entity.
func (kp *KafkaDataSourceMock) AlterClientQuotas(entity api.ClientQuotaEntity, quotas map[string]float64) error {
	if err := api.ValidateQuotaEntity(entity); err != nil {
		return err
	}
	kp.quotaMu.Lock()
	defer kp.quotaMu.Unlock()
	kp.seedQuotas()

	// Locate any existing entry for this entity.
	idx := -1
	for i, e := range kp.quotas {
		if quotaEntityEquals(e.Entity, entity) {
			idx = i
			break
		}
	}

	// Empty submission deletes the entity.
	if len(quotas) == 0 {
		if idx >= 0 {
			kp.quotas = append(kp.quotas[:idx], kp.quotas[idx+1:]...)
		}
		return nil
	}

	copied := make(map[string]float64, len(quotas))
	for k, v := range quotas {
		copied[k] = v
	}
	if idx >= 0 {
		kp.quotas[idx].Quotas = copied // replace semantics
	} else {
		kp.quotas = append(kp.quotas, api.ClientQuotaEntry{Entity: entity, Quotas: copied})
	}
	return nil
}

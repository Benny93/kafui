package mock

import (
	"sort"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
)

// mockRegistry is the in-memory schema-registry state backing the mock
// datasource. It supports version browsing, registration, deletion, compatibility
// levels and pre-registration compatibility checks so those flows are testable
// without a broker.
type mockRegistry struct {
	subjects map[string]*mockSubject
	nextID   int
	global   api.CompatibilityLevel
}

type mockSubject struct {
	versions []api.SchemaVersion // ascending by Version; Schema text populated
	// compatibility is the subject-specific level, or "" to fall back to global.
	compatibility api.CompatibilityLevel
}

// registry lazily seeds and returns the in-memory registry. Callers must hold no
// lock; this method manages kp.schemaRegMu itself for seeding but returns the
// pointer for direct use — every exported method below re-locks around mutation.
func (kp *KafkaDataSourceMock) registry() *mockRegistry {
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	if kp.schemaReg == nil {
		kp.schemaReg = seedMockRegistry()
	}
	return kp.schemaReg
}

func seedMockRegistry() *mockRegistry {
	r := &mockRegistry{
		subjects: map[string]*mockSubject{},
		nextID:   200,
		global:   api.CompatibilityBackward,
	}

	// orders-value: three plausibly evolved AVRO versions (v3 references a named
	// OrderItem record — the "with references" subject).
	r.subjects["orders-value"] = &mockSubject{
		compatibility: "", // falls back to global
		versions: []api.SchemaVersion{
			{Version: 1, ID: 101, SchemaType: "AVRO", Schema: `{"type":"record","name":"OrderCreatedEvent","namespace":"com.example.orders","fields":[{"name":"orderId","type":"string"},{"name":"customerId","type":"string"},{"name":"amount","type":"double"}]}`},
			{Version: 2, ID: 102, SchemaType: "AVRO", Schema: `{"type":"record","name":"OrderCreatedEvent","namespace":"com.example.orders","fields":[{"name":"orderId","type":"string"},{"name":"customerId","type":"string"},{"name":"amount","type":"double"},{"name":"items","type":{"type":"array","items":"string"}}]}`},
			{Version: 3, ID: 103, SchemaType: "AVRO", Schema: `{"type":"record","name":"OrderCreatedEvent","namespace":"com.example.orders","fields":[{"name":"orderId","type":"string"},{"name":"customerId","type":"string"},{"name":"amount","type":"double"},{"name":"items","type":{"type":"array","items":{"type":"record","name":"OrderItem","fields":[{"name":"productId","type":"string"},{"name":"quantity","type":"int"}]}}},{"name":"createdAt","type":"long"}]}`},
		},
	}
	r.subjects["orders-key"] = &mockSubject{
		versions: []api.SchemaVersion{
			{Version: 1, ID: 98, SchemaType: "AVRO", Schema: `{"type":"record","name":"OrderKey","namespace":"com.example.orders","fields":[{"name":"orderId","type":"string"}]}`},
		},
	}
	// payments-value has its own (subject-specific) compatibility level.
	r.subjects["payments-value"] = &mockSubject{
		compatibility: api.CompatibilityFull,
		versions: []api.SchemaVersion{
			{Version: 1, ID: 109, SchemaType: "AVRO", Schema: `{"type":"record","name":"PaymentProcessedEvent","namespace":"com.example.payments","fields":[{"name":"paymentId","type":"string"},{"name":"orderId","type":"string"},{"name":"amount","type":"double"}]}`},
			{Version: 2, ID: 110, SchemaType: "AVRO", Schema: `{"type":"record","name":"PaymentProcessedEvent","namespace":"com.example.payments","fields":[{"name":"paymentId","type":"string"},{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"status","type":{"type":"enum","name":"PaymentStatus","symbols":["SUCCESS","FAILED","PENDING"]}}]}`},
		},
	}
	r.subjects["user-events-value"] = &mockSubject{
		versions: []api.SchemaVersion{
			{Version: 1, ID: 116, SchemaType: "PROTOBUF", Schema: "syntax = \"proto3\";\nmessage UserEvent {\n  string user_id = 1;\n}\n"},
			{Version: 2, ID: 120, SchemaType: "PROTOBUF", Schema: "syntax = \"proto3\";\nmessage UserEvent {\n  string user_id = 1;\n  string email = 2;\n}\n"},
		},
	}
	r.subjects["inventory-value"] = &mockSubject{
		versions: []api.SchemaVersion{
			{Version: 1, ID: 130, SchemaType: "JSON", Schema: `{"type":"object","properties":{"warehouseId":{"type":"string"},"productId":{"type":"string"}}}`},
		},
	}
	return r
}

func (s *mockSubject) latest() api.SchemaVersion {
	return s.versions[len(s.versions)-1]
}

func (r *mockRegistry) effectiveCompat(s *mockSubject) (api.CompatibilityLevel, bool) {
	if s.compatibility != "" {
		return s.compatibility, true
	}
	return r.global, false
}

// GetSchemas returns all subject names currently registered.
func (kp *KafkaDataSourceMock) GetSchemas() ([]api.Schema, error) {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	names := make([]string, 0, len(r.subjects))
	for name := range r.subjects {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]api.Schema, len(names))
	for i, n := range names {
		out[i] = api.Schema{Subject: n}
	}
	return out, nil
}

// GetSchemaDetails returns latest-version metadata plus effective compatibility.
func (kp *KafkaDataSourceMock) GetSchemaDetails(subjects []string) ([]api.Schema, error) {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	out := make([]api.Schema, 0, len(subjects))
	for _, name := range subjects {
		s, ok := r.subjects[name]
		if !ok || len(s.versions) == 0 {
			out = append(out, api.Schema{Subject: name, SchemaType: "AVRO", Compatibility: string(r.global)})
			continue
		}
		latest := s.latest()
		compat, _ := r.effectiveCompat(s)
		out = append(out, api.Schema{
			Subject:       name,
			Version:       latest.Version,
			ID:            latest.ID,
			SchemaType:    latest.SchemaType,
			Compatibility: string(compat),
		})
	}
	return out, nil
}

// GetSchemaContent returns a version's schema text (latest when version <= 0).
func (kp *KafkaDataSourceMock) GetSchemaContent(subject string, version int) (string, error) {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	s, ok := r.subjects[subject]
	if !ok {
		return "", api.SubjectNotFoundError{Subject: subject}
	}
	if version <= 0 {
		return s.latest().Schema, nil
	}
	for _, v := range s.versions {
		if v.Version == version {
			return v.Schema, nil
		}
	}
	return "", api.SchemaVersionNotFoundError{Subject: subject, Version: version}
}

// GetSchemaVersions lists all versions of a subject (ascending).
func (kp *KafkaDataSourceMock) GetSchemaVersions(subject string) ([]api.SchemaVersion, error) {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	s, ok := r.subjects[subject]
	if !ok {
		return nil, api.SubjectNotFoundError{Subject: subject}
	}
	out := make([]api.SchemaVersion, len(s.versions))
	copy(out, s.versions)
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// GetGlobalCompatibility returns the mock global compatibility level.
func (kp *KafkaDataSourceMock) GetGlobalCompatibility() (api.CompatibilityLevel, error) {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	return r.global, nil
}

// GetSubjectCompatibility returns a subject's effective level with a fallback flag.
func (kp *KafkaDataSourceMock) GetSubjectCompatibility(subject string) (api.CompatibilityLevel, bool, error) {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	s, ok := r.subjects[subject]
	if !ok {
		return "", false, api.SubjectNotFoundError{Subject: subject}
	}
	level, specific := r.effectiveCompat(s)
	return level, specific, nil
}

// RegisterSchema appends a new version (creating the subject when new).
func (kp *KafkaDataSourceMock) RegisterSchema(subject, schemaText, schemaType string) (api.Schema, error) {
	if strings.Contains(schemaText, "INVALID") {
		return api.Schema{}, api.SchemaValidationError{Message: "schema contains INVALID marker"}
	}
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()

	st := strings.ToUpper(strings.TrimSpace(schemaType))
	if st == "" {
		st = "AVRO"
	}

	s, ok := r.subjects[subject]
	if !ok {
		s = &mockSubject{}
		r.subjects[subject] = s
	} else if strings.Contains(schemaText, "INCOMPATIBLE") {
		// Existing subject + magic marker → incompatible with prior versions.
		return api.Schema{}, api.SchemaIncompatibleError{Subject: subject, Message: "reader schema incompatible with writer schema"}
	}

	nextVersion := 1
	if len(s.versions) > 0 {
		nextVersion = s.latest().Version + 1
	}
	r.nextID++
	v := api.SchemaVersion{Version: nextVersion, ID: r.nextID, SchemaType: st, Schema: schemaText}
	s.versions = append(s.versions, v)
	return api.Schema{Subject: subject, Version: v.Version, ID: v.ID, SchemaType: st}, nil
}

// CheckSchemaCompatibility returns incompatible when the candidate contains the
// magic "INCOMPATIBLE" marker, compatible otherwise.
func (kp *KafkaDataSourceMock) CheckSchemaCompatibility(subject, schemaText, schemaType string) (bool, []string, error) {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	if _, ok := r.subjects[subject]; !ok {
		return false, nil, api.SubjectNotFoundError{Subject: subject}
	}
	if strings.Contains(schemaText, "INCOMPATIBLE") {
		return false, []string{"reader field 'x' is missing a default value", "incompatible with version 1"}, nil
	}
	return true, nil, nil
}

// DeleteSubject removes all versions of a subject, returning the deleted numbers.
func (kp *KafkaDataSourceMock) DeleteSubject(subject string, permanent bool) ([]int, error) {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	s, ok := r.subjects[subject]
	if !ok {
		return nil, api.SubjectNotFoundError{Subject: subject}
	}
	deleted := make([]int, len(s.versions))
	for i, v := range s.versions {
		deleted[i] = v.Version
	}
	delete(r.subjects, subject)
	return deleted, nil
}

// DeleteSchemaVersion removes a single version (version=-1 targets the latest).
func (kp *KafkaDataSourceMock) DeleteSchemaVersion(subject string, version int, permanent bool) error {
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	s, ok := r.subjects[subject]
	if !ok {
		return api.SubjectNotFoundError{Subject: subject}
	}
	target := version
	if version < 0 {
		target = s.latest().Version
	}
	for i, v := range s.versions {
		if v.Version == target {
			s.versions = append(s.versions[:i], s.versions[i+1:]...)
			if len(s.versions) == 0 {
				delete(r.subjects, subject)
			}
			return nil
		}
	}
	return api.SchemaVersionNotFoundError{Subject: subject, Version: target}
}

// SetGlobalCompatibility validates and updates the global level.
func (kp *KafkaDataSourceMock) SetGlobalCompatibility(level api.CompatibilityLevel) error {
	if !level.Valid() {
		return api.SchemaValidationError{Message: "invalid compatibility level: " + string(level)}
	}
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	r.global = level
	return nil
}

// SetSubjectCompatibility validates and updates a subject's level.
func (kp *KafkaDataSourceMock) SetSubjectCompatibility(subject string, level api.CompatibilityLevel) error {
	if !level.Valid() {
		return api.SchemaValidationError{Message: "invalid compatibility level: " + string(level)}
	}
	r := kp.registry()
	kp.schemaRegMu.Lock()
	defer kp.schemaRegMu.Unlock()
	s, ok := r.subjects[subject]
	if !ok {
		return api.SubjectNotFoundError{Subject: subject}
	}
	s.compatibility = level
	return nil
}

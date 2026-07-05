// Package datasource holds cross-backend datasource decorators. The Guard is the
// authorization + audit enforcement seam: it wraps any api.KafkaDataSource and
// interposes on state-changing operations.
//
// CONTRACT for future features: the Guard EMBEDS api.KafkaDataSource, so read and
// analytical methods pass through automatically. Every NEW mutating method added
// to api.KafkaDataSource MUST be overridden here with a gate check + audit record
// (use the do/doValue helpers and declare its authz resource+action). Denial
// happens BEFORE any effect. See CLAUDE.md ("Authorization/Audit seam").
package datasource

import (
	"context"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/audit"
	"github.com/Benny93/kafui/pkg/authz"
)

// Guard wraps a KafkaDataSource, enforcing the authorization Gate and emitting
// audit records for state-changing operations. It satisfies api.KafkaDataSource
// by embedding the wrapped implementation.
type Guard struct {
	api.KafkaDataSource
	gate  *authz.Gate
	audit *audit.Service
}

var _ api.KafkaDataSource = (*Guard)(nil)

// NewGuard wraps inner with gate + audit enforcement. It seeds the gate with the
// inner datasource's current cluster so the first checks resolve the right
// profile. A nil gate/audit degrades to allow-all / no-op respectively.
func NewGuard(inner api.KafkaDataSource, gate *authz.Gate, aud *audit.Service) *Guard {
	g := &Guard{KafkaDataSource: inner, gate: gate, audit: aud}
	if gate != nil {
		gate.SetCluster(inner.GetContext())
	}
	return g
}

// ref is one (resource, name, action) triple an operation touches.
type ref struct {
	rt     authz.ResourceType
	name   string
	action authz.Action
}

// SetContext delegates then re-resolves the gate's active cluster profile.
func (g *Guard) SetContext(name string) error {
	err := g.KafkaDataSource.SetContext(name)
	if err == nil && g.gate != nil {
		g.gate.SetCluster(name)
	}
	return err
}

// do runs an error-only operation: gate-check every ref first (deny before any
// effect), delegate, then emit one audit record with the classified result.
func (g *Guard) do(op string, params map[string]any, refs []ref, fn func() error) error {
	if err := g.check(refs); err != nil {
		g.record(op, params, refs, err)
		return err
	}
	err := fn()
	g.record(op, params, refs, err)
	return err
}

func (g *Guard) check(refs []ref) error {
	if g.gate == nil {
		return nil
	}
	for _, r := range refs {
		if err := g.gate.Check(r.action, r.rt, r.name); err != nil {
			return err
		}
	}
	return nil
}

func (g *Guard) record(op string, params map[string]any, refs []ref, err error) {
	if !g.audit.Enabled() {
		return
	}
	resources := make([]audit.Resource, 0, len(refs))
	for _, r := range refs {
		resources = append(resources, audit.Resource{
			Type:    string(r.rt),
			ID:      r.name,
			Alter:   authz.IsAltering(r.rt, r.action),
			Actions: []string{string(r.action)},
		})
	}
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	g.audit.Record(audit.Record{
		Cluster:   g.KafkaDataSource.GetContext(),
		Resources: resources,
		Operation: op,
		Params:    params,
		Result:    audit.Classify(err),
		Error:     errMsg,
	})
}

// --- Topic administration ---

func (g *Guard) CreateTopic(name string, numPartitions int32, replicationFactor int16, configs map[string]*string) error {
	return g.do("CreateTopic", map[string]any{"topic": name}, []ref{{authz.ResourceTopic, "", authz.ActionCreate}}, func() error {
		return g.KafkaDataSource.CreateTopic(name, numPartitions, replicationFactor, configs)
	})
}

func (g *Guard) DeleteTopic(name string) error {
	return g.do("DeleteTopic", map[string]any{"topic": name}, []ref{{authz.ResourceTopic, name, authz.ActionDelete}}, func() error {
		return g.KafkaDataSource.DeleteTopic(name)
	})
}

func (g *Guard) UpdateTopicConfig(name string, entries map[string]*string) error {
	return g.do("UpdateTopicConfig", map[string]any{"topic": name}, []ref{{authz.ResourceTopic, name, authz.ActionEdit}}, func() error {
		return g.KafkaDataSource.UpdateTopicConfig(name, entries)
	})
}

func (g *Guard) IncreasePartitions(name string, totalCount int32) error {
	return g.do("IncreasePartitions", map[string]any{"topic": name, "totalCount": totalCount}, []ref{{authz.ResourceTopic, name, authz.ActionEdit}}, func() error {
		return g.KafkaDataSource.IncreasePartitions(name, totalCount)
	})
}

func (g *Guard) ChangeReplicationFactor(name string, newFactor int16) error {
	return g.do("ChangeReplicationFactor", map[string]any{"topic": name, "factor": newFactor}, []ref{{authz.ResourceTopic, name, authz.ActionEdit}}, func() error {
		return g.KafkaDataSource.ChangeReplicationFactor(name, newFactor)
	})
}

func (g *Guard) PurgeTopicMessages(name string, partition int32) error {
	return g.do("PurgeTopicMessages", map[string]any{"topic": name, "partition": partition}, []ref{{authz.ResourceTopic, name, authz.ActionDeleteMessages}}, func() error {
		return g.KafkaDataSource.PurgeTopicMessages(name, partition)
	})
}

func (g *Guard) RecreateTopic(name string) error {
	// Recreate deletes then re-creates: requires both delete and create.
	refs := []ref{{authz.ResourceTopic, name, authz.ActionDelete}, {authz.ResourceTopic, "", authz.ActionCreate}}
	return g.do("RecreateTopic", map[string]any{"topic": name}, refs, func() error {
		return g.KafkaDataSource.RecreateTopic(name)
	})
}

func (g *Guard) ProduceMessage(ctx context.Context, topic string, rec api.ProduceRecord) error {
	return g.do("ProduceMessage", map[string]any{"topic": topic}, []ref{{authz.ResourceTopic, topic, authz.ActionProduceMessages}}, func() error {
		return g.KafkaDataSource.ProduceMessage(ctx, topic, rec)
	})
}

// --- Consumer groups ---

func (g *Guard) DeleteConsumerGroup(groupID string) error {
	return g.do("DeleteConsumerGroup", map[string]any{"group": groupID}, []ref{{authz.ResourceConsumerGroup, groupID, authz.ActionDelete}}, func() error {
		return g.KafkaDataSource.DeleteConsumerGroup(groupID)
	})
}

func (g *Guard) DeleteConsumerGroupOffsets(groupID string, topic string) error {
	return g.do("DeleteConsumerGroupOffsets", map[string]any{"group": groupID, "topic": topic}, []ref{{authz.ResourceConsumerGroup, groupID, authz.ActionResetOffsets}}, func() error {
		return g.KafkaDataSource.DeleteConsumerGroupOffsets(groupID, topic)
	})
}

func (g *Guard) ResetConsumerGroupOffsets(ctx context.Context, req api.OffsetResetRequest) error {
	return g.do("ResetConsumerGroupOffsets", map[string]any{"group": req.GroupID, "topic": req.Topic}, []ref{{authz.ResourceConsumerGroup, req.GroupID, authz.ActionResetOffsets}}, func() error {
		return g.KafkaDataSource.ResetConsumerGroupOffsets(ctx, req)
	})
}

// --- Schema registry ---

func (g *Guard) RegisterSchema(subject, schemaText, schemaType string) (api.Schema, error) {
	var out api.Schema
	err := g.do("RegisterSchema", map[string]any{"subject": subject}, []ref{{authz.ResourceSchema, "", authz.ActionCreate}}, func() error {
		var e error
		out, e = g.KafkaDataSource.RegisterSchema(subject, schemaText, schemaType)
		return e
	})
	return out, err
}

func (g *Guard) DeleteSubject(subject string, permanent bool) ([]int, error) {
	var out []int
	err := g.do("DeleteSubject", map[string]any{"subject": subject, "permanent": permanent}, []ref{{authz.ResourceSchema, subject, authz.ActionDelete}}, func() error {
		var e error
		out, e = g.KafkaDataSource.DeleteSubject(subject, permanent)
		return e
	})
	return out, err
}

func (g *Guard) DeleteSchemaVersion(subject string, version int, permanent bool) error {
	return g.do("DeleteSchemaVersion", map[string]any{"subject": subject, "version": version}, []ref{{authz.ResourceSchema, subject, authz.ActionDelete}}, func() error {
		return g.KafkaDataSource.DeleteSchemaVersion(subject, version, permanent)
	})
}

func (g *Guard) SetGlobalCompatibility(level api.CompatibilityLevel) error {
	return g.do("SetGlobalCompatibility", map[string]any{"level": string(level)}, []ref{{authz.ResourceSchema, "", authz.ActionModifyCompat}}, func() error {
		return g.KafkaDataSource.SetGlobalCompatibility(level)
	})
}

func (g *Guard) SetSubjectCompatibility(subject string, level api.CompatibilityLevel) error {
	return g.do("SetSubjectCompatibility", map[string]any{"subject": subject, "level": string(level)}, []ref{{authz.ResourceSchema, subject, authz.ActionModifyCompat}}, func() error {
		return g.KafkaDataSource.SetSubjectCompatibility(subject, level)
	})
}

// --- ACLs & quotas ---

func (g *Guard) CreateACL(entry api.ACLEntry) error {
	return g.do("CreateACL", map[string]any{"resource": entry.ResourceName, "operation": entry.Operation}, []ref{{authz.ResourceACL, "", authz.ActionCreate}}, func() error {
		return g.KafkaDataSource.CreateACL(entry)
	})
}

func (g *Guard) DeleteACL(entry api.ACLEntry) error {
	return g.do("DeleteACL", map[string]any{"resource": entry.ResourceName, "operation": entry.Operation}, []ref{{authz.ResourceACL, entry.ResourceName, authz.ActionDelete}}, func() error {
		return g.KafkaDataSource.DeleteACL(entry)
	})
}

func (g *Guard) AlterClientQuotas(entity api.ClientQuotaEntity, quotas map[string]float64) error {
	return g.do("AlterClientQuotas", nil, []ref{{authz.ResourceClientQuota, "", authz.ActionEdit}}, func() error {
		return g.KafkaDataSource.AlterClientQuotas(entity, quotas)
	})
}

// --- Broker / cluster configuration ---

func (g *Guard) AlterBrokerConfig(brokerID int32, key, value string) error {
	return g.do("AlterBrokerConfig", map[string]any{"broker": brokerID, "key": key}, []ref{{authz.ResourceClusterConfig, "", authz.ActionEdit}}, func() error {
		return g.KafkaDataSource.AlterBrokerConfig(brokerID, key, value)
	})
}

func (g *Guard) AlterReplicaLogDir(brokerID int32, topic string, partition int32, logDir string) error {
	return g.do("AlterReplicaLogDir", map[string]any{"broker": brokerID, "topic": topic, "partition": partition}, []ref{{authz.ResourceClusterConfig, "", authz.ActionEdit}}, func() error {
		return g.KafkaDataSource.AlterReplicaLogDir(brokerID, topic, partition, logDir)
	})
}

// --- Kafka Connect (connector name is the composite "<connect>/<name>") ---

func connectorName(connect, name string) string { return connect + "/" + name }

func (g *Guard) CreateConnector(connect, name string, config map[string]string) (api.Connector, error) {
	var out api.Connector
	err := g.do("CreateConnector", map[string]any{"connect": connect, "connector": name}, []ref{{authz.ResourceConnector, "", authz.ActionCreate}}, func() error {
		var e error
		out, e = g.KafkaDataSource.CreateConnector(connect, name, config)
		return e
	})
	return out, err
}

func (g *Guard) UpdateConnectorConfig(connect, name string, config map[string]string) (api.Connector, error) {
	var out api.Connector
	err := g.do("UpdateConnectorConfig", map[string]any{"connect": connect, "connector": name}, []ref{{authz.ResourceConnector, connectorName(connect, name), authz.ActionEdit}}, func() error {
		var e error
		out, e = g.KafkaDataSource.UpdateConnectorConfig(connect, name, config)
		return e
	})
	return out, err
}

func (g *Guard) DeleteConnector(connect, name string) error {
	return g.do("DeleteConnector", map[string]any{"connect": connect, "connector": name}, []ref{{authz.ResourceConnector, connectorName(connect, name), authz.ActionDelete}}, func() error {
		return g.KafkaDataSource.DeleteConnector(connect, name)
	})
}

func (g *Guard) PauseConnector(connect, name string) error {
	return g.do("PauseConnector", map[string]any{"connect": connect, "connector": name}, []ref{{authz.ResourceConnector, connectorName(connect, name), authz.ActionPause}}, func() error {
		return g.KafkaDataSource.PauseConnector(connect, name)
	})
}

func (g *Guard) ResumeConnector(connect, name string) error {
	return g.do("ResumeConnector", map[string]any{"connect": connect, "connector": name}, []ref{{authz.ResourceConnector, connectorName(connect, name), authz.ActionResume}}, func() error {
		return g.KafkaDataSource.ResumeConnector(connect, name)
	})
}

func (g *Guard) StopConnector(connect, name string) error {
	return g.do("StopConnector", map[string]any{"connect": connect, "connector": name}, []ref{{authz.ResourceConnector, connectorName(connect, name), authz.ActionPause}}, func() error {
		return g.KafkaDataSource.StopConnector(connect, name)
	})
}

func (g *Guard) RestartConnector(connect, name string) error {
	return g.do("RestartConnector", map[string]any{"connect": connect, "connector": name}, []ref{{authz.ResourceConnector, connectorName(connect, name), authz.ActionRestart}}, func() error {
		return g.KafkaDataSource.RestartConnector(connect, name)
	})
}

func (g *Guard) RestartConnectorTask(connect, name string, taskID int) error {
	return g.do("RestartConnectorTask", map[string]any{"connect": connect, "connector": name, "task": taskID}, []ref{{authz.ResourceConnector, connectorName(connect, name), authz.ActionRestart}}, func() error {
		return g.KafkaDataSource.RestartConnectorTask(connect, name, taskID)
	})
}

func (g *Guard) ResetConnectorOffsets(connect, name string) error {
	return g.do("ResetConnectorOffsets", map[string]any{"connect": connect, "connector": name}, []ref{{authz.ResourceConnector, connectorName(connect, name), authz.ActionResetOffsets}}, func() error {
		return g.KafkaDataSource.ResetConnectorOffsets(connect, name)
	})
}

// --- ksqlDB ---

func (g *Guard) ExecuteKsql(ctx context.Context, sql string, props map[string]string) (<-chan api.KsqlResultTable, error) {
	refs := []ref{{authz.ResourceSQLEngine, "", authz.ActionExecute}}
	if err := g.check(refs); err != nil {
		g.record("ExecuteKsql", map[string]any{"sql": sql}, refs, err)
		return nil, err
	}
	ch, err := g.KafkaDataSource.ExecuteKsql(ctx, sql, props)
	g.record("ExecuteKsql", map[string]any{"sql": sql}, refs, err)
	return ch, err
}

// --- Listing filters (AA-9): drop entries the active profile can't view. ---

func (g *Guard) filterNames(rt authz.ResourceType, names []string) []string {
	if g.gate == nil || !g.gate.Enabled() {
		return names
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if g.gate.Allowed(authz.ActionView, rt, n) {
			out = append(out, n)
		}
	}
	return out
}

func (g *Guard) GetTopicNames() ([]string, error) {
	names, err := g.KafkaDataSource.GetTopicNames()
	if err != nil {
		return names, err
	}
	return g.filterNames(authz.ResourceTopic, names), nil
}

func (g *Guard) GetTopics() (map[string]api.Topic, error) {
	topics, err := g.KafkaDataSource.GetTopics()
	if err != nil || g.gate == nil || !g.gate.Enabled() {
		return topics, err
	}
	for name := range topics {
		if !g.gate.Allowed(authz.ActionView, authz.ResourceTopic, name) {
			delete(topics, name)
		}
	}
	return topics, nil
}

func (g *Guard) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	groups, err := g.KafkaDataSource.GetConsumerGroups()
	if err != nil || g.gate == nil || !g.gate.Enabled() {
		return groups, err
	}
	out := groups[:0]
	for _, grp := range groups {
		if g.gate.Allowed(authz.ActionView, authz.ResourceConsumerGroup, grp.Name) {
			out = append(out, grp)
		}
	}
	return out, nil
}

func (g *Guard) GetSchemas() ([]api.Schema, error) {
	return g.filterSchemas(g.KafkaDataSource.GetSchemas())
}

func (g *Guard) GetSchemaDetails(subjects []string) ([]api.Schema, error) {
	return g.filterSchemas(g.KafkaDataSource.GetSchemaDetails(subjects))
}

func (g *Guard) filterSchemas(schemas []api.Schema, err error) ([]api.Schema, error) {
	if err != nil || g.gate == nil || !g.gate.Enabled() {
		return schemas, err
	}
	out := schemas[:0]
	for _, s := range schemas {
		if g.gate.Allowed(authz.ActionView, authz.ResourceSchema, s.Subject) {
			out = append(out, s)
		}
	}
	return out, nil
}

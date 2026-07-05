package mock

import (
	"sort"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
)

// mockConnectState is the in-memory Kafka Connect state backing the mock
// datasource. It supports two Connect clusters (one reachable with mixed-state
// connectors, one unreachable for degraded-listing exercises), a plugin list,
// and validation, with all write operations mutating this state so the UI flows
// are exercisable via `make run-mock`.
type mockConnectState struct {
	clusters map[string]*mockConnectCluster
	order    []string
	plugins  []api.ConnectorPlugin
}

type mockConnectCluster struct {
	name            string
	address         string
	version         string
	commit          string
	kafkaID         string
	reachable       bool
	consumerPattern string
	connectors      map[string]*mockConnector
}

type mockConnector struct {
	name     string
	class    string
	typ      api.ConnectorType
	config   map[string]string
	state    string
	workerID string
	trace    string
	topics   []string
	tasks    []api.ConnectorTask
}

// connect lazily seeds and returns the in-memory Connect state. Callers must
// hold no lock; this method manages connectMu itself for seeding but returns the
// pointer for direct use — every exported method below re-locks around mutation.
func (kp *KafkaDataSourceMock) connect() *mockConnectState {
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	if kp.connectState == nil {
		kp.connectState = seedMockConnect()
	}
	return kp.connectState
}

const longTrace = "org.apache.kafka.connect.errors.ConnectException: Tolerance exceeded in error handler\n" +
	"\tat org.apache.kafka.connect.runtime.errors.RetryWithToleranceOperator.execAndHandleError(RetryWithToleranceOperator.java:223)\n" +
	"\tat org.apache.kafka.connect.runtime.errors.RetryWithToleranceOperator.execute(RetryWithToleranceOperator.java:149)\n" +
	"\tat org.apache.kafka.connect.runtime.WorkerSinkTask.convertAndTransformRecord(WorkerSinkTask.java:513)\n" +
	"Caused by: org.apache.kafka.common.errors.SerializationException: Error deserializing Avro message for id 42\n" +
	"Caused by: java.net.ConnectException: Connection refused (schema registry unreachable)\n"

func seedMockConnect() *mockConnectState {
	primary := &mockConnectCluster{
		name:            "connect-primary",
		address:         "http://connect-primary:8083",
		version:         "3.7.0",
		commit:          "abc123def456",
		kafkaID:         "kafka-cluster-dev-01",
		reachable:       true,
		consumerPattern: "connect-<connector>",
		connectors:      map[string]*mockConnector{},
	}

	primary.connectors["orders-source"] = &mockConnector{
		name:  "orders-source",
		class: "io.debezium.connector.postgresql.PostgresConnector",
		typ:   api.ConnectorTypeSource,
		config: map[string]string{
			"connector.class":     "io.debezium.connector.postgresql.PostgresConnector",
			"tasks.max":           "2",
			"database.hostname":   "postgres",
			"database.user":       "debezium",
			"database.password":   "s3cr3t-pg-pass",
			"database.dbname":     "orders",
			"topic.prefix":        "orders",
			"schema.registry.url": "http://sr:8081",
		},
		state:    api.ConnectorStateRunning,
		workerID: "10.0.0.11:8083",
		topics:   []string{"orders.public.customers", "orders.public.orders"},
		tasks: []api.ConnectorTask{
			{ID: 0, WorkerID: "10.0.0.11:8083", State: api.ConnectorStateRunning},
			{ID: 1, WorkerID: "10.0.0.12:8083", State: api.ConnectorStateRunning},
		},
	}

	primary.connectors["orders-sink-es"] = &mockConnector{
		name:  "orders-sink-es",
		class: "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
		typ:   api.ConnectorTypeSink,
		config: map[string]string{
			"connector.class":     "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
			"tasks.max":           "3",
			"topics":              "orders.public.orders",
			"connection.url":      "http://elasticsearch:9200",
			"connection.password": "es-admin-pass",
			"key.converter":       "org.apache.kafka.connect.storage.StringConverter",
		},
		state:    api.ConnectorStateFailed,
		workerID: "10.0.0.13:8083",
		trace:    longTrace,
		topics:   []string{"orders.public.orders"},
		tasks: []api.ConnectorTask{
			{ID: 0, WorkerID: "10.0.0.13:8083", State: api.ConnectorStateRunning},
			{ID: 1, WorkerID: "10.0.0.13:8083", State: api.ConnectorStateFailed, Trace: longTrace},
			{ID: 2, WorkerID: "", State: api.ConnectorStateUnassigned},
		},
	}

	primary.connectors["metrics-sink-s3"] = &mockConnector{
		name:  "metrics-sink-s3",
		class: "io.confluent.connect.s3.S3SinkConnector",
		typ:   api.ConnectorTypeSink,
		config: map[string]string{
			"connector.class":       "io.confluent.connect.s3.S3SinkConnector",
			"tasks.max":             "1",
			"topics":                "metrics",
			"s3.bucket.name":        "kafka-metrics",
			"aws.secret.access.key": "AKIA-super-secret",
		},
		state:    api.ConnectorStatePaused,
		workerID: "10.0.0.11:8083",
		topics:   []string{"metrics"},
		tasks: []api.ConnectorTask{
			{ID: 0, WorkerID: "10.0.0.11:8083", State: api.ConnectorStatePaused},
		},
	}

	primary.connectors["audit-sink-jdbc"] = &mockConnector{
		name:  "audit-sink-jdbc",
		class: "io.confluent.connect.jdbc.JdbcSinkConnector",
		typ:   api.ConnectorTypeSink,
		config: map[string]string{
			"connector.class":     "io.confluent.connect.jdbc.JdbcSinkConnector",
			"tasks.max":           "1",
			"topics":              "audit",
			"connection.url":      "jdbc:postgresql://pg/audit",
			"connection.password": "audit-db-pass",
		},
		state:    api.ConnectorStateStopped,
		workerID: "",
		topics:   []string{"audit"},
		tasks:    []api.ConnectorTask{},
	}

	// Unreachable secondary cluster: no runtime info, no connectors surfaced.
	secondary := &mockConnectCluster{
		name:            "connect-secondary",
		address:         "http://connect-secondary:8083",
		reachable:       false,
		consumerPattern: "connect-<connector>",
		connectors:      map[string]*mockConnector{},
	}

	return &mockConnectState{
		clusters: map[string]*mockConnectCluster{
			primary.name:   primary,
			secondary.name: secondary,
		},
		order: []string{primary.name, secondary.name},
		plugins: []api.ConnectorPlugin{
			{Class: "io.debezium.connector.postgresql.PostgresConnector", Type: "source", Version: "2.5.0.Final"},
			{Class: "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector", Type: "sink", Version: "14.0.0"},
			{Class: "io.confluent.connect.s3.S3SinkConnector", Type: "sink", Version: "10.5.0"},
			{Class: "io.confluent.connect.jdbc.JdbcSinkConnector", Type: "sink", Version: "10.7.0"},
			{Class: "org.apache.kafka.connect.file.FileStreamSourceConnector", Type: "source", Version: "3.7.0"},
		},
	}
}

func (c *mockConnectCluster) consumerGroupFor(connector string) string {
	pattern := c.consumerPattern
	if pattern == "" {
		return "connect-" + connector
	}
	return strings.ReplaceAll(pattern, "<connector>", connector)
}

func (c *mockConnector) failedTaskCount() int {
	n := 0
	for _, t := range c.tasks {
		if strings.EqualFold(t.State, api.ConnectorStateFailed) {
			n++
		}
	}
	return n
}

func (kp *KafkaDataSourceMock) GetConnectClusters(withStats bool) ([]api.ConnectCluster, error) {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	out := make([]api.ConnectCluster, 0, len(s.order))
	for _, name := range s.order {
		c := s.clusters[name]
		cluster := api.ConnectCluster{
			Name:      c.name,
			Address:   c.address,
			Reachable: c.reachable,
		}
		if c.reachable {
			cluster.Version = c.version
			cluster.Commit = c.commit
			cluster.KafkaClusterID = c.kafkaID
			if withStats {
				for _, conn := range c.connectors {
					cluster.ConnectorCount++
					if strings.EqualFold(conn.state, api.ConnectorStateFailed) {
						cluster.FailedConnectorCount++
					}
					cluster.TaskCount += len(conn.tasks)
					cluster.FailedTaskCount += conn.failedTaskCount()
				}
			}
		}
		out = append(out, cluster)
	}
	return out, nil
}

func (kp *KafkaDataSourceMock) GetConnectorNames(connect string) ([]string, error) {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	c, ok := s.clusters[connect]
	if !ok {
		return nil, api.ConnectClusterNotFoundError{Connect: connect, Cluster: currentContext}
	}
	names := make([]string, 0, len(c.connectors))
	for name := range c.connectors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (kp *KafkaDataSourceMock) GetConnectors() ([]api.Connector, error) {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	var out []api.Connector
	for _, name := range s.order {
		c := s.clusters[name]
		if !c.reachable {
			continue // degraded: unreachable clusters omit their connectors
		}
		for _, conn := range c.connectors {
			out = append(out, kp.toConnector(c, conn))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ConnectCluster < out[j].ConnectCluster
	})
	return out, nil
}

func (kp *KafkaDataSourceMock) toConnector(c *mockConnectCluster, conn *mockConnector) api.Connector {
	res := api.Connector{
		ConnectCluster:  c.name,
		Name:            conn.name,
		Class:           conn.class,
		Type:            conn.typ,
		Topics:          append([]string(nil), conn.topics...),
		State:           conn.state,
		WorkerID:        conn.workerID,
		Trace:           conn.trace,
		TaskCount:       len(conn.tasks),
		FailedTaskCount: conn.failedTaskCount(),
	}
	if conn.typ == api.ConnectorTypeSink {
		res.ConsumerGroup = c.consumerGroupFor(conn.name)
	}
	return res
}

func (kp *KafkaDataSourceMock) GetConnectorDetails(connect, name string) (api.ConnectorDetails, error) {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	c, conn, err := s.lookup(connect, name)
	if err != nil {
		return api.ConnectorDetails{}, err
	}
	details := api.ConnectorDetails{
		ConnectCluster: connect,
		Name:           conn.name,
		Class:          conn.class,
		Type:           conn.typ,
		Config:         api.MaskConnectorConfig(conn.config),
		State:          conn.state,
		WorkerID:       conn.workerID,
		Trace:          conn.trace,
		Tasks:          append([]api.ConnectorTask(nil), conn.tasks...),
		Topics:         append([]string(nil), conn.topics...),
	}
	if conn.typ == api.ConnectorTypeSink {
		details.ConsumerGroup = c.consumerGroupFor(conn.name)
	}
	return details, nil
}

// lookup finds a connector under a Connect cluster, returning typed errors for
// unknown cluster / connector. Caller holds connectMu.
func (s *mockConnectState) lookup(connect, name string) (*mockConnectCluster, *mockConnector, error) {
	c, ok := s.clusters[connect]
	if !ok {
		return nil, nil, api.ConnectClusterNotFoundError{Connect: connect, Cluster: currentContext}
	}
	conn, ok := c.connectors[name]
	if !ok {
		return c, nil, api.ConnectorNotFoundError{Connector: name, Connect: connect}
	}
	return c, conn, nil
}

func (kp *KafkaDataSourceMock) CreateConnector(connect, name string, config map[string]string) (api.Connector, error) {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	c, ok := s.clusters[connect]
	if !ok {
		return api.Connector{}, api.ConnectClusterNotFoundError{Connect: connect, Cluster: currentContext}
	}
	if _, exists := c.connectors[name]; exists {
		return api.Connector{}, api.ConnectorAlreadyExistsError{Connector: name, Connect: connect}
	}
	typ := api.ConnectorTypeSource
	if _, hasTopics := config["topics"]; hasTopics {
		typ = api.ConnectorTypeSink
	}
	conn := &mockConnector{
		name:     name,
		class:    config["connector.class"],
		typ:      typ,
		config:   copyConfig(config),
		state:    api.ConnectorStateRunning,
		workerID: "10.0.0.11:8083",
		tasks:    []api.ConnectorTask{{ID: 0, WorkerID: "10.0.0.11:8083", State: api.ConnectorStateRunning}},
	}
	if t := config["topics"]; t != "" {
		conn.topics = strings.Split(t, ",")
	}
	c.connectors[name] = conn
	return kp.toConnector(c, conn), nil
}

func (kp *KafkaDataSourceMock) UpdateConnectorConfig(connect, name string, config map[string]string) (api.Connector, error) {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	c, conn, err := s.lookup(connect, name)
	if err != nil {
		return api.Connector{}, err
	}
	conn.config = copyConfig(config)
	if cls := config["connector.class"]; cls != "" {
		conn.class = cls
	}
	if t := config["topics"]; t != "" {
		conn.topics = strings.Split(t, ",")
	}
	return kp.toConnector(c, conn), nil
}

func (kp *KafkaDataSourceMock) DeleteConnector(connect, name string) error {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	if _, _, err := s.lookup(connect, name); err != nil {
		return err
	}
	delete(s.clusters[connect].connectors, name)
	return nil
}

func (kp *KafkaDataSourceMock) setState(connect, name, state string) error {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	_, conn, err := s.lookup(connect, name)
	if err != nil {
		return err
	}
	conn.state = state
	for i := range conn.tasks {
		switch state {
		case api.ConnectorStatePaused, api.ConnectorStateRunning:
			conn.tasks[i].State = state
			conn.tasks[i].Trace = ""
		}
	}
	return nil
}

func (kp *KafkaDataSourceMock) PauseConnector(connect, name string) error {
	return kp.setState(connect, name, api.ConnectorStatePaused)
}

func (kp *KafkaDataSourceMock) ResumeConnector(connect, name string) error {
	return kp.setState(connect, name, api.ConnectorStateRunning)
}

func (kp *KafkaDataSourceMock) StopConnector(connect, name string) error {
	return kp.setState(connect, name, api.ConnectorStateStopped)
}

func (kp *KafkaDataSourceMock) RestartConnector(connect, name string) error {
	return kp.setState(connect, name, api.ConnectorStateRunning)
}

func (kp *KafkaDataSourceMock) RestartConnectorTask(connect, name string, taskID int) error {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	_, conn, err := s.lookup(connect, name)
	if err != nil {
		return err
	}
	for i := range conn.tasks {
		if conn.tasks[i].ID == taskID {
			conn.tasks[i].State = api.ConnectorStateRunning
			conn.tasks[i].Trace = ""
			return nil
		}
	}
	return api.ConnectorNotFoundError{Connector: name, Connect: connect}
}

func (kp *KafkaDataSourceMock) ResetConnectorOffsets(connect, name string) error {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	_, conn, err := s.lookup(connect, name)
	if err != nil {
		return err
	}
	if !strings.EqualFold(conn.state, api.ConnectorStateStopped) {
		return api.ConnectorNotStoppedError{Connector: name, Connect: connect, State: conn.state}
	}
	return nil
}

func (kp *KafkaDataSourceMock) GetConnectorPlugins(connect string) ([]api.ConnectorPlugin, error) {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	if _, ok := s.clusters[connect]; !ok {
		return nil, api.ConnectClusterNotFoundError{Connect: connect, Cluster: currentContext}
	}
	return append([]api.ConnectorPlugin(nil), s.plugins...), nil
}

func (kp *KafkaDataSourceMock) ValidateConnectorConfig(connect, pluginClass string, config map[string]string) (api.ConnectorValidationResult, error) {
	s := kp.connect()
	kp.connectMu.Lock()
	defer kp.connectMu.Unlock()
	if _, ok := s.clusters[connect]; !ok {
		return api.ConnectorValidationResult{}, api.ConnectClusterNotFoundError{Connect: connect, Cluster: currentContext}
	}
	result := api.ConnectorValidationResult{
		Name:   pluginClass,
		Groups: []string{"Common"},
	}
	// Flag a missing required field to exercise the validation-error path.
	nameCfg := api.ConnectorConfigKeyValidation{Name: "name", Value: config["name"], Visible: true}
	if config["name"] == "" {
		nameCfg.Errors = []string{"Missing required configuration \"name\" which has no default value."}
		result.ErrorCount++
	}
	classCfg := api.ConnectorConfigKeyValidation{Name: "connector.class", Value: config["connector.class"], Visible: true}
	if config["connector.class"] == "" {
		classCfg.Errors = []string{"Missing required configuration \"connector.class\" which has no default value."}
		result.ErrorCount++
	}
	result.Configs = []api.ConnectorConfigKeyValidation{nameCfg, classCfg}
	return result, nil
}

func copyConfig(config map[string]string) map[string]string {
	out := make(map[string]string, len(config))
	for k, v := range config {
		out[k] = v
	}
	return out
}

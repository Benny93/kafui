# Metrics Collection, Graphs, and Monitoring Endpoints

## ADDED Requirements

### Requirement: Configurable per-broker metrics collection
The system SHALL support collecting metrics from each broker of a cluster via a per-cluster metrics configuration that selects a collection mechanism (JMX or Prometheus scraping), a port, and optional transport security and authentication settings. When no metrics configuration is present for a cluster, broker-level metric collection SHALL be skipped while all other functionality remains available.

#### Scenario: Collection mechanism selection
- **WHEN** a cluster's metrics configuration specifies type "JMX" together with a port
- **THEN** broker metrics are collected from each broker node using the JMX mechanism on that port

#### Scenario: Prometheus mechanism selection
- **WHEN** a cluster's metrics configuration specifies type "PROMETHEUS"
- **THEN** broker metrics are collected by scraping a Prometheus-format HTTP endpoint on each broker node

#### Scenario: No metrics configuration
- **WHEN** a cluster has no metrics configuration
- **THEN** no broker-level metric collection is attempted and per-broker metric sets are empty

### Requirement: JMX metrics collection
The system SHALL collect broker metrics over JMX by connecting to each broker's host at the configured JMX port, querying all management beans in the broker's server metrics domain, and converting every numeric attribute into a named, labeled metric value.

#### Scenario: Metric extraction and naming
- **WHEN** a management bean with multiple properties and numeric attributes is read from a broker
- **THEN** each numeric attribute becomes a metric whose name is composed of the bean domain, the first bean property value, and the attribute name (with dots and dashes replaced by underscores), and whose labels are the remaining bean properties

#### Scenario: Non-numeric attributes ignored
- **WHEN** a management bean attribute holds a non-numeric value
- **THEN** that attribute is omitted from the collected metrics

#### Scenario: JMX authentication
- **WHEN** the metrics configuration contains a username and password
- **THEN** the JMX connection is established using those credentials

#### Scenario: JMX over SSL
- **WHEN** the metrics configuration provides a client keystore (and optionally the cluster provides a truststore)
- **THEN** the JMX connection is established over SSL using those key materials

#### Scenario: SSL JMX unsupported at runtime
- **WHEN** SSL JMX is configured but the runtime cannot provide the required socket customization
- **THEN** a warning is logged and an empty metric set is returned for the affected brokers instead of failing

#### Scenario: Broker connection failure
- **WHEN** the JMX connection to one broker fails
- **THEN** the error is logged and collection continues for the remaining brokers

### Requirement: Prometheus endpoint scraping
The system SHALL collect broker metrics by issuing an HTTP GET request to a metrics path on each broker host at the configured port (defaulting to a well-known exporter port when unset), parsing the response body as Prometheus text exposition format.

#### Scenario: Scrape request construction
- **WHEN** scraping a broker with SSL disabled and no port configured
- **THEN** the request is sent over plain HTTP to the default exporter port at the standard metrics path

#### Scenario: Secure scraping
- **WHEN** the metrics configuration enables SSL or provides a keystore
- **THEN** the scrape request is sent over HTTPS using the configured truststore/keystore, and configured username/password are sent as HTTP basic authentication

#### Scenario: Exposition format parsing
- **WHEN** a scrape response contains counters, gauges, histograms, summaries, and untyped samples with help and type comment lines
- **THEN** all sample families are parsed into typed metric snapshots preserving names, labels, and values

#### Scenario: Scrape failure tolerance
- **WHEN** the scrape of one broker fails or returns an unreadable body
- **THEN** a warning is logged and an empty metric list is used for that broker without failing the overall collection

### Requirement: Periodic metrics refresh
The system SHALL refresh cluster state and metrics for every configured cluster on a fixed schedule with a configurable interval, defaulting to 30 seconds, and SHALL serve all metrics-related read endpoints from the most recently cached snapshot.

#### Scenario: Scheduled refresh
- **WHEN** the refresh interval elapses
- **THEN** metrics for all clusters are re-collected in parallel and the cached snapshots are replaced

#### Scenario: Reads served from cache
- **WHEN** a client requests cluster, broker, or exposition metrics
- **THEN** the response reflects the last completed collection cycle rather than triggering a new collection

### Requirement: Inferred metrics from cluster state
The system SHALL always derive a set of gauge metrics from the observed cluster state, independent of whether external broker metric collection is configured. These SHALL include: broker count; per-broker disk usage, usable disk and total disk bytes; topic count; per-topic partition count and disk size; per-partition next offset, oldest offset, in-sync replica count, replica count, and leader node id (-1 when leaderless); consumer group count; and per-group state code, member count, distinct host count, committed offsets per partition, and approximate lag per partition.

#### Scenario: Metrics without external collection
- **WHEN** a cluster has no JMX or Prometheus metrics configuration
- **THEN** the inferred gauge metrics are still produced from the periodically scraped cluster state

#### Scenario: Consumer lag derivation
- **WHEN** a consumer group has a committed offset for a partition whose end offset is known
- **THEN** a lag gauge equal to end offset minus committed offset is produced, labeled with group, topic, and partition

### Requirement: Cluster I/O rate aggregation
The system SHALL scan collected broker metrics for fifteen-minute-rate byte-in and byte-out topic-metrics values and aggregate them into per-broker and per-topic bytes-in/bytes-out rates, with per-topic values summed across brokers, and SHALL expose these aggregated rates in cluster and topic statistics.

#### Scenario: Broker rate extraction
- **WHEN** a broker exposes a fifteen-minute-rate bytes-in-per-second metric from its topic-metrics group without a topic label
- **THEN** that value is recorded as the broker's cluster-wide bytes-in rate (first matching value wins per broker)

#### Scenario: Topic rate summation
- **WHEN** multiple brokers expose fifteen-minute-rate bytes-in/out metrics labeled with the same topic
- **THEN** the per-topic rate is the sum of the values across brokers

#### Scenario: Rates in cluster overview
- **WHEN** a client requests cluster statistics
- **THEN** the response includes the aggregated cluster bytes-in and bytes-out per-second rates when they could be determined

### Requirement: Cluster metrics listing endpoint
The system SHALL provide an authenticated per-cluster endpoint that returns the collected metrics as a flat list, merging per-broker metric families with the same name by summing data points that share identical label sets, together with the inferred metrics.

#### Scenario: Cross-broker summarization
- **WHEN** two brokers report a summable metric family (gauge, counter, or untyped) with the same name and identical labels
- **THEN** the listing contains a single data point whose value is the sum across brokers

#### Scenario: Non-summable types
- **WHEN** a metric family type does not support merging
- **THEN** it is omitted from the summarized listing

### Requirement: Per-broker metrics endpoint
The system SHALL provide an authenticated endpoint that returns, for a given cluster and broker id, the raw metric families collected from that broker plus broker segment statistics.

#### Scenario: Broker metrics lookup
- **WHEN** a client requests metrics for a specific broker id
- **THEN** the metric families collected from that broker in the last cycle are returned; an empty result is returned when no metrics exist for that broker

### Requirement: Prometheus exposition endpoints
The system SHALL expose all collected metrics for external scraping in Prometheus text exposition format at an unversioned metrics path: one endpoint aggregating all clusters and one endpoint per named cluster. Every exposed sample SHALL carry a cluster-name label, and samples originating from a specific broker SHALL additionally carry a broker-id label. Metric families with the same name across clusters SHALL be merged into single families.

#### Scenario: Global exposition
- **WHEN** an external scraper requests the global metrics endpoint
- **THEN** the response contains inferred and per-broker metrics of all clusters that have exposition enabled, in Prometheus text format with the corresponding content-type header, each sample labeled with its cluster name

#### Scenario: Per-cluster exposition
- **WHEN** an external scraper requests the metrics endpoint for a named cluster with exposition enabled
- **THEN** only that cluster's metrics are returned in Prometheus text format

#### Scenario: Exposition opt-out
- **WHEN** a cluster's configuration disables Prometheus exposition (it is enabled by default)
- **THEN** the per-cluster endpoint returns not-found for it and its metrics are excluded from the global endpoint

#### Scenario: Unknown cluster
- **WHEN** the per-cluster exposition endpoint is requested for a cluster name that does not exist
- **THEN** a not-found response is returned

### Requirement: Metrics push to a push gateway
The system SHALL optionally push each cluster's freshly collected metrics (cluster-labeled, in the same form as the global exposition) to a configured Prometheus push-gateway after every collection cycle, with optional basic-authentication credentials.

#### Scenario: Push after collection
- **WHEN** a cluster is configured with a push-gateway URL and a collection cycle completes with a non-empty metric set
- **THEN** the metrics are pushed to that gateway, using basic authentication when a username and password are configured

#### Scenario: Push failure isolation
- **WHEN** pushing metrics to the gateway fails
- **THEN** a warning is logged and the collection cycle result is unaffected

### Requirement: Time-series storage client for graphs
The system SHALL support configuring, per cluster, one or more URLs of a Prometheus-compatible time-series query service as the metrics storage backing graphs, using the first reachable URL with automatic failover to the others, and optional TLS truststore configuration.

#### Scenario: Failover between storage instances
- **WHEN** the currently used storage URL becomes unreachable
- **THEN** subsequent queries are retried against the remaining configured URLs, and an error indicating no live instances is raised only when all fail

### Requirement: Predefined graph catalog
The system SHALL ship a built-in catalog of graph descriptions, each having a unique id, a query template in a PromQL-like language, a type of either instant or range (a range graph carrying a default time period), and a set of named parameters. All query templates SHALL be validated against the query-language grammar at startup, and startup SHALL fail when any template is syntactically invalid after placeholder substitution.

#### Scenario: Startup template validation
- **WHEN** the application starts with a graph description whose query template is syntactically invalid
- **THEN** startup fails with an error identifying the offending graph ids

#### Scenario: Catalog contents
- **WHEN** the catalog is inspected
- **THEN** it includes at least instant and range graphs over broker disk-usage bytes and topic partition offsets, including a topic-parameterized range graph

### Requirement: Graph listing endpoint
The system SHALL provide an authenticated per-cluster endpoint listing available graphs, returning for each its id, type (instant or range), default period as an ISO-8601 duration for range graphs, and its parameter names. The list SHALL be empty when the cluster has no time-series storage configured.

#### Scenario: Listing with storage configured
- **WHEN** a client lists graphs for a cluster that has a time-series storage configured
- **THEN** all catalog graph descriptions are returned with id, type, default period, and parameters

#### Scenario: Listing without storage
- **WHEN** a client lists graphs for a cluster without time-series storage configured
- **THEN** an empty list is returned

### Requirement: Graph data endpoint
The system SHALL provide an authenticated, audited per-cluster endpoint that executes a graph by id with optional from/to timestamps and a map of parameter values, and returns the query result in the response format of the Prometheus HTTP query API (status, result type, and result data).

#### Scenario: Query preparation
- **WHEN** a graph is executed
- **THEN** the query template's placeholders are substituted with the supplied parameter values plus the cluster name bound to the reserved cluster placeholder, before the query is sent to the time-series storage

#### Scenario: Missing parameter
- **WHEN** a graph defines parameters that are absent from the request
- **THEN** the request fails with a validation error naming the missing parameters

#### Scenario: Unknown graph id
- **WHEN** the requested graph id is not in the catalog
- **THEN** a not-found error is returned

#### Scenario: Storage not configured
- **WHEN** graph data is requested for a cluster without time-series storage configured
- **THEN** a validation error indicating the missing storage configuration is returned

#### Scenario: Instant graph execution
- **WHEN** an instant-type graph is executed
- **THEN** a single point-in-time query is issued and its result returned

#### Scenario: Range graph defaults
- **WHEN** a range-type graph is executed without from/to timestamps
- **THEN** the range defaults to the interval ending now and starting the graph's default period ago, and the request is rejected when the effective end is not after the start

#### Scenario: Step-size calculation
- **WHEN** a range query is issued
- **THEN** the query step is chosen so the result targets approximately 200 data points across the requested interval, with a minimum granularity of one second

### Requirement: Access control and auditing for metrics endpoints
The system SHALL enforce cluster-level access authorization on graph listing, graph data, cluster metrics, and broker metrics endpoints, and SHALL record audit entries for graph data retrieval operations.

#### Scenario: Unauthorized access
- **WHEN** a caller without access to the target cluster invokes a graph or metrics endpoint
- **THEN** the request is rejected by the authorization layer

### Requirement: Optional metadata-catalog export integration
The system SHALL support an optional integration, activated by configuring a catalog service URL and access token, that periodically (default every 30 seconds, configurable) exports cluster metadata to an external data-catalog/discovery service using bearer-token authentication. Each cluster SHALL be registered as a data source identified by a canonical resource name derived from its bootstrap servers; topics SHALL be exported as dataset entities and connectors of attached connect clusters as transformer entities with input/output lineage.

#### Scenario: Integration activation
- **WHEN** no catalog service URL is configured
- **THEN** no export components are active and no requests are made to any catalog service

#### Scenario: Topic export
- **WHEN** the export runs for a cluster
- **THEN** each topic passing the topic filter is sent as a dataset entity carrying topic metadata (partition count, replication settings, non-default configuration) and dataset fields derived from its key and value schemas in the schema registry (supporting Avro, JSON Schema, and Protobuf), batched into chunks

#### Scenario: Topic filtering
- **WHEN** a topics filter regular expression is configured
- **THEN** only matching topics are exported; otherwise all topics except those whose names start with an underscore are exported

#### Scenario: Connector export
- **WHEN** the export runs for a cluster with attached connect clusters
- **THEN** each connect cluster is registered as a data source and each connector is exported as a transformer entity whose inputs and outputs link topic and external-system resource names according to the connector's type and topics

#### Scenario: Per-topic export failure tolerance
- **WHEN** exporting one topic's metadata fails
- **THEN** the error is logged and export continues with the remaining topics

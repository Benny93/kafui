# Multi-Cluster Management & Dashboard

## ADDED Requirements

### Requirement: Multiple Independent Cluster Connections
The application SHALL be able to connect to multiple Kafka clusters, each configured independently with its own connection settings, and SHALL manage them concurrently within a single running instance.

#### Scenario: Several clusters configured
- **WHEN** the application is started with configuration entries for several Kafka clusters
- **THEN** the application establishes an independent connection context for each configured cluster and exposes all of them through its API and user interface

#### Scenario: Clusters are isolated from each other
- **WHEN** one configured cluster becomes unreachable
- **THEN** the remaining clusters continue to be served normally and only the unreachable cluster is reported as offline

### Requirement: Cluster Configuration Entries
Each cluster configuration entry SHALL require a cluster name and a list of bootstrap servers, and SHALL support optional per-cluster settings for security, client tuning, and integrations.

#### Scenario: Mandatory fields validated
- **WHEN** a cluster entry is missing a name or bootstrap servers while multiple clusters are configured
- **THEN** the application rejects the configuration at startup with a validation error

#### Scenario: Single unnamed cluster gets a default name
- **WHEN** exactly one cluster is configured and no name is provided for it
- **THEN** the application assigns it a default name automatically and starts normally

#### Scenario: Duplicate cluster names rejected
- **WHEN** two configured cluster entries share the same name
- **THEN** the application fails to start and reports that cluster names must be unique

### Requirement: Per-Cluster Security and Client Options
The application SHALL support per-cluster transport security and authentication settings, including a TLS truststore (location, password, optional certificate verification disable) and arbitrary client properties (for example SASL mechanism, JAAS configuration, and any other client option) passed through to the Kafka clients.

#### Scenario: TLS truststore applied
- **WHEN** a cluster entry specifies a truststore location and password
- **THEN** connections to that cluster's brokers (and, where applicable, its associated services) are made using that truststore

#### Scenario: Arbitrary client properties applied
- **WHEN** a cluster entry contains additional client properties such as SASL settings
- **THEN** those properties are applied to all clients created for that cluster

#### Scenario: Nested property keys flattened
- **WHEN** client properties are provided as nested structures in the configuration
- **THEN** they are flattened into dotted property keys before being passed to the clients

#### Scenario: Separate consumer and producer overrides
- **WHEN** a cluster entry specifies dedicated consumer properties or producer properties
- **THEN** those settings are applied only to consumers or producers respectively, in addition to the common properties

### Requirement: Per-Cluster Optional Integrations Configuration
Each cluster entry SHALL support optional configuration of associated services: a schema registry (with basic-auth or OAuth client-credentials authentication, mutually exclusive, plus optional dedicated keystore), one or more named Kafka Connect clusters (with address, credentials, keystore), a ksqlDB server (with credentials and keystore), and a metrics/monitoring source.

#### Scenario: Schema registry configured
- **WHEN** a cluster entry includes a schema registry address
- **THEN** the application creates a schema registry client for that cluster and marks the schema registry capability as available on it

#### Scenario: Conflicting schema registry authentication rejected
- **WHEN** both basic authentication and OAuth are configured for a cluster's schema registry, or OAuth is only partially configured
- **THEN** the application reports a configuration validation error

#### Scenario: Multiple service addresses with failover
- **WHEN** an integration address value contains a comma-separated list of URLs
- **THEN** the application treats them as equivalent instances and fails over to another live instance when the current one refuses connections

### Requirement: Per-Cluster Read-Only Mode
The application SHALL support a per-cluster read-only flag; when enabled, all state-changing operations against that cluster SHALL be rejected while read operations continue to work.

#### Scenario: Mutating request blocked
- **WHEN** a cluster is configured as read-only and a client submits a state-changing request scoped to that cluster
- **THEN** the request is rejected with an error indicating the cluster is in read-only mode

#### Scenario: Read requests allowed
- **WHEN** a cluster is configured as read-only and a client submits a read request
- **THEN** the request is processed normally

#### Scenario: Non-persistent analysis operations exempt
- **WHEN** a cluster is read-only and a client invokes an operation that uses a mutating request method but does not alter cluster state (for example starting a topic analysis or evaluating a message filter)
- **THEN** the operation is allowed

#### Scenario: Read-only indicated in cluster data
- **WHEN** the cluster list is retrieved
- **THEN** each cluster's payload indicates whether it is read-only, and the dashboard displays a "readonly" badge next to such clusters' names

### Requirement: Cluster List Retrieval
The application SHALL expose an operation that lists all configured clusters together with their current health status and cached overview statistics.

#### Scenario: Listing clusters
- **WHEN** a client requests the cluster list
- **THEN** the response contains, for each cluster: name, status, last collection error (if any), broker count, online partition count, topic count, production rate (bytes in per second), consumption rate (bytes out per second), read-only flag, cluster software version, and the list of enabled capabilities

#### Scenario: List filtered by user permissions
- **WHEN** access control is enabled and a user requests the cluster list
- **THEN** only clusters the user is permitted to view are included in the response

### Requirement: Cluster Health Status
The application SHALL track a health status for every cluster with the values online, offline, and initializing, and SHALL record the most recent collection error for offline clusters.

#### Scenario: Initializing before first collection
- **WHEN** the application has started but the first statistics collection for a cluster has not yet completed
- **THEN** that cluster's status is reported as "initializing"

#### Scenario: Online after successful collection
- **WHEN** a statistics collection cycle for a cluster completes successfully
- **THEN** that cluster's status is reported as "online"

#### Scenario: Offline on collection failure
- **WHEN** a statistics collection cycle for a cluster fails (for example the brokers are unreachable)
- **THEN** the cluster's status is reported as "offline" and the error message and details are exposed as the cluster's last error

### Requirement: Periodic Background Statistics Refresh
The application SHALL periodically refresh each cluster's statistics in the background on a fixed schedule, with a configurable interval defaulting to 30 seconds, collecting from all clusters in parallel.

#### Scenario: Scheduled refresh
- **WHEN** the refresh interval elapses
- **THEN** the application collects fresh statistics for every configured cluster in parallel and replaces the cached values

#### Scenario: Configurable interval
- **WHEN** the operator configures a custom metrics update rate
- **THEN** the background refresh runs at that interval instead of the default

#### Scenario: Reads served from cache
- **WHEN** a client requests cluster information between refresh cycles
- **THEN** the response is served from the most recently cached statistics without triggering a new collection

### Requirement: On-Demand Cluster Statistics Refresh
The application SHALL provide an operation to force an immediate refresh of a single cluster's cached statistics and return the updated cluster overview.

#### Scenario: Manual refresh
- **WHEN** a client invokes the refresh operation for a named cluster
- **THEN** the application collects that cluster's statistics immediately, updates the cache, and returns the refreshed cluster data

### Requirement: Statistics Collection Content
Each statistics collection cycle SHALL gather, per cluster: the broker topology and controller, the cluster software version, topic and partition state, enabled capabilities, connected Kafka Connect states, throughput metrics, and the metadata quorum/coordination type.

#### Scenario: Coordination type detection
- **WHEN** statistics are collected from a cluster
- **THEN** the application determines whether the cluster uses quorum-based (KRaft) or ZooKeeper-based coordination, reporting "unknown" when the cluster forbids quorum inspection (for example some managed offerings)

### Requirement: Detailed Cluster Statistics Retrieval
The application SHALL expose an operation returning detailed statistics for a single cluster, including broker count, active controller identifier, online and offline partition counts, in-sync and out-of-sync replica counts, under-replicated partition count, per-broker disk usage (total segment size and segment count), and version.

#### Scenario: Fetching cluster statistics
- **WHEN** a client requests detailed statistics for a named cluster
- **THEN** the response includes the counts and per-broker disk usage listed above from the cached statistics

#### Scenario: Unknown cluster
- **WHEN** a client requests statistics for a cluster name that is not configured
- **THEN** the application responds with a not-found error

### Requirement: Cluster Metrics Retrieval
The application SHALL expose an operation returning the raw metric values most recently collected for a single cluster.

#### Scenario: Fetching cluster metrics
- **WHEN** a client requests metrics for a named cluster
- **THEN** the response contains the list of collected metric items from the cached statistics

### Requirement: Per-Cluster Capability Detection
The application SHALL determine, on every statistics refresh, the set of capabilities enabled for each cluster — including configured integrations (schema registry, Kafka Connect, ksqlDB, metrics graphs), broker-derived capabilities (topic deletion enabled, ACL viewing, ACL editing, client quota management), and application-level toggles (full-text search availability and default, relative message timestamps) — and SHALL include this set in the cluster payload.

#### Scenario: Integration capabilities from configuration
- **WHEN** a cluster has a schema registry, Kafka Connect clusters, a ksqlDB server, or a metrics store configured
- **THEN** the corresponding capability flags appear in that cluster's capability set

#### Scenario: Topic deletion capability from broker settings
- **WHEN** the cluster's brokers have topic deletion enabled
- **THEN** the topic-deletion capability is included for that cluster; otherwise it is absent

#### Scenario: ACL capabilities from broker authorization
- **WHEN** the cluster reports that security authorization is available
- **THEN** the ACL-view capability is included; and the ACL-edit capability is additionally included only if the authenticated client holds alter-level authorization on the cluster

#### Scenario: Capability-gated UI
- **WHEN** a cluster's capability set lacks a given integration (for example schema registry)
- **THEN** the corresponding navigation entries and pages for that cluster are hidden and their routes are unavailable

### Requirement: Dashboard Cluster Overview Page
The user interface SHALL provide a dashboard page listing all clusters in a table with columns for cluster name, version, broker count, online partition count, topic count, production rate, and consumption rate, with rates rendered in human-readable byte units.

#### Scenario: Dashboard table shown
- **WHEN** a user opens the dashboard
- **THEN** all visible clusters are listed with the columns above, the table supports sorting and column resizing, and column widths persist across visits

#### Scenario: Row navigation
- **WHEN** a user clicks a cluster row in the dashboard table
- **THEN** the interface navigates to that cluster's default detail page

#### Scenario: Empty state
- **WHEN** the cluster list has loaded and contains no clusters
- **THEN** the dashboard displays an explicit "no clusters found" message, and a loading indicator is shown before the list is fetched

### Requirement: Dashboard Health Summary and Offline Filter
The dashboard SHALL display summary counters for the number of online and offline clusters and SHALL offer a toggle to show only offline clusters.

#### Scenario: Summary counters
- **WHEN** the dashboard is displayed
- **THEN** the counts of online clusters and offline clusters are shown as separate labeled indicators

#### Scenario: Offline-only filter
- **WHEN** the user enables the "only offline clusters" toggle
- **THEN** the table shows only clusters whose status is offline; disabling the toggle restores the full list

### Requirement: Sidebar Navigation Across Clusters
The user interface SHALL provide a persistent sidebar containing a dashboard link and one expandable menu section per cluster, each showing the cluster's name and a health status indicator.

#### Scenario: Per-cluster menu entries
- **WHEN** the sidebar renders for a cluster
- **THEN** it shows menu items for brokers, topics, and consumers unconditionally, and additionally schema registry, Kafka Connect, ksqlDB, and ACL entries only when the corresponding capability is enabled for that cluster

#### Scenario: Expand and collapse
- **WHEN** the user toggles a cluster's menu section
- **THEN** the section expands or collapses, and the open/closed state is persisted locally per cluster across sessions

#### Scenario: Auto-expansion
- **WHEN** only one cluster is configured, or the user is currently viewing a page belonging to a cluster
- **THEN** that cluster's menu section is expanded by default and scrolled into view

#### Scenario: Cluster name click
- **WHEN** the user clicks a cluster's name in the sidebar
- **THEN** the menu section opens and the interface navigates to that cluster's default detail page

#### Scenario: Active item highlighting
- **WHEN** the user is on a page corresponding to a sidebar menu item
- **THEN** that menu item is visually highlighted as active

#### Scenario: Cluster color labeling
- **WHEN** the user assigns a color to a cluster's sidebar section
- **THEN** the section is tinted with that color and the choice is persisted locally per cluster

### Requirement: Cluster-Scoped Routing
The user interface SHALL scope all cluster detail pages under a per-cluster route identified by the cluster name, and SHALL redirect the cluster root to the brokers overview.

#### Scenario: Default cluster page
- **WHEN** a user navigates to a cluster's root path
- **THEN** the interface redirects to that cluster's brokers page

#### Scenario: Feature-gated routes rejected
- **WHEN** a user navigates directly to a route for an integration that is not enabled on the current cluster
- **THEN** the route is not served for that cluster

#### Scenario: Read-only propagated to UI
- **WHEN** the current cluster is read-only
- **THEN** all pages within that cluster's scope suppress or disable state-changing actions

### Requirement: Cluster Connection Validation
The application SHALL be able to validate a cluster configuration by actively testing connectivity to the brokers and to each configured integration (schema registry, ksqlDB, each Kafka Connect cluster, metrics store) and reporting per-component success or error details.

#### Scenario: Validating a cluster configuration
- **WHEN** a cluster configuration validation is requested
- **THEN** the result reports, for the broker connection and for each configured integration independently, whether it succeeded or the error encountered

#### Scenario: Invalid truststore short-circuits
- **WHEN** the configured truststore cannot be loaded
- **THEN** validation fails immediately with a truststore-specific error message

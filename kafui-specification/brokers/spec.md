# Broker Management

## ADDED Requirements

### Requirement: Broker listing
The system SHALL provide, per cluster, a list of all brokers currently in the cluster, where each entry includes the broker id, host, and port.

#### Scenario: Listing brokers of a cluster
- **WHEN** a user requests the broker list for a cluster
- **THEN** the system returns one entry per broker node reported by the cluster, each containing the broker id, host name, and port

#### Scenario: Broker list retrieval fails
- **WHEN** the broker list cannot be fetched from the cluster
- **THEN** the user is shown an error state with the failure message and a control to retry the request

#### Scenario: Broker list is loading
- **WHEN** the broker list request is in progress
- **THEN** a loading indicator is displayed instead of the table

#### Scenario: Empty broker list
- **WHEN** the broker list request succeeds but returns no brokers
- **THEN** the table displays an empty-state message indicating no clusters are online

### Requirement: Broker listing metadata
Each broker entry in the listing SHALL be enriched with disk usage (total segment size in bytes and segment count), partition distribution figures (leader count, replica count, in-sync replica count), skew percentages for replicas and leaders, and byte-rate metrics (bytes in per second, bytes out per second) when available.

#### Scenario: Broker with full statistics
- **WHEN** cluster statistics are available for a broker
- **THEN** its list entry shows disk usage as a formatted byte size (2 decimal places) together with the segment count, the number of partitions for which it is leader, its total replica count, its in-sync replica count, and its replica and leader skew values

#### Scenario: Broker without disk usage data
- **WHEN** no disk usage statistics exist for a broker
- **THEN** the disk usage cell displays "N/A" instead of a size and count

#### Scenario: In-sync replica deficit highlighting
- **WHEN** a broker's in-sync replica count is lower than its total replica count
- **THEN** the in-sync replica value is visually highlighted as an attention/alert state

#### Scenario: In-sync replica data missing
- **WHEN** either the in-sync replica count or the replica count is unavailable for a broker
- **THEN** the in-sync replica cell is rendered empty

### Requirement: Active controller indication
The broker listing SHALL visually identify the broker that is the cluster's active controller.

#### Scenario: Active controller badge
- **WHEN** a broker's id equals the cluster's active controller id
- **THEN** that broker's id cell displays a marker with an explanatory tooltip reading "Active Controller"

#### Scenario: No active controller known
- **WHEN** the active controller id is unknown for the cluster
- **THEN** the broker summary panel displays a highlighted "No Active Controller" warning in place of the controller id

### Requirement: Broker list sorting
The broker listing table SHALL support sorting on its columns, including broker id, disk usage, in-sync replicas, replicas, replica skew, leader count, leader skew, port, and host.

#### Scenario: Sorting by a column
- **WHEN** the user activates sorting on a broker table column
- **THEN** rows are reordered according to that column's values

### Requirement: Navigation to broker detail
The broker listing SHALL allow navigation to a per-broker detail page.

#### Scenario: Clicking a broker row
- **WHEN** the user clicks a row (or the broker id link) in the broker list
- **THEN** the application navigates to the detail page of that broker within the current cluster

### Requirement: Brokers summary panel
The brokers page SHALL display a cluster-level summary panel with an uptime section (broker count, active controller id, cluster version) and a partitions section (online partitions out of total, under-replicated partition count, in-sync replicas out of total replicas, out-of-sync replica count, and controller type).

#### Scenario: Offline partitions present
- **WHEN** the offline partition count is greater than zero
- **THEN** the online partition value is rendered in an error style, alongside the total ("X of Y") partition count

#### Scenario: All partitions online
- **WHEN** the offline partition count is zero
- **THEN** the online partition indicator is rendered in a success style

#### Scenario: Under-replicated partitions indicator
- **WHEN** the under-replicated partition count is greater than zero
- **THEN** the value is rendered in an error style; otherwise it is rendered de-emphasized in a success style

#### Scenario: Replica synchronization indicator
- **WHEN** all replicas (in-sync plus out-of-sync) are in sync
- **THEN** the in-sync indicator shows the total in a success style; otherwise the in-sync count is shown in an error style next to the total ("X of Y")

#### Scenario: Controller type display
- **WHEN** the cluster's coordination mode is known
- **THEN** the panel shows "KRaft" or "ZooKeeper" accordingly, and "Unknown" when it cannot be determined

### Requirement: Broker list CSV export
The system SHALL allow exporting the broker list as CSV, both via a user-facing export control and via a machine-readable endpoint.

#### Scenario: Exporting from the UI
- **WHEN** the user activates the "Export CSV" control on the brokers page
- **THEN** a CSV file with a "brokers" name prefix is produced containing the visible table data, with the broker id column annotated with "(Active)" for the active controller, disk usage rendered as "size, N segment(s)" (or "N/A" when absent), and skew values rendered as formatted percentages

#### Scenario: Requesting CSV from the service
- **WHEN** a client requests the broker list in CSV format for a cluster
- **THEN** the service returns the broker list serialized as CSV text

### Requirement: Partition distribution and skew statistics
The system SHALL compute, from the cluster's topic/partition descriptions, per-broker counts of partition replicas, partition leaderships, and in-sync replicas, plus a skew percentage for replicas and for leaders defined as the percentage deviation of the broker's count from the average count across brokers holding that role, rounded to one decimal place.

#### Scenario: Skew computation
- **WHEN** the cluster contains at least 50 partitions and the average per-broker count is non-zero
- **THEN** each broker's skew is computed as ((brokerCount − average) / average) × 100, treating a missing count as zero, and rounded half-up to one decimal place

#### Scenario: Too few partitions for meaningful skew
- **WHEN** the cluster contains fewer than 50 partitions
- **THEN** skew values are not calculated and are reported as absent

#### Scenario: Skew display formatting
- **WHEN** a skew value is present
- **THEN** it is displayed as a percentage with two decimal places; when absent it is displayed as "-"

#### Scenario: Skew severity coloring
- **WHEN** a displayed skew value is at least 10% and below 20%
- **THEN** it is shown in a warning style; when it is 20% or more it is shown in an attention/error style

#### Scenario: Skew column explanation
- **WHEN** the user hovers the replica skew column header's info icon
- **THEN** a tooltip explains that skew is the divergence from the average brokers' value

### Requirement: Broker detail page
The system SHALL provide a per-broker detail page titled with the broker id, offering back-navigation to the broker list, a summary strip (segment size, segment count, port, host), and tabbed sections for log directories (default), configs, and metrics.

#### Scenario: Viewing an existing broker
- **THEN** the summary strip shows the broker's disk segment size formatted as bytes with 2-decimal precision, its segment count, its port, and its host

#### Scenario: Unknown broker id
- **WHEN** the requested broker id does not exist in the cluster's broker list
- **THEN** a not-found error page is shown for that broker with a retry control

#### Scenario: Metrics tab authorization
- **WHEN** the user lacks permission to view cluster configuration
- **THEN** the metrics tab navigation is disabled/gated accordingly

### Requirement: Broker log directories listing
The system SHALL report, per broker, all of its log directories, each with its path name, any error message, and the replicas it hosts grouped by topic with per-partition size in bytes and offset lag.

#### Scenario: Fetching log directories for selected brokers
- **WHEN** a client requests log directories for a cluster with an optional list of broker ids
- **THEN** the system returns log directory entries for the requested brokers only, or for all brokers when no filter is given, ignoring requested ids that are not part of the cluster

#### Scenario: Log directory entry content
- **WHEN** log directory data is returned
- **THEN** each entry contains the directory name, an error message if the directory reported a failure, and a topics collection where each topic lists its partitions with broker id, partition number, size, and offset lag

#### Scenario: Log directory table on broker detail
- **WHEN** the user opens the log directories tab of a broker
- **THEN** a sortable table shows per directory: name, error, the number of topics, and the total number of partitions summed across those topics (zero when no topics)

#### Scenario: Log directory data unavailable
- **WHEN** no log directory data is returned for the broker
- **THEN** the table displays the message "Log dir data not available"

#### Scenario: Log directory query timeout
- **WHEN** fetching log directory descriptions from the cluster times out
- **THEN** the system returns an empty result instead of failing the request

### Requirement: Replica log directory reassignment
The system SHALL allow moving a topic partition's replica on a given broker to a different log directory on that broker, subject to configuration-edit permission.

#### Scenario: Successful reassignment
- **WHEN** a client submits a topic name, partition number, and target log directory for a broker
- **THEN** the replica for that topic partition on that broker is reassigned to the given log directory and a success response is returned

#### Scenario: Unknown topic or partition
- **WHEN** the given topic or partition does not exist
- **THEN** the operation fails with a topic-or-partition-not-found error

#### Scenario: Unknown log directory
- **WHEN** the given log directory does not exist on the broker
- **THEN** the operation fails with a log-directory-not-found error

#### Scenario: Permission enforcement
- **WHEN** the requester lacks both view and edit permission on cluster configuration
- **THEN** the reassignment request is rejected

### Requirement: Broker configuration viewing
The system SHALL expose the full configuration of a broker, where each entry includes the config key, current value, source, sensitivity flag, read-only flag, and synonym chain; viewing requires cluster-configuration view permission.

#### Scenario: Fetching broker configuration
- **WHEN** a client with view permission requests the configuration of an existing broker
- **THEN** the system returns all config entries with name, value, source, sensitive flag, read-only flag, and synonyms

#### Scenario: Configuration for a nonexistent broker
- **WHEN** the requested broker id is not among the cluster's nodes
- **THEN** the request fails with a not-found error identifying the broker id

#### Scenario: Read-only cluster
- **WHEN** the cluster is configured as read-only
- **THEN** every broker config entry is reported as read-only regardless of its intrinsic mutability

### Requirement: Broker configuration table presentation
The broker configs tab SHALL present entries in a table with Key, Value, and Source columns, ordered by source priority: dynamic sources first, then static broker config, then default config, then unknown sources.

#### Scenario: Default ordering by source
- **WHEN** the config table renders
- **THEN** entries whose source is any dynamic config type appear before static broker config entries, which appear before default config entries, which appear before entries of unknown source

#### Scenario: Human-readable source labels
- **WHEN** a config entry's source is displayed
- **THEN** it is shown as a friendly label (e.g. "Dynamic broker config", "Static broker config", "Default config", "Unknown"), and the Source column header offers an info tooltip explaining each source category

#### Scenario: Dynamic value emphasis
- **WHEN** a config entry's source is a broker-specific dynamic config
- **THEN** its value is visually emphasized to distinguish it from non-dynamic values

### Requirement: Broker configuration search
The broker configs tab SHALL provide a search input that filters config entries by key or value using case-insensitive substring matching.

#### Scenario: Filtering by key or value
- **WHEN** the user types a query into the config search field ("Search by Key or Value")
- **THEN** only entries whose key contains the query, or whose value contains the query, case-insensitively, remain visible

### Requirement: Config value display formatting
The system SHALL format displayed config values based on the key's inferred unit and the entry's sensitivity.

#### Scenario: Sensitive value masking
- **WHEN** a config entry is flagged sensitive
- **THEN** its value is displayed as a mask ("**********") with a "Sensitive Value" hint instead of the raw value

#### Scenario: Byte-unit values
- **WHEN** a config key ends in ".bytes" and its value parses to a positive integer
- **THEN** the value is displayed as a human-readable byte size with the exact byte count available as a hint; non-positive or non-numeric values are shown as-is

#### Scenario: Millisecond-unit values
- **WHEN** a config key ends in ".ms"
- **THEN** the value is displayed with the "ms" unit suffix

### Requirement: Broker configuration editing
The system SHALL allow inline editing of a single broker config value by key, requiring cluster-configuration view and edit permission, with user confirmation before applying.

#### Scenario: Entering edit mode
- **WHEN** a user with edit permission activates the Edit control on a config row
- **THEN** the value cell switches to a text input pre-filled with the current value, with Save and Cancel controls

#### Scenario: Read-only property
- **WHEN** a config entry is read-only
- **THEN** its Edit control is disabled and a "Property is read-only" message is conveyed

#### Scenario: Saving a changed value
- **WHEN** the user saves a value different from the current one
- **THEN** a confirmation prompt ("Are you sure you want to change the value?") is shown, and only upon confirmation is the update submitted for that broker and config key

#### Scenario: Saving an unchanged value
- **WHEN** the user saves without modifying the value
- **THEN** no update is submitted and the cell returns to view mode

#### Scenario: Cancelling an edit
- **WHEN** the user cancels editing
- **THEN** the cell returns to view mode without any change being submitted

#### Scenario: Invalid config update
- **WHEN** the cluster rejects the new value as invalid
- **THEN** the operation fails with an invalid-request error carrying the cluster's message

#### Scenario: Unauthorized config update
- **WHEN** the requester lacks view or edit permission on cluster configuration
- **THEN** the update request is rejected

### Requirement: Per-broker metrics retrieval
The system SHALL expose the most recently collected per-broker metric snapshots for a given broker id.

#### Scenario: Metrics available
- **WHEN** a client requests metrics for a broker that has collected metric data
- **THEN** the system returns that broker's metric snapshots

#### Scenario: Metrics unavailable
- **WHEN** no metric data exists for the requested broker
- **THEN** the request yields a not-found response

### Requirement: Broker metrics display
The broker detail metrics tab SHALL render the broker's metrics as a read-only structured (JSON-style) document view.

#### Scenario: Rendering metrics
- **WHEN** metrics data is available for the broker
- **THEN** it is serialized and displayed in a read-only structured viewer

#### Scenario: Metrics data missing
- **WHEN** metrics data is not available
- **THEN** the viewer displays the text "Metrics data not available"

### Requirement: Broker operation auditing
All broker-related operations (listing, metrics retrieval, log directory listing, config viewing, config update, replica log directory reassignment) SHALL be subject to access validation before execution and SHALL be recorded in the audit trail with their operation name and parameters (such as broker id).

#### Scenario: Audited operation
- **WHEN** any broker operation completes, successfully or not
- **THEN** an audit record is emitted identifying the cluster, the operation, and its parameters

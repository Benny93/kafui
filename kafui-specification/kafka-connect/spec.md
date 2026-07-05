# Kafka Connect Integration

## ADDED Requirements

### Requirement: Multiple Connect Clusters per Kafka Cluster
The system SHALL allow an operator to configure zero or more Kafka Connect clusters for each managed Kafka cluster, each identified by a unique name and a service address, with optional basic-auth credentials (username/password), optional TLS keystore settings, and an optional consumer-group naming pattern used to associate sink connectors with their consumer groups.

#### Scenario: Multiple Connect clusters configured
- **WHEN** two or more Connect clusters are configured for a Kafka cluster
- **THEN** each Connect cluster is independently addressable by its configured name in all Connect-related operations

#### Scenario: Unknown Connect cluster referenced
- **WHEN** any Connect operation references a Connect cluster name that is not configured for the given Kafka cluster
- **THEN** the operation fails with a "not found" error identifying the Connect name and Kafka cluster

### Requirement: List Connect Clusters
The system SHALL provide a list of all configured Connect clusters for a Kafka cluster, with an option to include aggregate statistics per Connect cluster.

#### Scenario: List without statistics
- **WHEN** the Connect cluster list is requested without statistics
- **THEN** each configured Connect cluster is returned with at least its name, address, and runtime cluster information (version, commit, cluster id) when available

#### Scenario: List with statistics
- **WHEN** the Connect cluster list is requested with statistics
- **THEN** each Connect cluster additionally reports its total connector count, failed connector count, total task count, and failed task count

#### Scenario: Connect cluster unreachable
- **WHEN** a configured Connect cluster cannot be contacted while collecting cluster information
- **THEN** the Connect cluster is still listed with empty runtime information rather than failing the whole request

#### Scenario: Access filtering
- **WHEN** the requesting user lacks view permission for a particular Connect cluster
- **THEN** that Connect cluster is omitted from the returned list

### Requirement: Connect Clusters Overview Page
The system SHALL present a Connect clusters page showing summary statistics tiles (cluster count, total connectors with failed-connector warning count, total tasks with failed-task warning count) and a filterable, sortable table of Connect clusters with columns for name, version, connector count, and running-task count.

#### Scenario: Navigate from cluster row to its connectors
- **WHEN** the user selects a Connect cluster row
- **THEN** the connectors view opens pre-filtered to connectors of that Connect cluster

#### Scenario: No Connect clusters
- **WHEN** no Connect clusters exist for the Kafka cluster
- **THEN** the table shows an empty-state message indicating there are no Connect clusters

### Requirement: List Connector Names of a Connect Cluster
The system SHALL provide the list of connector names deployed on a specific Connect cluster.

#### Scenario: Names requested
- **WHEN** the connector name list of a Connect cluster is requested by a user with view permission on that Connect cluster
- **THEN** the names of all connectors deployed on it are returned

### Requirement: Aggregated Connector Listing Across Connect Clusters
The system SHALL provide a single aggregated listing of all connectors across every Connect cluster of a Kafka cluster, where each entry includes the Connect cluster name, connector name, connector plugin class, connector type (source or sink), the topics the connector uses, the connector status, total task count, failed task count, and (for sink connectors) the associated consumer group.

#### Scenario: Aggregation across clusters
- **WHEN** the aggregated connector listing is requested
- **THEN** connectors from all configured Connect clusters are combined into one result set

#### Scenario: One Connect cluster failing
- **WHEN** one Connect cluster is unreachable during aggregation
- **THEN** its connectors are omitted and connectors from the remaining Connect clusters are still returned

#### Scenario: Topics unavailable on old Connect versions
- **WHEN** a Connect cluster does not support reporting the topics used by a connector
- **THEN** the connector is returned with an empty topic list instead of an error

#### Scenario: Access filtering per connector
- **WHEN** the requesting user lacks view permission for a specific connector
- **THEN** that connector is excluded from the aggregated listing

### Requirement: Connector Search and Full-Text Search
The system SHALL support filtering the aggregated connector listing by a search term, with two matching modes: plain substring matching and an optional full-text (n-gram) search mode that can be toggled per request and defaulted by server configuration.

#### Scenario: Search term supplied
- **WHEN** a search term is provided with the aggregated connector listing request
- **THEN** only connectors matching the term (by attributes such as name, status, or type) are returned

#### Scenario: Full-text mode toggled
- **WHEN** the full-text search flag is enabled for the request
- **THEN** matching uses the full-text index semantics; otherwise the configured default mode applies

#### Scenario: Search input in the UI
- **WHEN** the user types into the connectors search field
- **THEN** the search term is reflected in the page's query state and the listing is re-fetched with that term, with a control to toggle full-text mode

### Requirement: Connector Listing Sort Order
The system SHALL support server-side sorting of the aggregated connector listing by name, Connect cluster, type, or status, in ascending or descending order, defaulting to ascending by name.

#### Scenario: Sort by status descending
- **WHEN** the listing is requested sorted by status in descending order
- **THEN** connectors are returned ordered by their status state in reverse order

#### Scenario: No sort specified
- **WHEN** no sort column is specified
- **THEN** results are ordered by connector name ascending

### Requirement: Connectors Table Presentation
The system SHALL present the aggregated connectors in a table with columns for Name (linking to connector details), Connect cluster, Type, Plugin class, Topics (each topic linking to its topic page), Status, Consumer group (linking to consumer group details), Running Tasks (running/total with failure indication), and a per-row actions menu; the table SHALL support client-side pagination, column sorting, per-column filters (multi-select for Connect, Type, Plugin, Topics, Status; text for Name and Consumers), column resizing with persisted widths, and filter state persisted in the URL query.

#### Scenario: Empty result
- **WHEN** no connectors match the current search and filters
- **THEN** the table shows a "no connectors found" empty-state message

#### Scenario: Statistics tiles reflect the filtered view
- **WHEN** table filters reduce the visible connector rows
- **THEN** the summary tiles above the table (connector count with failed-connector warning, task count with failed-task warning) recompute from the currently filtered rows

### Requirement: CSV Export
The system SHALL support exporting both the Connect clusters listing and the aggregated connectors listing as CSV files reflecting the current view.

#### Scenario: Export connectors
- **WHEN** the user triggers CSV export on the connectors view
- **THEN** a CSV file of the current connectors table (including topics joined into a single field and status state) is produced with a connectors-specific file name prefix

#### Scenario: Export Connect clusters
- **WHEN** the user triggers CSV export on the Connect clusters view
- **THEN** a CSV file of the Connect clusters table is produced with a clusters-specific file name prefix

### Requirement: Connector Details
The system SHALL provide a detail view for a single connector including its name, type, plugin class, current configuration, status (state, worker id, and error trace when present), the topics it uses, its task list, and, for sink connectors, a link to the associated consumer group; secret-like configuration values SHALL be masked.

#### Scenario: Details requested
- **WHEN** a connector's details are requested by a user with view permission
- **THEN** the connector's configuration, topics, and status are returned in one consistent view

#### Scenario: Status endpoint missing the connector
- **WHEN** the connector exists but its status cannot be found on the Connect cluster
- **THEN** the connector is reported with an "unassigned" state and an empty task list instead of an error

#### Scenario: Failed connector trace
- **WHEN** the connector state is FAILED and an error trace is available
- **THEN** the state indicator is clickable and opens a dialog showing the worker id and the full error trace

#### Scenario: Overview metrics
- **WHEN** the detail view is displayed
- **THEN** it shows worker id, type, plugin class, state, count of running tasks, and count of failed tasks (highlighted as an error when greater than zero)

### Requirement: Connector Tasks with Status
The system SHALL list a connector's tasks, each with its task id, assigned worker, state (RUNNING, FAILED, PAUSED, RESTARTING, or UNASSIGNED), and error trace when present.

#### Scenario: Tasks listed
- **WHEN** the task list of a connector is requested
- **THEN** every task is returned with its id, worker, state, and any trace

#### Scenario: Long trace display
- **WHEN** a task has an error trace longer than the display limit
- **THEN** the trace is truncated in the table row and the row can be expanded to show the full trace

#### Scenario: Connector without tasks endpoint data
- **WHEN** task information cannot be found for the connector
- **THEN** an empty task list is returned instead of an error

### Requirement: Create Connector
The system SHALL allow creating a new connector on a chosen Connect cluster by supplying a connector name and a JSON configuration object, rejecting creation when a connector with the same name already exists on that Connect cluster.

#### Scenario: Successful creation
- **WHEN** a valid name and JSON configuration are submitted for a Connect cluster
- **THEN** the connector is created and the user is taken to the new connector's detail view

#### Scenario: Duplicate name
- **WHEN** a connector with the submitted name already exists on the target Connect cluster
- **THEN** creation is rejected with a validation error stating the connector already exists

#### Scenario: Creation form validation
- **WHEN** the creation form is displayed
- **THEN** it requires a non-empty name and a syntactically valid JSON object configuration before submission is enabled, offers a Connect cluster selector (hidden when only one Connect cluster exists, defaulting to the first), and disables the create entry point entirely when no Connect clusters are available or the cluster is in read-only mode

### Requirement: Update Connector Configuration
The system SHALL allow retrieving and replacing a connector's configuration as a JSON object, returning the updated connector on success.

#### Scenario: Config viewed and edited
- **WHEN** the user opens the connector's configuration tab
- **THEN** the current (secret-masked) configuration is shown in a JSON editor and can be submitted only when it is a valid JSON object and has been modified

#### Scenario: Masked secrets warning
- **WHEN** the displayed configuration contains masked secret placeholders
- **THEN** a warning instructs the user to replace the placeholders with real values before saving to avoid breaking the connector

#### Scenario: Config updated
- **WHEN** a valid replacement configuration is submitted by a user with edit permission
- **THEN** the connector's configuration is replaced and the updated connector is returned

### Requirement: Delete Connector
The system SHALL allow deleting a connector from its Connect cluster after user confirmation.

#### Scenario: Deletion confirmed
- **WHEN** the user confirms deletion of a connector
- **THEN** the connector is removed and, when deleted from the detail view, the user is returned to the connectors listing

### Requirement: Connector Lifecycle State Changes
The system SHALL support the connector state actions pause, resume, stop, and restart, applied to a named connector on a named Connect cluster.

#### Scenario: Pause or stop a running connector
- **WHEN** the connector state is RUNNING
- **THEN** the actions menu offers Pause and Stop, and invoking one transitions the connector accordingly

#### Scenario: Resume a paused or stopped connector
- **WHEN** the connector state is PAUSED or STOPPED
- **THEN** the actions menu offers Resume, and invoking it returns the connector to running

#### Scenario: Restart connector
- **WHEN** the Restart Connector action is invoked
- **THEN** the connector instance is restarted without forcing a restart of its tasks

### Requirement: Restart All or Failed Tasks
The system SHALL provide bulk task-restart actions on a connector: restart all tasks, and restart only tasks whose state is FAILED, implemented by restarting each matching task individually.

#### Scenario: Restart all tasks
- **WHEN** the Restart All Tasks action is invoked
- **THEN** every task of the connector is restarted

#### Scenario: Restart failed tasks
- **WHEN** the Restart Failed Tasks action is invoked
- **THEN** only tasks currently in the FAILED state are restarted

### Requirement: Restart Individual Task
The system SHALL allow restarting a single connector task by its task id, guarded by a confirmation prompt in the UI.

#### Scenario: Task restart confirmed
- **WHEN** the user confirms the restart of a specific task
- **THEN** that task is restarted on the Connect cluster

### Requirement: Reset Connector Offsets
The system SHALL allow resetting a connector's offsets, permitting the operation only for connectors in the STOPPED state and requiring user confirmation.

#### Scenario: Reset on stopped connector
- **WHEN** the connector is STOPPED and the user confirms the reset
- **THEN** the connector's offsets are reset

#### Scenario: Reset on non-stopped connector
- **WHEN** an offset reset is attempted while the connector is not STOPPED
- **THEN** the operation fails with an error explaining the connector must be stopped first, and the UI keeps the action disabled unless the state is STOPPED

#### Scenario: Connector missing
- **WHEN** an offset reset targets a connector that does not exist
- **THEN** a "not found" error identifying the connector and Connect cluster is returned

### Requirement: List Available Connector Plugins
The system SHALL list the connector plugins installed on a Connect cluster, including each plugin's class, type, and version.

#### Scenario: Plugins requested
- **WHEN** the plugin list of a Connect cluster is requested by a user with view permission
- **THEN** all installed connector plugins are returned

### Requirement: Validate Connector Configuration Against a Plugin
The system SHALL validate a candidate connector configuration against a named connector plugin on a Connect cluster and return the validation outcome, including the total error count, configuration groups, and per-field definitions with any per-field error messages.

#### Scenario: Invalid configuration
- **WHEN** a configuration missing required fields is validated against a plugin
- **THEN** the response reports a non-zero error count and identifies the offending fields with their error messages

#### Scenario: Valid configuration
- **WHEN** a configuration satisfying the plugin's requirements is validated
- **THEN** the response reports an error count of zero

### Requirement: Secret Masking in Connector Configurations
The system SHALL mask values of secret-like configuration keys (such as passwords, keys, and tokens) whenever connector configurations are returned or displayed.

#### Scenario: Config with secrets returned
- **WHEN** a connector configuration containing secret-like keys is retrieved
- **THEN** the corresponding values are replaced with a masking placeholder

### Requirement: Permission-Gated Connector Actions
The system SHALL enforce per-action authorization on Connect resources: viewing requires view permission, creation requires create permission, configuration changes require edit permission, lifecycle and restart actions require operate permission, offset resets require a dedicated reset-offsets permission, and deletion requires delete permission; permissions may be granted at the Connect-cluster or individual-connector scope, and mutating operations SHALL be recorded in the audit trail.

#### Scenario: Unauthorized action
- **WHEN** a user without the required permission attempts a connector action
- **THEN** the action is rejected and the corresponding UI control is unavailable

#### Scenario: Read-only cluster mode
- **WHEN** the Kafka cluster is configured as read-only
- **THEN** the connector creation entry point is hidden

### Requirement: Connect Navigation Structure
The system SHALL organize the Connect area under a per-Kafka-cluster section with two tabs — Connect clusters and Connectors (defaulting to the clusters tab) — and a connector detail page with Tasks (default), Config, and Topics tabs.

#### Scenario: Entering the Connect area
- **WHEN** the user opens the Connect section for a Kafka cluster
- **THEN** the Connect clusters tab is shown by default with the create-connector and CSV-export actions in the page header

#### Scenario: Connector topics tab
- **WHEN** the user opens the Topics tab of a connector
- **THEN** the topics used by the connector are listed, each linking to that topic's detail page, with an empty-state message when there are none

### Requirement: Loading and Error States
The system SHALL show a loading indicator while Connect listings are being fetched and, on fetch failure, an error page with the failure status and a retry control.

#### Scenario: Fetch failure with retry
- **WHEN** loading the Connect clusters or connectors listing fails
- **THEN** an error view with the status and message is shown, and activating retry re-issues the request

# Topic Management

## ADDED Requirements

### Requirement: Topic List Retrieval
The system SHALL provide a paginated list of topics for a selected cluster, returning the topics for the requested page together with the total page count.

#### Scenario: Default pagination
- **WHEN** a user requests the topic list without specifying a page or page size
- **THEN** the system returns the first page using a default page size of 25 topics and includes the total number of pages

#### Scenario: Explicit page selection
- **WHEN** a user requests a specific page and page size
- **THEN** the system returns only the topics belonging to that page slice and the page count computed from the filtered result set

#### Scenario: Only existing topics are listed
- **WHEN** cached topic metadata references a topic that no longer exists on the cluster
- **THEN** that topic is excluded from the returned list

#### Scenario: Empty result
- **WHEN** no topics match the current filters
- **THEN** the list view displays an explicit "no topics found" indication

### Requirement: Topic List Search
The system SHALL support filtering the topic list by a search term matched against topic names, with an optional full-text search mode.

#### Scenario: Search by name
- **WHEN** a user enters a search term
- **THEN** only topics whose names match the term are returned and pagination is applied to the filtered set

#### Scenario: Full-text search relevance ordering
- **WHEN** full-text search mode is enabled and no explicit sort column is selected
- **THEN** results are returned in relevance order instead of alphabetical order

### Requirement: Internal Topics Visibility Toggle
The system SHALL classify topics as internal based on a configurable internal-name prefix and SHALL allow users to show or hide internal topics in the list.

#### Scenario: Hiding internal topics
- **WHEN** the "show internal topics" option is disabled
- **THEN** internal topics are excluded from the list and from pagination

#### Scenario: Preference persistence
- **WHEN** a user toggles internal-topic visibility
- **THEN** the preference is persisted locally and reapplied on subsequent visits, and the list is reset to the first page

#### Scenario: Internal topic labeling
- **WHEN** internal topics are shown
- **THEN** each internal topic is visually marked as internal in the list

### Requirement: Topic List Sorting
The system SHALL support sorting the topic list, in ascending or descending order, by name, total partitions, out-of-sync replica count, replication factor, message count, or size.

#### Scenario: Default sort
- **WHEN** no sort column is specified and full-text search is not active
- **THEN** topics are sorted by name in ascending order

#### Scenario: Descending sort
- **WHEN** a user selects a sort column with descending order
- **THEN** the full filtered topic set is sorted accordingly before pagination is applied

### Requirement: Topic List Columns
The system SHALL display for each listed topic: its name, partition count, number of out-of-sync replicas, replication factor, message count, and on-disk size.

#### Scenario: Message count unavailable
- **WHEN** the message count for a topic cannot be determined
- **THEN** the list displays a "not available" indication for that topic instead of a number

#### Scenario: Size formatting
- **WHEN** topic size is displayed
- **THEN** it is rendered in human-readable byte units

### Requirement: Topic List Export
The system SHALL allow exporting the current topic list (respecting the active search, internal-visibility, and sort settings) as a CSV file.

#### Scenario: CSV download
- **WHEN** a user triggers the export action
- **THEN** a CSV file containing all matching topics (not just the current page) is generated for download with a name derived from the cluster

### Requirement: Topic Creation
The system SHALL allow creating a topic by specifying a name, partition count, optional replication factor, and configuration settings including cleanup policy, retention time, maximum partition size, maximum message size, minimum in-sync replicas, and arbitrary custom configuration entries.

#### Scenario: Successful creation
- **WHEN** a user submits a valid creation form
- **THEN** the topic is created on the cluster with the provided partition count, replication factor, and configuration map, and the user is navigated to the new topic's detail page

#### Scenario: Replication factor omitted
- **WHEN** the replication factor field is left empty
- **THEN** the topic is created using the cluster's default replication factor

#### Scenario: Empty configuration values omitted
- **WHEN** any configuration value is left empty
- **THEN** that configuration entry is excluded from the creation request

#### Scenario: Name validation
- **WHEN** a user enters a topic name that is empty, longer than 249 characters, or contains characters other than alphanumerics, underscore, hyphen, and dot
- **THEN** the form reports a validation error and submission is blocked

#### Scenario: Partition count validation
- **WHEN** a user enters a partition count below 1 or a non-numeric value
- **THEN** the form reports a validation error and submission is blocked

#### Scenario: Custom parameter validation
- **WHEN** a user adds custom configuration entries with an empty name, an empty value, or duplicate names
- **THEN** the form reports a validation error identifying the offending entry

#### Scenario: Cleanup policy choices
- **WHEN** a user selects a cleanup policy
- **THEN** the options offered are delete, compact, and combined compact-with-delete

#### Scenario: Retention time helper
- **WHEN** a user edits the retention time in milliseconds
- **THEN** the form shows a human-readable duration hint and offers quick-select presets, and values below -1 are rejected

#### Scenario: Visibility after creation
- **WHEN** a topic was just created but its metadata is not yet visible on the cluster
- **THEN** the system retries loading it for a bounded period before reporting a metadata error

#### Scenario: Creation failure
- **WHEN** the cluster rejects the creation request
- **THEN** the error is surfaced to the user and no navigation occurs

### Requirement: Topic Cloning
The system SHALL allow creating a copy of an existing topic, either as a server-side clone using the source topic's partition count, replication factor, and configuration, or by prefilling the creation form with the source topic's settings for a user-chosen new name.

#### Scenario: Copy from list selection
- **WHEN** exactly one topic is selected in the list and the user chooses the copy action
- **THEN** the creation form opens prefilled with the selected topic's name and settings for editing before submission

#### Scenario: Copy selection constraint
- **WHEN** zero or more than one topic is selected
- **THEN** the copy action is disabled

### Requirement: Topic Details Overview
The system SHALL provide a topic detail view summarizing partition count, replication factor, count of under-replicated partitions, in-sync replica count out of total replicas, internal/external classification, total segment size, segment count, cleanup policy, and total message count.

#### Scenario: Overview metrics display
- **WHEN** a user opens a topic's detail page
- **THEN** all summary metrics listed above are shown, with under-replicated partition and in-sync replica indicators highlighted as healthy when fully replicated and as an alert otherwise

#### Scenario: Message count derivation
- **WHEN** the total message count is displayed
- **THEN** it equals the sum over all partitions of (latest offset minus earliest offset)

#### Scenario: Topic not found
- **WHEN** a user opens the detail page of a topic that does not exist or is not visible
- **THEN** an error page with the failure status is shown and a retry action is offered

### Requirement: Per-Partition Information
The topic detail view SHALL list every partition with its identifier, replica assignment (highlighting the leader and out-of-sync replicas), earliest offset, next offset, and per-partition message count.

#### Scenario: Partition table
- **WHEN** a user views a topic's overview
- **THEN** each partition row shows the partition id, its replicas with leader and out-of-sync status marked, first offset, next offset, and message count computed as the offset range

### Requirement: Topic Configuration Listing
The system SHALL list all configuration entries of a topic, including each entry's name, effective value, and default value, and SHALL distinguish entries whose value differs from the default.

#### Scenario: Config listing
- **WHEN** a user opens the topic settings view
- **THEN** all configuration entries are listed with name, current value, and default value, and entries overriding their default are visually distinguished

#### Scenario: Sensitive values masked
- **WHEN** a configuration entry is marked sensitive
- **THEN** its value and default value are masked in the display

#### Scenario: Human-readable units
- **WHEN** a configuration value represents a duration in milliseconds or a size in bytes
- **THEN** a human-readable formatted equivalent is displayed alongside the raw value, except for non-positive values that denote "unbounded"

#### Scenario: Config read permission missing
- **WHEN** the topic exists but the system lacks permission to describe its configuration
- **THEN** an empty configuration list is returned rather than an error

### Requirement: Topic Settings Editing
The system SHALL allow editing an existing topic's configuration (cleanup policy, retention time, retention size, maximum message size, minimum in-sync replicas, and custom entries) while keeping the topic name and partition count immutable in the edit form.

#### Scenario: Immutable fields
- **WHEN** a user opens the edit form for an existing topic
- **THEN** the topic name and partition count fields are not editable

#### Scenario: Update submission
- **WHEN** a user submits changed configuration values
- **THEN** the system applies the configuration update to the topic, confirms success to the user, and returns to the topic detail view

#### Scenario: Only meaningful changes sent
- **WHEN** the update request is built
- **THEN** configuration entries that are unchanged and still at their default source are excluded from the request

### Requirement: Partition Count Increase
The system SHALL allow increasing a topic's partition count to a specified total, and SHALL reject decreases or no-op requests.

#### Scenario: Successful increase
- **WHEN** a user requests a total partition count greater than the current count and confirms the warning prompt
- **THEN** the partitions are created and the response reports the topic name and new total partition count

#### Scenario: Decrease rejected
- **WHEN** the requested total is lower than the current partition count
- **THEN** the request is rejected with a validation error stating the current count is higher than requested

#### Scenario: Equal count rejected
- **WHEN** the requested total equals the current partition count
- **THEN** the request is rejected with a validation error stating the topic already has that many partitions

#### Scenario: Confirmation required
- **WHEN** a user submits a partition increase from the UI
- **THEN** an explicit confirmation dialog warning about the operation's consequences is shown before execution

### Requirement: Replication Factor Change
The system SHALL allow changing a topic's replication factor by computing and applying a partition replica reassignment across the cluster's online brokers.

#### Scenario: Successful change
- **WHEN** a user requests a valid new replication factor and confirms
- **THEN** replicas are added to or removed from each partition to reach the requested factor and the response reports the topic name and new total replication factor

#### Scenario: Validation of requested factor
- **WHEN** the requested factor equals the current factor, is less than 1, or exceeds the number of brokers in the cluster
- **THEN** the request is rejected with a validation error describing the reason

#### Scenario: Balanced replica placement on increase
- **WHEN** the replication factor is increased
- **THEN** new replicas are assigned preferring the brokers currently hosting the fewest replicas

#### Scenario: Leader preservation on decrease
- **WHEN** the replication factor is decreased
- **THEN** the partition leader's replica is never removed, and removals prefer the most-loaded brokers

#### Scenario: Offline brokers excluded
- **WHEN** the reassignment is computed
- **THEN** only brokers currently online are considered as replica targets

### Requirement: Topic Deletion
The system SHALL allow deleting a topic after user confirmation, provided topic deletion is enabled on the cluster.

#### Scenario: Successful deletion
- **WHEN** a user confirms deletion of a topic
- **THEN** the topic is deleted from the cluster, a success notification names the deleted topic, and from the detail page the user is returned to the topic list

#### Scenario: Deletion disabled on cluster
- **WHEN** the cluster's broker configuration restricts topic deletion
- **THEN** the delete (and recreate) actions are disabled with an explanatory hint, and a direct deletion request is rejected with a validation error

### Requirement: Topic Recreation
The system SHALL allow recreating a topic (delete and re-create) preserving its name, partition count, replication factor, and configuration.

#### Scenario: Successful recreation
- **WHEN** a user confirms the recreate action
- **THEN** the topic is deleted and re-created with identical partition count, replication factor, and configuration, and success is confirmed to the user

#### Scenario: Deletion still propagating
- **WHEN** re-creation fails because the topic still exists after the delete
- **THEN** the system retries creation on a fixed delay up to a configurable maximum, and reports a recreation timeout error if retries are exhausted

### Requirement: Clear Topic Messages
The system SHALL allow purging all messages from a topic, or from a single selected partition, after user confirmation; this SHALL only be permitted for topics whose cleanup policy includes delete.

#### Scenario: Clear whole topic
- **WHEN** a user confirms the clear-messages action on a topic with delete cleanup policy
- **THEN** all messages in all partitions are purged and a success notification is shown

#### Scenario: Clear single partition
- **WHEN** a user invokes clear-messages on a specific partition row in the topic overview
- **THEN** only that partition's messages are purged

#### Scenario: Non-delete cleanup policy
- **WHEN** a topic's cleanup policy does not include delete
- **THEN** the clear-messages action is disabled with a hint explaining the restriction

### Requirement: Bulk Topic Operations
The topic list SHALL support selecting multiple topics and performing batch deletion and batch message purge, each guarded by a confirmation prompt.

#### Scenario: Batch delete
- **WHEN** a user selects several topics and confirms batch deletion
- **THEN** every selected topic is deleted and the selection is cleared

#### Scenario: Batch purge
- **WHEN** a user selects several topics and confirms batch message purge
- **THEN** the messages of every selected topic are cleared, the selection is cleared, and the list is refreshed

#### Scenario: Internal topics not selectable
- **WHEN** the list is rendered
- **THEN** internal topics cannot be selected for batch operations

#### Scenario: Permission-gated batch actions
- **WHEN** the user lacks the required permission for any topic in the selection
- **THEN** the corresponding batch action is presented as not permitted

### Requirement: Per-Topic Row Actions
Each non-internal topic row in the list SHALL offer a contextual action menu with clear messages, recreate topic, and remove topic, each requiring confirmation.

#### Scenario: Actions disabled for internal or read-only
- **WHEN** a topic is internal or the cluster is in read-only mode
- **THEN** the row action menu is disabled

### Requirement: Topic Detail Navigation
The topic detail page SHALL organize content into sections for overview, messages, consumers, settings, statistics, ACLs, and (when at least one exists) connectors, and SHALL offer edit settings, clear messages, recreate, and remove actions from the page header.

#### Scenario: Connectors section conditional
- **WHEN** no data-integration connectors reference the topic
- **THEN** the connectors section is not shown

#### Scenario: Header actions disabled
- **WHEN** the cluster is read-only or the topic is internal
- **THEN** the header action menu is disabled

### Requirement: Consumer Groups on Topic
The system SHALL list all consumer groups consuming a topic, showing group id (linked to the group's detail page), number of active consumers, consumer lag for the topic with a trend indicator, coordinator broker id, and group state.

#### Scenario: Consumer group listing
- **WHEN** a user opens the consumers section of a topic
- **THEN** all consumer groups for the topic are listed with the fields above and an empty-state message when none exist

#### Scenario: Group filtering
- **WHEN** a user types a search term
- **THEN** the group list is filtered by group id, case-insensitively

#### Scenario: Periodic lag refresh
- **WHEN** a user selects a refresh interval
- **THEN** lag values are re-fetched on that interval, the chosen interval is persisted locally, and lag trends (rising/falling) are indicated

### Requirement: Active Producer States
The system SHALL expose the active (transactional/idempotent) producer states for a topic, ordered by partition ascending and, within a partition, by producer id descending.

#### Scenario: Producer state listing
- **WHEN** the active producer states for a topic are requested
- **THEN** every active producer session is returned with its partition and producer identity in the defined order

### Requirement: Topic Access Rules Listing
The system SHALL list the access-control entries whose resource pattern matches a given topic.

#### Scenario: Matching ACLs
- **WHEN** a user opens the ACLs section of a topic
- **THEN** all access-control entries matching the topic name (including wildcard/prefix patterns) are listed

### Requirement: Topic Connectors Listing
The system SHALL list the data-integration connectors that read from or write to a given topic, filtered to those the user is allowed to see.

#### Scenario: Connector listing
- **WHEN** connectors referencing the topic exist
- **THEN** they are listed in the topic's connectors section

### Requirement: Topic Analysis Start
The system SHALL allow starting a background analysis of a topic that scans every message from the earliest offset to the current end of each partition and aggregates statistics.

#### Scenario: Start analysis
- **WHEN** a user starts an analysis for an existing topic
- **THEN** the request is acknowledged immediately and the scan proceeds asynchronously in the background

#### Scenario: Analysis of missing topic
- **WHEN** an analysis is requested for a topic that does not exist
- **THEN** the request fails with a not-found error and no analysis is started

#### Scenario: Duplicate analysis rejected
- **WHEN** an analysis is already running for the same topic on the same cluster
- **THEN** a new start request is rejected with an error stating the topic is already being analyzed

#### Scenario: No prior analysis
- **WHEN** the statistics section is opened for a topic that has never been analyzed
- **THEN** a "start analysis" action is offered instead of results

### Requirement: Topic Analysis Progress
While an analysis is running, the system SHALL report its progress including start time, completion percentage, number of messages scanned, and number of bytes scanned.

#### Scenario: Progress display
- **WHEN** an analysis is in progress and the statistics section is open
- **THEN** the UI polls the analysis state about once per second and shows a progress bar with completion percentage, elapsed time, messages scanned, and bytes scanned

#### Scenario: Completion percentage derivation
- **WHEN** progress is computed
- **THEN** the percentage equals processed offsets divided by the total offset range across scanned partitions, capped at 100

### Requirement: Topic Analysis Cancellation
The system SHALL allow cancelling a running topic analysis.

#### Scenario: User cancellation
- **WHEN** a user stops a running analysis
- **THEN** the background scan is interrupted, its consumer resources are released, the cancellation is confirmed to the user, and no result is recorded for the cancelled run

### Requirement: Topic Analysis Results
On completion, the system SHALL store and return the analysis result containing start and finish timestamps plus aggregate statistics for the whole topic and for each partition: total message count, minimum and maximum offsets, minimum and maximum timestamps, null key and null value counts, approximate distinct key and value counts, and key and value size distributions (sum, minimum, maximum, average, and 50th/75th/95th/99th/99.9th percentiles).

#### Scenario: Result display
- **WHEN** a completed analysis exists and the statistics section is opened
- **THEN** the totals (message count, offset range, timestamp range, null keys/values, approximate unique keys/values) and key/value size distributions are displayed, along with the completion time and a restart action

#### Scenario: Per-partition breakdown
- **WHEN** result rows for individual partitions are expanded
- **THEN** the same statistics are shown scoped to that partition

#### Scenario: Distinct count estimation bound
- **WHEN** approximate distinct key or value counts are reported
- **THEN** the reported estimate never exceeds the total message count

#### Scenario: Hourly message counts
- **WHEN** an analysis completes
- **THEN** the result includes per-hour message counts for messages with timestamps within the last 14 days, ordered chronologically

#### Scenario: Empty topic
- **WHEN** an analysis completes on a topic with no messages
- **THEN** the statistics section states that the topic appears to be empty instead of showing zeros

#### Scenario: Only latest result retained
- **WHEN** an analysis is re-run for a topic
- **THEN** the new result replaces the previously stored result for that topic

### Requirement: Topic Analysis Failure Reporting
If an analysis terminates with an error, the system SHALL record the failure with start and finish timestamps and error details, and return it in place of statistics.

#### Scenario: Failed analysis
- **WHEN** the background scan aborts due to an error
- **THEN** subsequent analysis queries return a result carrying the error description and timing information rather than statistics

### Requirement: Permission-Gated Topic Operations
All topic operations SHALL be gated by per-topic, per-action authorization: viewing requires view rights; creation, cloning, and recreation require create rights; configuration and partition/replication changes require edit rights; deletion requires delete rights; message purge requires message-delete rights; and running or viewing analysis requires the corresponding analysis rights.

#### Scenario: Unauthorized operation
- **WHEN** a user without the required right attempts an operation
- **THEN** the operation is refused and the corresponding UI controls are presented as not permitted

#### Scenario: Read-only cluster
- **WHEN** a cluster is configured as read-only
- **THEN** all mutating topic actions (create, edit, delete, recreate, purge) are disabled in the UI

### Requirement: Operation Auditing
The system SHALL record an audit entry for every topic operation, capturing the cluster, operation name, relevant parameters, and outcome.

#### Scenario: Audited action
- **WHEN** any topic operation completes, successfully or not
- **THEN** an audit record of the operation and its outcome is emitted

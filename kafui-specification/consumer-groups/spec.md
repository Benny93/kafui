# Consumer Groups & Offset Management

## ADDED Requirements

### Requirement: Consumer Group List with Pagination
The system SHALL provide a paginated list of consumer groups for a selected cluster, returning the groups for the requested page together with the total page count.

#### Scenario: Default pagination
- **WHEN** the consumer group list is requested without an explicit page number or page size
- **THEN** the first page is returned using a configurable default page size (25 unless configured otherwise), together with the total number of pages computed from the count of all matching groups

#### Scenario: Explicit page selection
- **WHEN** the consumer group list is requested with a page number and page size
- **THEN** the system returns only the groups falling into that page slice, applied after filtering and sorting

#### Scenario: Invalid paging values
- **WHEN** a page number or page size of zero or less is supplied
- **THEN** the system substitutes the default value (page 1, default page size) instead of failing

### Requirement: Consumer Group List Entry Content
Each entry in the consumer group list SHALL expose the group id, the number of members, the number of topics the group relates to, the total consumer lag, the coordinator broker, the group state, whether the group is a simple consumer group, and the partition assignor.

#### Scenario: Viewing the list
- **WHEN** a user views the consumer group list
- **THEN** each row shows group id (linking to the group's detail view), number of members, number of topics, consumer lag, coordinator broker id, and state, with an explanatory tooltip for each state value

#### Scenario: Topic count derivation
- **WHEN** the number of topics for a group is computed
- **THEN** it equals the count of distinct topics appearing either in the group's committed offsets or in any active member's partition assignment

### Requirement: Consumer Group Search
The system SHALL support filtering the consumer group list by a search string matched against the group id, with an optional full-text search mode that can be toggled by the user and defaulted by configuration.

#### Scenario: Searching by group id
- **WHEN** a user enters a search string
- **THEN** only consumer groups whose id matches the search string are returned, and pagination and page count reflect the filtered set

#### Scenario: Full-text search toggle
- **WHEN** full-text search is enabled for the consumer group resource
- **THEN** the search string is evaluated with full-text matching semantics instead of plain substring/prefix matching

### Requirement: Consumer Group Filter by State
The system SHALL support filtering the consumer group list by one or more consumer group states (Stable, Preparing Rebalance, Completing Rebalance, Empty, Dead, Unknown).

#### Scenario: Multi-state filter
- **WHEN** a user selects one or more states in the state filter
- **THEN** only groups currently in one of the selected states are returned; a group whose state cannot be determined is treated as Unknown

#### Scenario: No state filter
- **WHEN** no state filter is provided
- **THEN** groups in all states are returned

### Requirement: Consumer Group List Sorting
The system SHALL support sorting the consumer group list by name, state, number of members, number of topics, or total consumer lag, in ascending or descending order, with sorting applied before pagination.

#### Scenario: Default ordering
- **WHEN** no sort field is specified
- **THEN** groups are sorted by group name in ascending order

#### Scenario: Sorting by state
- **WHEN** sorting by state is requested
- **THEN** groups are ordered by a fixed state priority (Stable, Completing Rebalance, Preparing Rebalance, Empty, Dead, Unknown, then any other states), reversed when descending order is requested

#### Scenario: Sorting by lag
- **WHEN** sorting by consumer lag is requested
- **THEN** groups are ordered by their total lag, treating a group with no computable lag as zero

#### Scenario: Sorting by members or topics
- **WHEN** sorting by member count or topic count is requested
- **THEN** groups are ordered by the respective numeric value in the requested direction

### Requirement: Consumer Group List Export
The system SHALL allow exporting the full consumer group list (all pages, honoring the active search, state filter, and sort order) as a CSV download.

#### Scenario: CSV export
- **WHEN** a user triggers the CSV export from the consumer group list
- **THEN** a CSV file is produced containing one row per matching consumer group with at least group id, member count, topic count, lag, coordinator id, and state

### Requirement: Authorization Filtering of Consumer Groups
The system SHALL only include consumer groups the requesting user is authorized to view in list, lag, and per-topic results, and SHALL require the appropriate permission for view, delete, and offset-reset operations.

#### Scenario: Restricted visibility
- **WHEN** a user without visibility permission for certain groups requests the consumer group list
- **THEN** those groups are omitted from the results and from the page count basis

#### Scenario: Unauthorized mutation
- **WHEN** a user without the required permission attempts to delete a group or reset/delete its offsets
- **THEN** the operation is rejected and no change is made

### Requirement: Consumer Group Detail View
The system SHALL provide a detail view for a single consumer group showing its state, member count, count of assigned topics, count of assigned partitions, coordinator broker id, total lag, and a per-partition breakdown.

#### Scenario: Fetching group details
- **WHEN** a user opens the detail view for an existing group
- **THEN** the system returns the group's state (with explanatory tooltip), member count, assigned-topic count, assigned-partition count, coordinator broker id, total consumer lag, and the list of topic-partitions associated with the group

#### Scenario: Unknown group
- **WHEN** details are requested for a group id that does not exist on the cluster
- **THEN** the system responds with a not-found outcome

#### Scenario: Data-integration group cross-link
- **WHEN** the group id follows the naming convention of a managed data-integration connector and such a connector is configured
- **THEN** the detail view shows a link to that connector

### Requirement: Per-Partition Assignment Data
For each topic-partition associated with a consumer group, the detail view SHALL expose topic name, partition number, committed (current) offset, end offset, per-partition lag, and — when an active member is assigned to the partition — that member's consumer id and host.

#### Scenario: Partition with committed offset
- **WHEN** a partition has a committed offset for the group
- **THEN** the entry contains the committed offset, the partition's end offset, and the lag computed as end offset minus committed offset (zero when the end offset is unavailable)

#### Scenario: Assigned partition without committed offset
- **WHEN** an active member is assigned a partition for which the group has no committed offset
- **THEN** the partition still appears in the breakdown with the member's consumer id and host, without a committed offset

#### Scenario: Member attribution
- **WHEN** a partition is in an active member's assignment
- **THEN** the partition entry carries that member's consumer id and host

### Requirement: Detail View Topic Grouping
The detail view SHALL group the partition breakdown by topic, showing per-topic aggregate lag, and SHALL let the user expand a topic to see its partitions and filter topics by name.

#### Scenario: Topic rows
- **WHEN** the detail view renders
- **THEN** one row per distinct topic is shown with the topic name (linking to the topic view) and the topic's consumer lag, computed as the sum of its partitions' lags; a topic with no lag values shows a not-available indicator

#### Scenario: Expanding a topic
- **WHEN** a user expands a topic row
- **THEN** the topic's partitions are listed with partition number, consumer id, host, lag, current offset, and end offset, sortable by any of these columns

#### Scenario: Topic name filter
- **WHEN** a user enters a topic search string on the detail view
- **THEN** only topic rows whose name contains the string remain visible

### Requirement: Consumer Groups for a Topic
The system SHALL list all consumer groups related to a given topic, where a group is related if any active member is assigned a partition of the topic or the group has committed offsets for the topic.

#### Scenario: Listing groups for a topic
- **WHEN** the consumer groups of a topic are requested
- **THEN** every group with an active assignment on the topic or committed offsets for it is returned with group id, state, coordinator, partition assignor, and lag scoped to that topic

#### Scenario: Topic-scoped member count
- **WHEN** the member count is reported in a topic-scoped group entry
- **THEN** only members that have at least one partition of that topic in their assignment are counted

#### Scenario: Topic-scoped lag with no offsets
- **WHEN** a related group has no committed offsets for the topic
- **THEN** its topic-scoped lag is reported as undefined rather than zero

### Requirement: Delete Consumer Group
The system SHALL allow deleting a consumer group by id, guarded by a confirmation prompt in the user interface.

#### Scenario: Successful deletion
- **WHEN** a user confirms deletion of a consumer group
- **THEN** the group is deleted on the cluster and the user is returned to the consumer group list

#### Scenario: Read-only mode
- **WHEN** the cluster is configured as read-only
- **THEN** the delete action is not offered

### Requirement: Delete Committed Offsets for a Topic
The system SHALL allow deleting a consumer group's committed offsets for a specific topic, guarded by a confirmation prompt.

#### Scenario: Deleting a topic's offsets
- **WHEN** a user confirms deletion of a group's offsets for a topic from the detail view
- **THEN** all committed offsets of that group for that topic are removed while the group itself and offsets for other topics remain

### Requirement: Reset Offsets — Modes
The system SHALL support resetting a consumer group's committed offsets for one topic to the earliest offsets, the latest offsets, the offsets at a given timestamp, or explicitly specified per-partition offsets.

#### Scenario: Reset to earliest
- **WHEN** a reset to earliest is requested
- **THEN** the group's committed offset for each targeted partition is set to that partition's earliest available offset

#### Scenario: Reset to latest
- **WHEN** a reset to latest is requested
- **THEN** the group's committed offset for each targeted partition is set to that partition's end offset

#### Scenario: Reset to timestamp
- **WHEN** a reset to a timestamp is requested
- **THEN** each targeted partition's committed offset is set to the offset of the first record at or after that timestamp; for partitions with no such record the end offset is used instead

#### Scenario: Reset to explicit offsets
- **WHEN** a reset with explicit per-partition offsets is requested
- **THEN** each listed partition's committed offset is set to the given value, with a missing offset value treated as zero

### Requirement: Reset Offsets — Partition Scope
Offset resets SHALL apply either to an explicitly selected set of partitions or, when no partitions are specified, to all partitions of the topic.

#### Scenario: Whole-topic reset
- **WHEN** a reset request names a topic but no partitions
- **THEN** all partitions of the topic are reset

#### Scenario: Partial reset
- **WHEN** a reset request names specific partitions
- **THEN** only those partitions are reset and other partitions' committed offsets are untouched

### Requirement: Reset Offsets — Inactive-Group Precondition
The system SHALL only reset a group's offsets when the group exists and is inactive (Empty or Dead), rejecting the request otherwise.

#### Scenario: Active group
- **WHEN** a reset is attempted while the group is in any state other than Empty or Dead
- **THEN** the request fails with a validation error stating that offsets can only be reset for an inactive group and naming the current state

#### Scenario: Nonexistent group
- **WHEN** a reset is attempted for a group id not present in the cluster's group listing
- **THEN** the request fails with a not-found error

### Requirement: Reset Offsets — Input Validation
The system SHALL validate reset requests: a timestamp reset requires a timestamp, an explicit-offset reset requires a non-empty per-partition offset list, and an unrecognized reset type is rejected.

#### Scenario: Missing timestamp
- **WHEN** a timestamp reset is submitted without a timestamp value
- **THEN** the request is rejected with a validation error

#### Scenario: Missing partition offsets
- **WHEN** an explicit-offset reset is submitted without per-partition offsets
- **THEN** the request is rejected with a validation error

### Requirement: Reset Offsets — Bounds Clamping
For explicit-offset resets, the system SHALL clamp each requested offset into the partition's valid range before committing.

#### Scenario: Offset below earliest
- **WHEN** a requested offset is lower than the partition's earliest offset
- **THEN** the partition is reset to the earliest offset instead

#### Scenario: Offset above latest
- **WHEN** a requested offset is greater than the partition's latest offset
- **THEN** the partition is reset to the latest offset instead

### Requirement: Reset Offsets — User Interface Flow
The user interface SHALL provide a reset-offsets form reachable from the group detail view, offering topic selection from the group's associated topics, reset-type selection, partition multi-selection, and conditional inputs per reset type.

#### Scenario: Opening the form
- **WHEN** a user opens the reset form
- **THEN** it defaults to the earliest reset type, offers the group's distinct topics for selection, and lists only the selected topic's partitions in the partition selector

#### Scenario: Conditional inputs
- **WHEN** the timestamp reset type is selected with at least one partition
- **THEN** a date-time picker (honoring the user's timezone) is shown and a timestamp is required; when the explicit-offset type is selected, one required numeric offset input (minimum 0) is shown per selected partition

#### Scenario: Submission gating
- **WHEN** no partitions are selected
- **THEN** the submit action is disabled

#### Scenario: Changing the topic
- **WHEN** the user changes the selected topic
- **THEN** any previously selected partitions and entered offsets are cleared

#### Scenario: No assigned topics
- **WHEN** the group has no associated topics
- **THEN** the reset-offsets action is disabled in the detail view

#### Scenario: Successful reset
- **WHEN** the reset submission succeeds
- **THEN** the user is navigated back to the previous view

### Requirement: Lag Calculation
Consumer lag SHALL be computed per partition as end offset minus committed offset, aggregated by summation, and reported as undefined when no committed offsets exist.

#### Scenario: Per-partition lag
- **WHEN** a partition has both a committed offset and a known end offset
- **THEN** its lag is the end offset minus the committed offset

#### Scenario: Unknown end offset
- **WHEN** a partition's end offset is unavailable
- **THEN** that partition contributes zero to aggregated lag

#### Scenario: No committed offsets
- **WHEN** a group has no committed offsets at all in the considered scope
- **THEN** its lag is reported as undefined and displayed as a not-available indicator rather than zero

#### Scenario: Group total lag
- **WHEN** a group's total lag is computed
- **THEN** it is the sum of per-partition lags across all partitions with committed offsets

### Requirement: Bulk Lag Query for Monitoring
The system SHALL provide a bulk lag query that returns, for a set of named consumer groups, each group's total lag and per-topic lag, optionally including a per-topic-partition breakdown, sourced from a periodically refreshed cluster snapshot.

#### Scenario: Bulk lag retrieval
- **WHEN** lag is requested for a list of group names
- **THEN** the response maps each known, authorized group to its total lag and per-topic lag sums, along with the snapshot's completion timestamp

#### Scenario: Partition breakdown
- **WHEN** the request asks for partition-level detail
- **THEN** each group's response additionally includes lag per partition grouped by topic

#### Scenario: Conditional refresh
- **WHEN** the request carries the timestamp of the previously received snapshot and no newer snapshot exists
- **THEN** an empty result echoing that timestamp is returned instead of duplicate data

#### Scenario: Assigned partitions without offsets in snapshot lag
- **WHEN** an active member is assigned a partition with no committed offset
- **THEN** that partition is still included in the group's lag aggregation with a lag contribution of zero

### Requirement: Lag Trend Indicators and Auto-Refresh
The user interface SHALL support a user-selectable, persisted auto-refresh interval for lag data and, while auto-refresh is active, SHALL display a trend indicator next to each lag value comparing it with the previous reading.

#### Scenario: Rising lag
- **WHEN** auto-refresh is active and a lag value is greater than its previous reading
- **THEN** an upward trend indicator is displayed next to the value

#### Scenario: Falling lag
- **WHEN** auto-refresh is active and a lag value is lower than its previous reading
- **THEN** a downward trend indicator is displayed next to the value

#### Scenario: No baseline
- **WHEN** auto-refresh is off or no previous reading exists
- **THEN** the lag value is shown without a trend indicator

#### Scenario: Persisted refresh rate
- **WHEN** a user selects a refresh interval
- **THEN** the choice is persisted locally and reapplied on subsequent visits

### Requirement: Detail View Export
The user interface SHALL allow exporting the group detail's topic table as a CSV file.

#### Scenario: Exporting group details
- **WHEN** a user triggers the export on the group detail view
- **THEN** a CSV file of the currently displayed topic rows is downloaded

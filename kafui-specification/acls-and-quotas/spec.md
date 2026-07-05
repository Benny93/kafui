# Kafka ACL Management and Client Quotas

## ADDED Requirements

### Requirement: List ACL bindings
The system SHALL list all ACL bindings of a selected cluster, where each binding exposes the principal, resource type, resource name, name pattern type (literal or prefixed), operation, permission type (allow or deny), and host.

#### Scenario: Listing all ACLs
- **WHEN** a user requests the ACL list for a cluster without any filter
- **THEN** the system returns all ACL bindings defined on that cluster, each containing principal, resource type, resource name, name pattern type, operation, permission type, and host

#### Scenario: Stable ordering
- **WHEN** the ACL list is requested repeatedly without changes to the cluster
- **THEN** the bindings are returned in a deterministic, stable order across calls

### Requirement: Filter ACL list by resource pattern
The system SHALL support filtering the ACL list by resource type, resource name, and name pattern type, with each filter being optional.

#### Scenario: Filter by resource type
- **WHEN** a user requests the ACL list with a resource type filter (e.g. topic, consumer group, cluster, transactional id, delegation token, user)
- **THEN** only ACL bindings matching that resource type are returned

#### Scenario: Filter by resource name and pattern type
- **WHEN** a user supplies a resource name and/or a name pattern type (literal or prefixed) as filters
- **THEN** only ACL bindings whose resource pattern matches the supplied filter are returned

#### Scenario: Omitted filters default to match-any
- **WHEN** resource type or name pattern type filters are omitted
- **THEN** the corresponding filter dimension matches any value

### Requirement: Search ACLs by principal
The system SHALL support a free-text search parameter that narrows the ACL list by principal name, with an optional enhanced (n-gram / full-text) matching mode that can be toggled per request and defaulted by configuration.

#### Scenario: Principal search
- **WHEN** a user enters a search term in the ACL list view
- **THEN** only ACL bindings whose principal matches the search term are shown

#### Scenario: Enhanced search toggle
- **WHEN** the enhanced full-text search mode is enabled for the request (explicitly or by configured default)
- **THEN** principal matching uses the enhanced matching algorithm instead of plain substring matching

### Requirement: Create a custom ACL binding
The system SHALL allow creating a single ACL binding by specifying principal, host, resource type, resource name, name pattern type, operation, and permission type.

#### Scenario: Successful creation
- **WHEN** a user submits a valid ACL binding definition for a cluster
- **THEN** the binding is created on the cluster and a success response is returned

#### Scenario: Invalid principal format rejected
- **WHEN** the supplied principal is empty or not in the form `principalType:principalName`
- **THEN** the request is rejected with a validation error describing the expected format

### Requirement: Delete an ACL binding
The system SHALL allow deleting an existing ACL binding identified by its full definition (principal, host, resource pattern, operation, permission type).

#### Scenario: Successful deletion
- **WHEN** a user requests deletion of an existing ACL binding
- **THEN** the binding is removed from the cluster and a success response is returned

#### Scenario: Binding not found
- **WHEN** the ACL binding to delete does not exist
- **THEN** the system responds with a not-found error

#### Scenario: Deletion confirmation in the UI
- **WHEN** a user clicks the delete action on an ACL row in the UI
- **THEN** a confirmation prompt is shown, and the deletion is only performed after the user confirms

### Requirement: Consumer ACL convenience flow
The system SHALL provide a convenience operation that creates the ACL set required for a consuming application: allow READ and DESCRIBE on the specified topics and on the specified consumer groups, for a given principal and host. Topics and consumer groups MAY each be specified either as an explicit list of names (literal patterns) or as a name prefix (prefixed pattern).

#### Scenario: Consumer ACLs with explicit names
- **WHEN** a user submits a consumer ACL request with principal, host, a list of topics, and a list of consumer groups
- **THEN** allow-READ and allow-DESCRIBE bindings with literal patterns are created for each listed topic and each listed consumer group

#### Scenario: Consumer ACLs with prefixes
- **WHEN** a user submits a consumer ACL request using a topic prefix and/or a consumer group prefix
- **THEN** allow-READ and allow-DESCRIBE bindings with prefixed patterns are created for the given prefixes

#### Scenario: Exclusive choice per resource in the UI
- **WHEN** the user switches a resource selector between exact and prefixed mode in the consumer ACL form
- **THEN** the value of the deselected mode is cleared so only one addressing mode is submitted per resource kind

### Requirement: Producer ACL convenience flow
The system SHALL provide a convenience operation that creates the ACL set required for a producing application: allow WRITE, DESCRIBE, and CREATE on the specified topics (by names or prefix); allow WRITE and DESCRIBE on the specified transactional id (exact) or transactional id prefix; and, when idempotence is requested, allow IDEMPOTENT_WRITE on the cluster resource.

#### Scenario: Producer topic ACLs
- **WHEN** a user submits a producer ACL request with principal, host, and topics (as a name list or prefix)
- **THEN** allow bindings for WRITE, DESCRIBE, and CREATE are created for those topics with the corresponding literal or prefixed pattern type

#### Scenario: Transactional id ACLs
- **WHEN** the producer ACL request includes a transactional id or a transactional id prefix
- **THEN** allow bindings for WRITE and DESCRIBE are created on the transactional id resource with the corresponding pattern type

#### Scenario: Idempotent producer
- **WHEN** the producer ACL request has the idempotent flag set to true
- **THEN** an additional allow binding for IDEMPOTENT_WRITE is created on the cluster resource

### Requirement: Stream application ACL convenience flow
The system SHALL provide a convenience operation that creates the ACL set required for a stream-processing application: allow READ on the given input topics, allow WRITE on the given output topics, and allow ALL operations on consumer groups and topics whose names are prefixed with the application id, for a given principal and host.

#### Scenario: Stream app ACL creation
- **WHEN** a user submits a stream application ACL request with principal, host, application id, input topics, and output topics
- **THEN** the system creates allow-READ bindings for each input topic (literal), allow-WRITE bindings for each output topic (literal), and allow-ALL bindings with a prefixed pattern equal to the application id for both the consumer group and topic resource types

### Requirement: Export ACLs as CSV
The system SHALL export the (optionally filtered) ACL list of a cluster as plain-text CSV with the header `Principal,ResourceType,PatternType,ResourceName,Operation,PermissionType,Host` and one row per binding.

#### Scenario: CSV export
- **WHEN** a user requests the ACL list in CSV format
- **THEN** the response is plain text consisting of the fixed 7-column header followed by one comma-separated line per ACL binding, honoring the same filter and search parameters as the JSON list

### Requirement: Declarative ACL synchronization from CSV
The system SHALL accept a CSV document describing the complete desired ACL state of a cluster and reconcile the cluster to it: bindings present in the CSV but missing on the cluster are created, and bindings present on the cluster but absent from the CSV are deleted.

#### Scenario: Sync applies additions and deletions
- **WHEN** a user submits an ACL CSV for a cluster
- **THEN** the system computes the difference against the current ACL state, creates all missing bindings, deletes all extra bindings, and logs the sync plan

#### Scenario: Already in sync
- **WHEN** the submitted CSV exactly matches the current ACL state
- **THEN** no changes are made to the cluster

#### Scenario: Optional header and blank lines
- **WHEN** the submitted CSV starts with the standard header line (case-insensitive, spaces ignored) or contains blank lines
- **THEN** the header line and blank lines are skipped during parsing

#### Scenario: Malformed CSV rejected
- **WHEN** a CSV line does not have exactly 7 columns, contains a blank value, or contains an unrecognized resource type, pattern type, operation, or permission value
- **THEN** the sync is rejected with a validation error identifying the offending line (and column where applicable) and no changes are applied

### Requirement: ACL page in the UI
The system SHALL provide an access control list page per cluster showing all ACL bindings in a table with columns for principal, resource type, resource name with a pattern-type badge, host, operation, and permission, plus a delete action per row.

#### Scenario: Viewing the ACL table
- **WHEN** a user opens the ACL page of a cluster
- **THEN** a table of ACL bindings is displayed with the columns principal, resource, pattern (with a literal/prefixed badge), host, operation, and permission, where the permission is rendered as a colored badge distinguishing allow from deny

#### Scenario: Client-side column filtering
- **WHEN** the user applies column filters in the table
- **THEN** resource type and operation support multi-select filtering and pattern and host support text filtering

#### Scenario: Search resets pagination
- **WHEN** the user enters a new search term on the ACL page
- **THEN** the list navigates back to the first page

#### Scenario: Loading and error states
- **WHEN** the ACL list is loading or fails to load
- **THEN** the page shows a loading indicator, or an error view with the failure status and a retry action, respectively

### Requirement: ACL creation form in the UI
The system SHALL provide an ACL creation panel with a type selector offering four form variants: Custom ACL, For Consumers, For Producers, and For Kafka Stream Apps, each mapped to the corresponding creation operation.

#### Scenario: Selecting an ACL form type
- **WHEN** the user opens the create-ACL panel and chooses one of the four ACL types
- **THEN** the matching form is shown with fields appropriate to that flow (custom: principal, host, resource type, operation, permission, matching pattern with exact/prefixed choice; consumers: topics and consumer groups; producers: topics, transactional id, idempotent flag; stream apps: input topics, output topics, application id)

#### Scenario: Topic and group suggestions
- **WHEN** the user fills in topic or consumer group selectors in the convenience forms
- **THEN** the selectors offer the cluster's existing topics and consumer groups as multi-select options

#### Scenario: Successful submission closes the panel
- **WHEN** the user submits a valid form and the creation succeeds
- **THEN** the panel closes and the ACL list reflects the new bindings

#### Scenario: Read-only mode
- **WHEN** the cluster is configured as read-only or the user lacks ACL edit permission
- **THEN** the create-ACL action is disabled

### Requirement: ACL authorization and auditing
The system SHALL require a view-level ACL permission for listing/exporting ACLs and an edit-level ACL permission for creating, deleting, and synchronizing ACLs, and SHALL record each of these operations in the audit log.

#### Scenario: Unauthorized access denied
- **WHEN** a user without the required ACL permission invokes an ACL operation
- **THEN** the operation is rejected with an authorization error

#### Scenario: Operations audited
- **WHEN** an ACL list, create, delete, CSV export, or CSV sync operation completes (successfully or not)
- **THEN** an audit record including the operation name and cluster is produced

### Requirement: ACL capability detection
The system SHALL detect per cluster whether ACL viewing is supported (a cluster authorizer is enabled) and whether ACL editing is supported (the connected identity holds ALTER or ALL cluster authorization), and expose these as cluster feature flags.

#### Scenario: ACL view feature
- **WHEN** the cluster reports that security/authorization is enabled
- **THEN** the ACL-view feature flag is advertised for that cluster

#### Scenario: ACL edit feature
- **WHEN** ACL viewing is supported and the connected identity is authorized with ALTER or ALL on the cluster
- **THEN** the ACL-edit feature flag is advertised for that cluster

### Requirement: List client quotas
The system SHALL list all client quota entries of a cluster, where each entry identifies a quota entity by any combination of user, client id, and IP address, together with a map of quota property names to numeric values.

#### Scenario: Listing quotas
- **WHEN** a user requests the client quota list for a cluster
- **THEN** all quota entries are returned, each with its user, client id, and/or IP identifiers and its quota property/value map

#### Scenario: Deterministic ordering
- **WHEN** the quota list is returned
- **THEN** entries are sorted by user, then client id, then IP (absent values last), and each entry's quota properties are sorted by property name

### Requirement: Upsert client quotas
The system SHALL provide a single upsert operation that creates or replaces the quota property set for a quota entity identified by user, client id, and/or IP address; properties present on the entity but absent from the submitted set SHALL be cleared.

#### Scenario: Creating a new quota entry
- **WHEN** a non-empty quota property map is submitted for an entity that has no existing quotas
- **THEN** the quotas are applied and the response indicates that a new entry was created (status 201)

#### Scenario: Updating an existing quota entry
- **WHEN** a non-empty quota property map is submitted for an entity that already has quotas
- **THEN** the entity's quotas are replaced by the submitted set — new properties applied, changed values updated, and previously set properties missing from the submission cleared — and the response indicates an update (status 200)

#### Scenario: Missing entity identifier rejected
- **WHEN** an upsert is submitted with user, client id, and IP all absent
- **THEN** the request is rejected with a validation error stating that the quota entity id is not set

### Requirement: Delete client quotas
The system SHALL delete a quota entity's entry when an upsert is submitted with an empty or absent quota property map.

#### Scenario: Deleting via empty quota set
- **WHEN** an upsert for an existing quota entity is submitted with no quota properties
- **THEN** all quota properties of that entity are cleared and the response indicates deletion (status 204)

### Requirement: Client quota authorization and auditing
The system SHALL require a view-level client-quota permission for listing quotas and an edit-level client-quota permission for upserting/deleting them, and SHALL audit these operations including the submitted quota update parameters.

#### Scenario: Unauthorized quota change denied
- **WHEN** a user without the client-quota edit permission submits a quota upsert
- **THEN** the operation is rejected with an authorization error

### Requirement: Client quota capability detection
The system SHALL detect per cluster whether the broker supports client quota management and expose this as a cluster feature flag.

#### Scenario: Quota management feature advertised
- **WHEN** the connected cluster supports the client quota administration APIs
- **THEN** the client-quota-management feature flag is advertised for that cluster

# Schema Registry Integration

## ADDED Requirements

### Requirement: Per-Cluster Schema Registry Connection
The system SHALL allow each managed cluster to be configured with an optional schema registry endpoint, given as one or more URLs, and SHALL scope all schema operations to the registry of the addressed cluster.

#### Scenario: Registry configured for a cluster
- **WHEN** a cluster configuration includes a schema registry URL
- **THEN** all schema-related operations for that cluster are executed against that registry endpoint

#### Scenario: Multiple registry URLs with failover
- **WHEN** several registry URLs are configured for a cluster and a request to the currently active URL fails with a connection-level error
- **THEN** the system retries the request against the next configured URL, and returns an error indicating no live registry instances only when all URLs are unavailable

#### Scenario: Registry not configured
- **WHEN** a schema operation is requested for a cluster that has no schema registry configured
- **THEN** the operation is rejected with a validation error stating that no schema registry is set for that cluster

### Requirement: Schema Registry Authentication Configuration
The system SHALL support authenticating to a schema registry using either HTTP basic authentication (username and password) or OAuth 2.0 client-credentials (token URL, client id, client secret), configured per cluster, and SHALL reject invalid combinations at configuration time.

#### Scenario: Basic authentication
- **WHEN** a registry username and/or password is configured
- **THEN** all registry requests carry HTTP basic-authentication credentials

#### Scenario: OAuth authentication
- **WHEN** a token URL, client id, and client secret are all configured for the registry
- **THEN** the system obtains tokens via the client-credentials flow and attaches them to registry requests

#### Scenario: Partial OAuth configuration rejected
- **WHEN** only some of the OAuth parameters (token URL, client id, client secret) are configured
- **THEN** startup/validation fails with an error indicating the OAuth configuration is incomplete

#### Scenario: Conflicting authentication methods rejected
- **WHEN** both basic authentication and OAuth are configured for the same registry
- **THEN** startup/validation fails with an error indicating that only one authentication method may be configured

### Requirement: Schema Registry TLS Configuration
The system SHALL support TLS connections to the schema registry, including a configurable truststore (location, password, optional hostname/certificate verification toggle) and an optional client keystore (location, password) for mutual TLS, configured per cluster.

#### Scenario: Custom truststore
- **WHEN** a truststore is configured for the cluster
- **THEN** registry server certificates are validated against that truststore

#### Scenario: Mutual TLS
- **WHEN** a registry-specific client keystore is configured
- **THEN** the system presents the client certificate from that keystore when connecting to the registry

### Requirement: Registry Connectivity Validation
The system SHALL be able to validate a cluster's schema registry configuration by performing a connectivity check and reporting the result as part of cluster configuration validation.

#### Scenario: Validating registry settings
- **WHEN** a cluster configuration containing registry settings is validated
- **THEN** the response includes whether the registry is reachable with the given connection, authentication, and TLS settings

### Requirement: List Subjects with Pagination
The system SHALL provide a paginated listing of schema registry subjects for a cluster, returning for each subject its name, latest schema id, schema type, latest version, and effective compatibility level, together with the total page count.

#### Scenario: Default pagination
- **WHEN** the subject list is requested without explicit paging parameters
- **THEN** the first page is returned with a default page size of 25 subjects

#### Scenario: Explicit page selection
- **WHEN** a page number and page size are supplied
- **THEN** only the subjects belonging to that page are returned and the total page count reflects the full filtered subject set

#### Scenario: Empty result
- **WHEN** no subjects match the current filter/page
- **THEN** an empty list is returned and the user interface shows a "no schemas found" message

### Requirement: Subject Search
The system SHALL support filtering the subject list by a search term matched against subject names, with an optional full-text search mode that can be toggled per request and defaulted by configuration.

#### Scenario: Search by name
- **WHEN** a search term is provided
- **THEN** only subjects whose names match the term are returned and pagination is applied to the filtered set

#### Scenario: Full-text search toggle
- **WHEN** full-text search is enabled (by request parameter or configuration default)
- **THEN** the search term is evaluated using full-text matching semantics instead of plain substring matching

### Requirement: Subject List Sorting
The system SHALL support sorting the subject list by subject name, schema id, schema type, latest version, or compatibility level, in ascending or descending order.

#### Scenario: Sort by a column
- **WHEN** a sort column and sort order are supplied with the listing request
- **THEN** the returned page contains subjects ordered accordingly across the whole filtered set (server-side sorting)

#### Scenario: No sort specified
- **WHEN** no sort column is supplied
- **THEN** subjects are returned in the registry's natural listing order

### Requirement: Access-Controlled Subject Visibility
The system SHALL evaluate per-subject view permissions when listing subjects and SHALL exclude subjects the requesting user is not permitted to see; mutating operations SHALL require corresponding create, edit, or delete permissions, and changing the global compatibility level SHALL require a dedicated permission.

#### Scenario: Filtered listing
- **WHEN** a user without view permission on some subjects requests the subject list
- **THEN** those subjects are omitted from the results and from the page count

#### Scenario: Read-only cluster
- **WHEN** the cluster is configured as read-only
- **THEN** the user interface hides schema creation, editing, deletion, and compatibility-change controls

### Requirement: View Latest Schema of a Subject
The system SHALL return the latest version of a subject, including subject name, schema id, version number, schema type, raw schema text, and the effective compatibility level (the subject-level setting, or the global level when no subject-level setting exists).

#### Scenario: Latest version retrieval
- **WHEN** the latest schema of an existing subject is requested
- **THEN** the full schema record with its effective compatibility level is returned and displayed with the schema content rendered in a syntax-aware viewer

#### Scenario: Unknown subject
- **WHEN** a schema is requested for a subject that does not exist in the registry
- **THEN** a not-found error is returned

### Requirement: Associated Topic Link
The system SHALL attempt to associate a subject with a topic by stripping a configurable subject-name suffix (default "-value") and matching the remainder against known topic names, and SHALL expose the matched topic so the user interface can offer direct navigation to it.

#### Scenario: Subject matches a topic
- **WHEN** a subject name minus the configured suffix equals the name of an existing topic
- **THEN** the schema details include that topic name and the details page shows a "go to topic" navigation action

### Requirement: View All Versions of a Subject
The system SHALL return every stored version of a subject, each with its version number, schema id, schema type, and schema text.

#### Scenario: Version history display
- **WHEN** the version list of a subject is requested
- **THEN** all versions are returned, and the details page lists them with expandable rows revealing each version's schema content

### Requirement: Retrieve a Specific Schema Version
The system SHALL allow retrieving a single schema of a subject by its version number.

#### Scenario: Fetch by version
- **WHEN** an existing subject and version number are specified
- **THEN** that exact schema version is returned

#### Scenario: Missing version
- **WHEN** the specified version does not exist
- **THEN** a not-found error is returned

### Requirement: Compare Schema Versions
The system SHALL provide a side-by-side comparison view of any two versions of a subject, with independently selectable left and right versions and a rendered textual diff of the schema contents.

#### Scenario: Diff two versions
- **WHEN** the user selects a left version and a right version of a subject
- **THEN** both schema texts are shown side by side with differences highlighted, and JSON-based schemas are pretty-printed before comparison

#### Scenario: Deep-linkable comparison
- **WHEN** the comparison page is opened with left/right version identifiers in the URL query parameters
- **THEN** those versions are preselected, and changing a selection updates the URL accordingly

### Requirement: Supported Schema Types
The system SHALL support the schema types AVRO, JSON (JSON Schema), and PROTOBUF for registration, display, and editing.

#### Scenario: Type selection at creation
- **WHEN** a new schema is created
- **THEN** the user chooses one of AVRO, JSON, or PROTOBUF, with AVRO preselected as the default

#### Scenario: Type-aware rendering
- **WHEN** schema content is displayed or edited
- **THEN** it is rendered with syntax handling appropriate to its schema type, and AVRO/JSON content is pretty-printed while PROTOBUF content is shown verbatim

### Requirement: Register a New Schema Subject
The system SHALL allow registering a new subject by providing a subject name, schema text, and schema type, and SHALL return the resulting latest schema record on success.

#### Scenario: Successful registration
- **WHEN** a valid subject name, schema text, and schema type are submitted
- **THEN** the schema is registered in the registry and the newly stored latest version is returned; the user interface then navigates to the new subject's details page

#### Scenario: Subject name validation
- **WHEN** the subject name contains characters outside letters, digits, underscore, hyphen, and dot
- **THEN** the creation form rejects the input with a validation message before submission

#### Scenario: Required fields
- **WHEN** subject name, schema text, or schema type is missing
- **THEN** the form cannot be submitted and field-level error messages are shown

### Requirement: Register a New Version of an Existing Subject
The system SHALL allow submitting an updated schema text for an existing subject as a new version, presenting the current latest schema read-only alongside an editable copy, with the schema type fixed to the subject's existing type.

#### Scenario: New version submission
- **WHEN** the user edits the schema text of an existing subject and submits
- **THEN** the modified schema is registered as the subject's next version

#### Scenario: Unchanged form not submittable
- **WHEN** neither the schema text nor the compatibility level has been modified
- **THEN** the submit action remains disabled

#### Scenario: Syntax pre-validation
- **WHEN** the subject's type is AVRO or JSON and the edited text is not a valid JSON object
- **THEN** a syntax validation error is shown and submission is blocked

### Requirement: Registration Error Mapping
The system SHALL translate registry rejections during registration into distinct, user-meaningful errors.

#### Scenario: Incompatible schema rejected
- **WHEN** the registry rejects the submitted schema as incompatible with the subject's current schema (conflict)
- **THEN** the system reports a schema-compatibility error

#### Scenario: Invalid schema rejected
- **WHEN** the registry rejects the submitted schema as unprocessable/invalid
- **THEN** the system reports a validation error that includes the registry's error message

### Requirement: Delete Subject
The system SHALL allow deleting an entire subject, removing all of its versions from the registry, guarded by a confirmation prompt in the user interface.

#### Scenario: Delete whole subject
- **WHEN** the user confirms deletion of a subject
- **THEN** all versions of the subject are removed and the user is returned to the subject list

### Requirement: Delete a Specific Schema Version
The system SHALL allow deleting a single version of a subject by version number, and SHALL also support deleting only the latest version of a subject.

#### Scenario: Delete one version
- **WHEN** deletion of a specific version of a subject is requested
- **THEN** only that version is removed from the registry

#### Scenario: Delete latest version
- **WHEN** deletion of the latest version of a subject is requested
- **THEN** the most recent version is removed while earlier versions remain

### Requirement: Compatibility Level Values
The system SHALL support the compatibility levels BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE, and NONE for both global and per-subject settings.

#### Scenario: Level selection
- **WHEN** a compatibility level is set globally or for a subject
- **THEN** only one of the enumerated values is accepted

### Requirement: Global Compatibility Level Get and Set
The system SHALL expose the registry's global compatibility level for reading and allow updating it, with the user interface requiring an explicit confirmation before applying the change.

#### Scenario: Read global level
- **WHEN** the global compatibility level is requested
- **THEN** the current registry-wide level is returned; if it cannot be determined, a not-found result is returned

#### Scenario: Update global level with confirmation
- **WHEN** the user selects a new global level and confirms the warning that this may affect subject compatibility levels
- **THEN** the registry-wide compatibility level is updated

### Requirement: Per-Subject Compatibility Level Get and Set
The system SHALL resolve a subject's effective compatibility level (subject-specific setting, falling back to the global level when none is set) and SHALL allow setting a subject-specific compatibility level.

#### Scenario: Effective level fallback
- **WHEN** a subject has no subject-specific compatibility setting
- **THEN** the subject's displayed compatibility level equals the global level

#### Scenario: Update subject level
- **WHEN** the user changes the compatibility level in the subject edit form and submits
- **THEN** the subject-specific compatibility level is updated in the registry independently of any schema text change

### Requirement: Pre-Registration Compatibility Check
The system SHALL provide an operation that checks a candidate schema (schema text plus schema type) against the latest version of a subject under the effective compatibility rules, returning whether it is compatible without registering it.

#### Scenario: Compatible candidate
- **WHEN** a candidate schema compatible with the subject's latest version is checked
- **THEN** the response indicates compatibility is satisfied and no new version is created

#### Scenario: Incompatible candidate
- **WHEN** a candidate schema violating the effective compatibility rules is checked
- **THEN** the response indicates the schema is not compatible and no new version is created

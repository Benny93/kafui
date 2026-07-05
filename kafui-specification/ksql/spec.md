# KSQL / Streaming SQL Integration

## ADDED Requirements

### Requirement: Per-cluster ksqlDB server configuration
The system SHALL allow an optional ksqlDB server endpoint to be configured for each cluster, identified by a base URL.

#### Scenario: ksqlDB endpoint configured
- **WHEN** a cluster configuration includes a ksqlDB server URL
- **THEN** all streaming-SQL features for that cluster are directed at that endpoint

#### Scenario: ksqlDB endpoint absent
- **WHEN** a cluster configuration does not include a ksqlDB server URL
- **THEN** no ksqlDB client is created for the cluster and streaming-SQL features are not offered for it

### Requirement: ksqlDB feature availability flag
The system SHALL expose a per-cluster feature flag indicating whether ksqlDB integration is enabled, so that clients can show or hide streaming-SQL functionality.

#### Scenario: Feature reported when configured
- **WHEN** a cluster has a ksqlDB endpoint configured
- **THEN** the cluster's reported feature set includes the ksqlDB capability

### Requirement: Multiple ksqlDB endpoints with failover
The system SHALL accept a list of ksqlDB server URLs per cluster and SHALL fail over between them, retrying an alternative instance when a connection is refused.

#### Scenario: One instance unreachable
- **WHEN** a configured ksqlDB instance refuses connections and another configured instance is reachable
- **THEN** the request is served by a reachable instance

#### Scenario: No live instances
- **WHEN** all configured ksqlDB instances are unavailable
- **THEN** the operation fails with an error stating that no live ksqlDB instances are available

### Requirement: ksqlDB basic authentication configuration
The system SHALL support optional username/password credentials per cluster for the ksqlDB server and SHALL send them as HTTP basic authentication on every ksqlDB request. The password SHALL never be exposed in logs or diagnostic string output.

#### Scenario: Credentials configured
- **WHEN** ksqlDB credentials are configured for a cluster
- **THEN** requests to the ksqlDB server carry basic-authentication headers built from those credentials

### Requirement: ksqlDB TLS configuration
The system SHALL support TLS for the ksqlDB connection using the cluster-level truststore for server-certificate verification and an optional ksqlDB-specific keystore for client (mutual-TLS) authentication.

#### Scenario: Truststore configured
- **WHEN** a truststore is configured for the cluster
- **THEN** ksqlDB server certificates are validated against that truststore

#### Scenario: Keystore configured
- **WHEN** a ksqlDB keystore is configured
- **THEN** the client presents the keystore certificate when connecting to the ksqlDB server

### Requirement: ksqlDB configuration via settings UI
The system SHALL provide a cluster-configuration form section for ksqlDB with a required server URL field, optional authentication credentials, and optional keystore settings, and the section SHALL be collapsible/removable to disable the integration.

#### Scenario: Enabling ksqlDB in the form
- **WHEN** a user activates the ksqlDB configuration section
- **THEN** inputs for the server URL, authentication toggle, and keystore appear and the URL is mandatory

### Requirement: Response size limit
The system SHALL bound the size of buffered ksqlDB responses with a configurable maximum (default on the order of 20 MB).

#### Scenario: Default applied
- **WHEN** no explicit response buffer size is configured
- **THEN** the default maximum buffer size is applied to ksqlDB responses

### Requirement: List streams
The system SHALL provide a per-cluster operation that lists all ksqlDB streams, returning for each stream its name, backing topic, key format, and value format.

#### Scenario: Streams retrieved
- **WHEN** a client requests the stream list for a cluster with ksqlDB configured
- **THEN** the response contains one entry per stream with name, topic, key format, and value format

#### Scenario: Legacy format column
- **WHEN** the ksqlDB server reports only a single legacy "format" column instead of separate key/value formats
- **THEN** the legacy format value is returned as the stream's value format

#### Scenario: Unexpected listing result
- **WHEN** the ksqlDB server returns a result that is not a recognizable stream listing
- **THEN** the operation fails with an error indicating the stream list could not be retrieved

### Requirement: List tables
The system SHALL provide a per-cluster operation that lists all ksqlDB tables, returning for each table its name, backing topic, key format, value format, and whether it is windowed.

#### Scenario: Tables retrieved
- **WHEN** a client requests the table list for a cluster with ksqlDB configured
- **THEN** the response contains one entry per table with name, topic, key format, value format, and windowed flag

#### Scenario: Unexpected listing result
- **WHEN** the ksqlDB server returns a result that is not a recognizable table listing
- **THEN** the operation fails with an error indicating the table list could not be retrieved

### Requirement: ksqlDB overview page
The UI SHALL provide a per-cluster ksqlDB page showing summary counts of tables and streams and two tabbed listings (Tables and Streams) with sortable columns for name, topic, key format, value format, and windowed status; the Tables tab SHALL be shown by default.

#### Scenario: Overview loaded
- **WHEN** the user opens the ksqlDB page and both listings load successfully
- **THEN** the page shows the number of tables and the number of streams and renders the Tables tab

#### Scenario: Listing failure
- **WHEN** either listing fails to load
- **THEN** the page shows an error state with the failure message and a retry action that refetches both listings

#### Scenario: Loading state
- **WHEN** the listings are being fetched
- **THEN** a loading indicator is displayed

### Requirement: Two-phase statement execution
The system SHALL execute an arbitrary KSQL statement in two phases: a registration request that accepts the statement text plus optional streams properties and returns an execution handle (pipe identifier), followed by a separate streaming request that opens the result pipe for that handle.

#### Scenario: Statement registered
- **WHEN** a client submits a KSQL statement with optional streams properties for a cluster
- **THEN** the system returns a unique pipe identifier without yet executing the statement

#### Scenario: Pipe opened
- **WHEN** a client opens the result pipe with a valid pipe identifier
- **THEN** the registered statement is executed and its results are streamed back as server-sent events

### Requirement: Execution handle lifetime and single use
A registered execution handle SHALL be single-use and SHALL expire if not consumed within a short time window (one minute).

#### Scenario: Handle consumed
- **WHEN** a result pipe is opened for a handle
- **THEN** the handle is invalidated so it cannot be reused

#### Scenario: Unknown or expired handle
- **WHEN** a client opens a result pipe with an unknown, already-used, or expired handle
- **THEN** the request is rejected with a validation error stating no command is registered under that identifier

### Requirement: Statement validation before execution
The system SHALL parse the submitted KSQL text before execution and SHALL reject invalid input with a descriptive error result instead of forwarding it to the server.

#### Scenario: Unparsable input
- **WHEN** the submitted text cannot be parsed as KSQL
- **THEN** an error result is returned stating the statement is invalid or unsupported

#### Scenario: Multiple statements
- **WHEN** the submitted text contains more than one statement
- **THEN** an error result is returned stating that only a single statement is supported

#### Scenario: No statement
- **WHEN** the submitted text contains no valid statement
- **THEN** an error result is returned stating no valid statement was found

#### Scenario: Unsupported statement type
- **WHEN** the statement is of an unsupported type (topic printing, variable define/undefine)
- **THEN** an error result is returned stating the statement type is unsupported

### Requirement: Query routing by statement kind
The system SHALL route SELECT statements (push and pull queries) to the ksqlDB streaming query endpoint and all other statements to the ksqlDB statement endpoint.

#### Scenario: Select statement
- **WHEN** the parsed statement is a SELECT query
- **THEN** it is executed against the streaming query interface and rows are delivered incrementally

#### Scenario: Non-select statement
- **WHEN** the parsed statement is any other supported statement (DDL, DML, SHOW/LIST, DESCRIBE, EXPLAIN, connector operations, etc.)
- **THEN** it is executed against the statement interface and its full response is returned

### Requirement: Streams properties as execution parameters
The system SHALL accept an optional map of string key/value streams properties with each statement and pass them to the ksqlDB server to parameterize execution (e.g., auto.offset.reset).

#### Scenario: Properties supplied
- **WHEN** a statement is submitted with streams properties
- **THEN** those properties are forwarded with the execution request

#### Scenario: Properties omitted
- **WHEN** no streams properties are supplied
- **THEN** the statement executes with an empty property set

### Requirement: Tabular result model
All execution results SHALL be delivered as a sequence of named tables, each carrying a header title, an ordered list of column names, a list of value rows, and an error flag.

#### Scenario: Result event delivered
- **WHEN** the server produces any result for a statement
- **THEN** the client receives it as one or more table objects with header, column names, and row values

### Requirement: Live streaming of select query results
For SELECT queries the system SHALL first emit a schema table containing the column names and then emit one row table per received data row, continuing until the query completes or the client disconnects.

#### Scenario: Schema first
- **WHEN** a select query begins returning data
- **THEN** the first emitted table is a schema table whose column names are parsed from the query schema, correctly handling nested structured types and quoted identifiers

#### Scenario: Rows streamed
- **WHEN** subsequent data rows arrive from the server
- **THEN** each is emitted as a row table appended live to the displayed result set

#### Scenario: In-stream error
- **WHEN** the server emits an error message within the query stream
- **THEN** the stream terminates with an error carrying the server's error text

#### Scenario: Truncated response stream tolerated
- **WHEN** the server terminates a streaming response without properly closing the response structure (a known server-side defect)
- **THEN** the system treats the stream as complete rather than surfacing a parse error

### Requirement: Statement response interpretation
The system SHALL interpret typed statement responses (status, properties, queries, source/query/topic descriptions, stream/table/topic listings, execution plans, function listings and descriptions, connector operations) and render each as an appropriately titled table; unrecognized response types SHALL still be rendered generically rather than dropped.

#### Scenario: Recognized response type
- **WHEN** a statement response carries a known type marker
- **THEN** it is converted to a table with a human-readable title and the type-appropriate columns

#### Scenario: Unknown response type
- **WHEN** a statement response carries an unrecognized type marker
- **THEN** its fields are rendered dynamically into a generic result table

#### Scenario: Empty response body
- **WHEN** a statement (for example an INSERT) completes with an empty response body
- **THEN** a synthetic success table is emitted indicating the query succeeded

### Requirement: Execution error reporting
The system SHALL convert every execution failure into an error-flagged result table titled as an execution error, containing the server's error details when parseable and a plain message otherwise, so that errors travel through the same result channel as data.

#### Scenario: Server returns HTTP error with structured body
- **WHEN** the ksqlDB server responds with an error status and a parseable body
- **THEN** an error table is emitted whose columns and values reflect the server's error fields (such as type, error code, message, statement text, and affected entities)

#### Scenario: Server returns unparseable error
- **WHEN** the error response body cannot be parsed
- **THEN** an error table is emitted containing the HTTP status and raw body text

#### Scenario: Unexpected internal failure
- **WHEN** any unexpected exception occurs during execution
- **THEN** an error table is emitted containing the exception message

### Requirement: Query editor
The UI SHALL provide a query page with a syntax-highlighted SQL editor requiring non-empty input, a keyboard shortcut (Ctrl+Enter / Cmd+Enter) to execute, a control to clear the editor, and an editor that is read-only while a query is running.

#### Scenario: Execute via shortcut
- **WHEN** the user presses the execute shortcut with a non-empty statement
- **THEN** the statement is submitted exactly as pressing the Execute button would

#### Scenario: Empty statement
- **WHEN** the user attempts to execute with an empty statement
- **THEN** a validation error is shown and no request is sent

#### Scenario: Execution in progress
- **WHEN** a query is running
- **THEN** the Execute control is disabled and the editor does not accept edits

### Requirement: Stream properties editor
The query page SHALL provide a dynamic key/value list for entering streams properties, allowing rows to be added and removed, preventing a new row from being added while an existing row has an empty key, and omitting the properties entirely when none are filled in.

#### Scenario: Add property row
- **WHEN** the user adds a property and fills its key and value
- **THEN** the property is included in the execution request

#### Scenario: Remove last remaining row
- **WHEN** the user removes the only property row
- **THEN** the row is reset to empty rather than removed

#### Scenario: No properties entered
- **WHEN** the first property row's key is empty at submission
- **THEN** the request is sent without any streams properties

### Requirement: Live result rendering
The query page SHALL consume the result pipe as a live event stream, rendering a titled result table whose rows grow as row events arrive, replacing the display when a new schema or non-row result arrives, pretty-printing JSON-structured cell values, and showing a placeholder when a result has no columns.

#### Scenario: Rows accumulate
- **WHEN** row events arrive for an open query
- **THEN** each row is appended to the currently displayed table without losing previously displayed rows

#### Scenario: JSON cell values
- **WHEN** a cell value is a JSON object or array
- **THEN** it is rendered as formatted (indented) JSON text

#### Scenario: Error event surfaced
- **WHEN** an error-flagged table arrives on the stream
- **THEN** an error notification is shown with a title derived from the error type and code and a message including the statement text and affected entities when present

#### Scenario: Success event surfaced
- **WHEN** a success table for a non-returning statement arrives
- **THEN** a success notification is shown

### Requirement: Query termination
The UI SHALL show a persistent in-progress indicator with an abort control while consuming query results, and activating it (or leaving the page) SHALL close the result stream, which terminates the server-side query delivery.

#### Scenario: User aborts a running query
- **WHEN** the user activates the abort control while a push query is streaming
- **THEN** the event-stream connection is closed, result consumption stops, and the UI indicates the consumption was cancelled

#### Scenario: Stream closes
- **WHEN** the result stream ends for any reason (completion, abort, or error)
- **THEN** the in-progress state is cleared

### Requirement: Clear results
The query page SHALL provide a control to discard the currently displayed results, enabled only when results are present and no query is running.

#### Scenario: Results cleared
- **WHEN** the user activates the clear-results control
- **THEN** the displayed result table is removed and focus returns to the editor

### Requirement: Access control for ksqlDB operations
All ksqlDB operations (executing statements, opening result pipes, listing streams and tables) SHALL require a ksqlDB-execute permission on the target cluster, SHALL be recorded in the audit log, and the UI SHALL disable the query entry point for users lacking the permission.

#### Scenario: Unauthorized user
- **WHEN** a user without ksqlDB-execute permission invokes any ksqlDB operation
- **THEN** the operation is rejected by the authorization layer

#### Scenario: Authorized operation audited
- **WHEN** an authorized user registers or executes a statement or lists streams/tables
- **THEN** an audit record is written containing the operation name and its parameters

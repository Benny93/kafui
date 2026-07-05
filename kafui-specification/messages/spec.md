# Message Browsing, Producing, Serialization, and Data Masking

## ADDED Requirements

### Requirement: Message Browsing with Seek Modes
The system SHALL allow users to browse messages of a topic using one of the following seek modes: newest-first (from latest), oldest-first (from earliest), live tailing, from a specific offset (forward), to a specific offset (backward), from a specific timestamp (forward), and to a specific timestamp (backward). When no mode is specified, the system SHALL default to newest-first.

#### Scenario: Newest-first browsing
- **WHEN** a user requests messages with the "newest" seek mode
- **THEN** the system reads backward from the end of each selected partition and returns the most recent messages first

#### Scenario: Oldest-first browsing
- **WHEN** a user requests messages with the "oldest" seek mode
- **THEN** the system reads forward from the beginning of each selected partition and returns the earliest messages first

#### Scenario: Default mode when unspecified
- **WHEN** a browse request omits the seek mode
- **THEN** the system behaves as if the newest-first mode was requested

### Requirement: Offset-Based Seeking
The system SHALL support seeking to a user-supplied offset, reading forward from that offset or backward to that offset. An offset outside a partition's valid range SHALL be clamped to the partition's beginning or end offset.

#### Scenario: Seek from offset
- **WHEN** a user requests messages "from offset" N
- **THEN** consumption starts at offset N in each selected non-empty partition and proceeds toward newer messages

#### Scenario: Offset out of range
- **WHEN** the requested offset is greater than a partition's end offset or lower than its beginning offset
- **THEN** the effective offset for that partition is clamped to the end or beginning offset respectively

#### Scenario: Offset mode without offset value
- **WHEN** an offset-based mode is requested without an offset value
- **THEN** the request is rejected with a validation error

### Requirement: Timestamp-Based Seeking
The system SHALL support seeking by timestamp, resolving the earliest offset whose message timestamp is greater than or equal to the requested timestamp for each partition.

#### Scenario: Seek since timestamp
- **WHEN** a user requests messages "from timestamp" T
- **THEN** each partition is positioned at the first offset with a timestamp >= T and consumption proceeds forward

#### Scenario: Backward timestamp seek with no matching offset
- **WHEN** a user requests messages "to timestamp" T and all messages in a partition are older than T
- **THEN** that partition is read backward from its end offset

#### Scenario: Timestamp mode without timestamp value
- **WHEN** a timestamp-based mode is requested without a timestamp value
- **THEN** the request is rejected with a validation error

### Requirement: Partition Selection for Browsing
The system SHALL allow users to restrict browsing to a chosen subset of a topic's partitions; when no partitions are specified, all partitions SHALL be included. Empty partitions SHALL be skipped during polling.

#### Scenario: Browse selected partitions
- **WHEN** a user selects specific partitions for a browse request
- **THEN** only messages from those partitions are returned

#### Scenario: No partitions specified
- **WHEN** a browse request does not specify partitions
- **THEN** messages from all of the topic's partitions are considered

### Requirement: Page Size Limits
The system SHALL limit the number of messages returned per browse page, using a configurable default page size (100 if not configured) and a configurable maximum page size (500 if not configured). Requested limits that are missing, non-positive, or above the maximum SHALL be replaced with the default page size.

#### Scenario: Limit above maximum
- **WHEN** a user requests a page size greater than the configured maximum
- **THEN** the default page size is used instead

#### Scenario: No limit provided
- **WHEN** a browse request does not specify a page size
- **THEN** the default page size is used

### Requirement: Cursor-Based Pagination
The system SHALL support fetching subsequent pages via an opaque cursor. When a page completes and more matching data may remain, the completion event SHALL include a cursor identifier; submitting that cursor SHALL continue polling from the position after the last polled offsets, reusing the original query's serde selection, filters, and page size. Cursors SHALL be retained server-side in a bounded store (at least 10,000 entries) and requests with unknown or evicted cursors SHALL be rejected with a descriptive error.

#### Scenario: Next page available
- **WHEN** a browse page finishes and unread messages remain in the polled range
- **THEN** the stream's completion event contains a cursor identifier for the next page

#### Scenario: Fully polled range
- **WHEN** a browse page finishes and all targeted partitions were fully polled
- **THEN** the completion event contains no cursor

#### Scenario: Continue with cursor
- **WHEN** a user submits a browse request containing a previously returned cursor identifier
- **THEN** polling resumes from the stored position with the same filters, serdes, and page size as the original request

#### Scenario: Unknown cursor
- **WHEN** a request references a cursor identifier that is not present in the cursor store
- **THEN** the request is rejected with an error explaining that the cursor was not found or evicted

### Requirement: Streaming Browse Response with Event Types
The system SHALL deliver browse results as a stream of typed events: phase events (human-readable progress descriptions), message events (one deserialized message each), consuming-statistics events, and a final done event. Statistics SHALL include bytes consumed, messages consumed, elapsed polling time, and the count of filter application errors.

#### Scenario: Progress phases emitted
- **WHEN** a browse request is being processed
- **THEN** the client receives phase events describing the current stage (e.g. consumer creation, which partitions are being polled)

#### Scenario: Consuming statistics emitted
- **WHEN** each poll of the broker completes
- **THEN** the client receives a statistics event with cumulative bytes consumed, messages consumed, elapsed milliseconds, and filter error count

#### Scenario: Completion event
- **WHEN** polling finishes
- **THEN** the client receives a done event carrying final statistics and, if applicable, the next-page cursor

### Requirement: Message Ordering in Results
The system SHALL order messages within a page by timestamp (ascending for forward modes, descending for backward modes) while preserving offset order within each partition.

#### Scenario: Backward mode ordering
- **WHEN** messages from multiple partitions are returned in a backward mode
- **THEN** they are merged in descending timestamp order and messages of the same partition appear in descending offset order

### Requirement: Live Tailing
The system SHALL support a live tailing mode that positions consumers at the end of all selected partitions and continuously streams newly arriving messages until the client cancels. Tailing SHALL not apply a page limit and SHALL not produce pagination cursors, and the rate of messages delivered to the client SHALL be throttled (20 messages per second).

#### Scenario: New message arrives during tailing
- **WHEN** a client is tailing a topic and a new message is produced to it
- **THEN** the message is deserialized, filtered, and streamed to the client without a new request

#### Scenario: Client cancels tailing
- **WHEN** the client aborts the tailing connection
- **THEN** the server stops polling and releases the consumer

### Requirement: Polling Resource Controls
The system SHALL support a configurable poll timeout and an optional per-cluster polling throttle expressed as a maximum consumption byte rate applied across all browsing operations on that cluster.

#### Scenario: Throttle configured
- **WHEN** a cluster is configured with a polling throttle rate and polling exceeds that byte rate
- **THEN** subsequent polls are delayed so that average consumption stays within the configured rate

### Requirement: Per-Message Metadata
Each returned message SHALL include: partition, offset, timestamp, timestamp type (create time, log-append time, or none), key content, value content, headers as a name-to-string map, key size and value size in bytes (absent for null key/value), total headers size in bytes, the serde names used to render the key and the value, and any serde-provided additional deserialization properties.

#### Scenario: Message rendered with metadata
- **WHEN** a message is returned from a browse request
- **THEN** it carries partition, offset, timestamp and timestamp type, headers, key/value content, key/value/headers sizes, and the names of the serdes that produced the key and value renderings

#### Scenario: Null key or value
- **WHEN** a message has a null key or null value
- **THEN** the corresponding content and size fields are absent rather than rendered as text

### Requirement: String Containment Filtering
The system SHALL support filtering browsed messages by a search string that matches when the string is contained in the message key, the value, or any header name or header value. The search SHALL also match unicode-escaped representations of non-ASCII input.

#### Scenario: Match in header
- **WHEN** a search string is contained in a header name or header value of a message
- **THEN** the message is included in the results

#### Scenario: Non-ASCII search text
- **WHEN** the search string contains non-ASCII characters and message content stores them in escaped unicode form (upper- or lowercase hex)
- **THEN** the message still matches

### Requirement: Programmable Smart Filters
The system SHALL support user-defined filter expressions written in a safe expression language, evaluated per message against a record variable exposing: partition, offset, timestamp in epoch milliseconds, key (parsed as a JSON structure when possible, otherwise the raw string), the raw key text, value (parsed as JSON when possible), the raw value text, and headers as a string map. The expression SHALL evaluate to a boolean; non-boolean results SHALL be treated as an error.

#### Scenario: Filter on parsed JSON field
- **WHEN** a filter expression references a field of the parsed message value and a message's value contains that field with a matching content
- **THEN** the message is included in the results

#### Scenario: Non-JSON payload
- **WHEN** a message value cannot be parsed as JSON
- **THEN** the value is exposed to the expression as its raw string form

### Requirement: Smart Filter Registration
The system SHALL provide an operation to register a smart filter expression for a topic, returning a short filter identifier derived deterministically from the expression content; browse requests SHALL reference registered filters by identifier. Registered filters SHALL be kept in a bounded server-side store, and browse requests referencing an unknown identifier SHALL be rejected with a validation error.

#### Scenario: Register and use filter
- **WHEN** a user registers a filter expression and issues a browse request with the returned identifier
- **THEN** only messages satisfying the expression are returned

#### Scenario: Re-register same expression
- **WHEN** the same expression is registered again
- **THEN** the same identifier is returned

#### Scenario: Unknown filter id
- **WHEN** a browse request references a filter identifier that was never registered or was evicted
- **THEN** the request fails with a validation error naming the missing identifier

### Requirement: Smart Filter Test Execution
The system SHALL provide an operation to test a smart filter expression against user-supplied sample message data (key, value, headers, partition, offset, timestamp) without consuming from the topic, returning either the boolean result or a descriptive compilation or execution error.

#### Scenario: Valid expression tested
- **WHEN** a syntactically valid expression is tested against sample data
- **THEN** the response contains the boolean evaluation result

#### Scenario: Compilation error
- **WHEN** an expression fails to compile
- **THEN** the response contains an error message prefixed as a compilation error, and no evaluation occurs

#### Scenario: Runtime error
- **WHEN** a compiled expression throws during evaluation of the sample data
- **THEN** the response contains an error message prefixed as an execution error

### Requirement: Filter Error Tolerance During Polling
When applying a filter to a polled message raises an error, the system SHALL skip that message, increment the filter-error counter reported in the consuming statistics, and continue polling.

#### Scenario: Filter fails on one message
- **WHEN** a smart filter throws for a specific message during browsing
- **THEN** that message is omitted, the filter error count increases, and remaining messages are still processed

### Requirement: Producing Messages
The system SHALL allow users to produce a message to a topic, specifying optional key, optional value, optional headers (name-value string pairs), an optional explicit target partition, the serde to use for key and value serialization, and optional per-serde serialization properties. The topic MUST exist and an explicit partition MUST be a valid partition index, otherwise the request SHALL be rejected with a validation error.

#### Scenario: Produce with all fields
- **WHEN** a user sends a message with key, value, headers, and a valid partition
- **THEN** the key and value are serialized with the chosen serdes and the record is written to the requested partition with the given headers

#### Scenario: Invalid partition
- **WHEN** the requested partition index exceeds the topic's partition count
- **THEN** the request is rejected with a validation error

#### Scenario: Null key or value produced
- **WHEN** a user omits the key or the value
- **THEN** the record is produced with a null key or null value respectively

#### Scenario: Serialization properties passed
- **WHEN** the user supplies serde-specific properties (e.g. an explicit schema subject)
- **THEN** those properties are passed to the serializer and influence serialization

### Requirement: Produce Form Assistance
The message-producing user interface SHALL offer a partition selector listing the topic's partitions, key and value serde selectors pre-set to the suggested serde, dynamic input controls for the selected serde's declared parameters (reset when the serde changes), an option to keep form contents after sending, and the ability to pre-fill the form from an existing browsed message ("reproduce").

#### Scenario: Reproduce existing message
- **WHEN** a user chooses to reproduce a browsed message
- **THEN** the produce form opens pre-filled with that message's key, value, and headers

#### Scenario: Serde change resets parameters
- **WHEN** the user switches the key or value serde in the form
- **THEN** previously entered serde-specific parameter values are cleared

### Requirement: Serde Listing per Topic
The system SHALL provide an operation that lists, for a given topic and usage (serialize or deserialize), the applicable serdes for the key and for the value. Each entry SHALL include the serde name, an optional human-readable description, an optional schema description with additional schema properties, the serde's declared parameters (name, display name, allowed values), and a flag marking exactly one entry as the preferred (suggested) serde.

#### Scenario: List serdes for deserialization
- **WHEN** a user requests serdes for a topic with usage "deserialize"
- **THEN** the response lists, separately for key and value, all serdes able to deserialize that topic part, with the suggested serde marked as preferred

#### Scenario: Serde exposes schema
- **WHEN** a listed serde can describe the topic's schema
- **THEN** the entry contains the schema text and additional properties (such as subject, schema id, version, schema type)

### Requirement: Serde Auto-Detection and Selection
When no serde is explicitly chosen for browsing or producing, the system SHALL select one automatically by evaluating configured serdes in configuration order: a serde whose configured topic key/value name pattern matches the topic (or which was explicitly configured without a pattern) and which supports the operation is chosen; otherwise a configured cluster-level default key/value serde is used; otherwise the string serde is used. Explicitly chosen serdes that do not exist or cannot handle the topic/target SHALL cause a validation error.

#### Scenario: Pattern-based selection
- **WHEN** a serde is configured with a topic values pattern matching the browsed topic
- **THEN** that serde is automatically used for value deserialization when none is specified

#### Scenario: Fallthrough to string serde
- **WHEN** no configured serde pattern matches and no default serde is configured
- **THEN** the string serde is used

#### Scenario: Explicit serde cannot handle topic
- **WHEN** a user explicitly selects a serde that cannot deserialize (or serialize) the given topic part
- **THEN** the request is rejected with a validation error

#### Scenario: Unknown serde name
- **WHEN** a request names a serde that is not registered
- **THEN** the request is rejected with a validation error

### Requirement: Deserialization Fallback
If deserializing a message's key or value with the selected serde fails, the system SHALL fall back to a string-based fallback serde, return the fallback rendering, and mark the affected part with the fallback serde's name so the user interface can flag it.

#### Scenario: Value fails to deserialize
- **WHEN** the selected value serde throws while deserializing a record
- **THEN** the value is rendered by the fallback serde and the message's value serde name identifies the fallback

#### Scenario: Fallback indicator in UI
- **WHEN** a browsed message was rendered with the fallback serde
- **THEN** the message list displays a warning indicator on the affected key or value

### Requirement: Built-In Serde Formats
The system SHALL provide built-in serdes including at least: String (configurable character encoding), Int32, Int64, UInt32, UInt64, Base64, Hex (configurable delimiter and letter case), binary UUID, embedded Avro, MessagePack, Protobuf based on user-provided schema/descriptor files, schema-less raw Protobuf decoding, and a schema-registry-backed serde. The String, Base64, and Hex serdes SHALL support both serialization and deserialization of any topic.

#### Scenario: Hex round trip
- **WHEN** a user produces a message with the Hex serde and browses it with the Hex serde
- **THEN** the browsed rendering is the hexadecimal representation of the produced bytes

#### Scenario: Configurable string encoding
- **WHEN** the String serde is configured with a non-default character encoding
- **THEN** keys and values are encoded and decoded using that encoding

### Requirement: Schema Registry Serde
The schema-registry serde SHALL resolve subjects using the topic name (key/value subject naming), support Avro, Protobuf, and JSON Schema types, render deserialized payloads as JSON, report the subject, schema id, version, and schema type as additional properties, and refuse to deserialize payloads lacking the magic-byte/schema-id prefix. For serialization it SHALL validate and encode input against the registered schema and SHALL support an explicit subject override parameter.

#### Scenario: Deserialize registry-encoded message
- **WHEN** a message encoded with a registered schema is browsed with the schema-registry serde
- **THEN** the payload is rendered as JSON and the schema id and version used are exposed as additional deserialization properties

#### Scenario: Payload without schema prefix
- **WHEN** a payload without the magic byte and schema id prefix is deserialized with the schema-registry serde
- **THEN** deserialization fails with an explanatory error (triggering fallback rendering during browsing)

#### Scenario: Serialize with subject parameter
- **WHEN** a user produces a message with the schema-registry serde and an explicit subject property
- **THEN** the message is serialized against that subject's schema

### Requirement: Internal Topic Serdes
The system SHALL provide read-only serdes automatically bound by topic-name pattern to well-known internal topics, including the consumer-offsets topic and replication-tool internal topics (heartbeats, checkpoints, offset syncs), decoding their binary formats into readable JSON.

#### Scenario: Browse consumer offsets topic
- **WHEN** a user browses the internal consumer-offsets topic
- **THEN** its binary records are decoded into a readable representation without manual serde configuration

### Requirement: Serde Configuration
The system SHALL allow declaring serdes per cluster in configuration with: a unique name, optional properties (overriding built-in defaults), optional topic key and topic value name patterns controlling auto-selection, and optional cluster-level default key and value serde names. Built-in serdes not explicitly configured SHALL be auto-configured when possible and remain selectable without being bound to any topic. Duplicate serde names SHALL be rejected at startup.

#### Scenario: Built-in serde with overridden properties
- **WHEN** configuration declares a built-in serde name with custom properties
- **THEN** the serde is instantiated with those properties instead of auto-configuration

#### Scenario: Duplicate names
- **WHEN** two serde entries in a cluster's configuration share the same name
- **THEN** startup fails with a validation error

### Requirement: Pluggable Custom Serde Extension Point
The system SHALL define a public serde extension API allowing third parties to package custom serialization/deserialization logic as external artifacts. A custom serde SHALL be declared in configuration with a name, an implementation class name, and an artifact file path; the system SHALL load it in an isolated, child-first class loading scope, instantiate it via a no-argument constructor, call a configuration hook with layered property resolvers (serde-level, cluster-level, global), and call a close hook at shutdown. The API SHALL let implementations report applicability per topic and target (key/value), provide serializers and deserializers, expose an optional description, optional schema description, and optional declared parameters, and return deserialization results as text tagged as plain string or JSON plus arbitrary additional properties.

#### Scenario: Custom serde loaded from configuration
- **WHEN** configuration declares a custom serde with class name and artifact path
- **THEN** the implementation is loaded in isolation, configured with its properties, and appears in the serde lists for matching topics

#### Scenario: Missing class name or path
- **WHEN** a custom serde declaration lacks the class name or artifact path
- **THEN** startup fails with a validation error

#### Scenario: Custom serde used for browsing
- **WHEN** a user selects a custom serde for browsing
- **THEN** its deserializer output (text, string/JSON type, additional properties) is rendered like any built-in serde's output

### Requirement: Data Masking Rules
The system SHALL support per-cluster masking rules applied to message keys and values at display time (never altering stored data). Each rule SHALL define: an action of MASK, REPLACE, or REMOVE; a topic-keys name pattern and/or a topic-values name pattern (at least one required); and a field selector given either as an explicit field-name list or a field-name regex (not both; when neither is given, all fields are affected).

#### Scenario: Rule applies by topic pattern
- **WHEN** a message is browsed from a topic whose name matches a rule's topic-values pattern
- **THEN** the rule is applied to the rendered message value

#### Scenario: Both fields list and pattern configured
- **WHEN** a masking rule specifies both a fields list and a fields-name pattern
- **THEN** configuration is rejected with a validation error

#### Scenario: Neither topic pattern set
- **WHEN** a masking rule specifies neither a topic-keys nor a topic-values pattern
- **THEN** configuration is rejected with a validation error

### Requirement: Masking Policy Semantics
For JSON-parseable content, masking SHALL be applied recursively: selected fields (at any nesting depth, including inside arrays) are transformed while other fields are traversed unchanged. MASK SHALL replace characters class-wise using a four-element replacement set for uppercase letters, lowercase letters, digits, and other characters (defaults "X","x","n","-"), preserving space and line separators. REPLACE SHALL substitute affected scalar values with a replacement string (default "***DATA_MASKED***"). REMOVE SHALL delete the affected fields. For content that is not a JSON object or array, the first applicable rule SHALL be applied to the whole string (REMOVE yielding the literal "null"). Multiple applicable rules SHALL be applied in configuration order to JSON content.

#### Scenario: Character-class masking
- **WHEN** a MASK rule affects a field containing "Ab 1?"
- **THEN** the displayed value is "Xx n-" (with default replacement characters)

#### Scenario: Replace nested field
- **WHEN** a REPLACE rule targets a field that holds an object
- **THEN** every scalar within that object is replaced with the replacement string

#### Scenario: Remove field
- **WHEN** a REMOVE rule targets a field present in a JSON value
- **THEN** the field is absent from the displayed value

#### Scenario: Non-JSON string masked
- **WHEN** a MASK rule applies to a value that is not valid JSON
- **THEN** the entire string is masked character-by-character

### Requirement: Message List Presentation
The message browsing interface SHALL display messages in a table with expandable rows showing offset, partition, timestamp (absolute, or relative with absolute shown on hover), truncated key and value previews, and an expanded detail view with tabs for key, value, and headers plus metadata (timestamp type, sizes, serde names, serde-provided extra properties).

#### Scenario: Expand a message
- **WHEN** a user clicks a message row
- **THEN** an expanded view opens with key, value, and headers tabs and the message's metadata

#### Scenario: Empty result
- **WHEN** a browse completes with no matching messages
- **THEN** the table shows an explicit "no messages found" state

### Requirement: Field Preview in Message List
The message list SHALL allow users to define preview projections for the key and value columns as named JSON-path expressions, so that only the selected fields are rendered in the table cells.

#### Scenario: Preview field configured
- **WHEN** a user defines a preview projection with a field label and JSON path for the value column
- **THEN** each row's value cell renders the label and the extracted field instead of the full payload

### Requirement: Saved Smart Filters in UI
The browsing interface SHALL let users create, name, edit, delete, and re-apply smart filter expressions, with an option to persist saved filters locally across sessions.

#### Scenario: Apply a saved filter
- **WHEN** a user selects a previously saved smart filter
- **THEN** it is registered with the server and applied to subsequent browse requests

### Requirement: Per-Message Export
The system SHALL allow exporting a single browsed message by copying it to the clipboard or saving it as a file, containing the message's value, offset, key, partition, headers, and timestamp in a structured text format.

#### Scenario: Save message as file
- **WHEN** a user chooses "save as file" on a browsed message
- **THEN** a file containing the message's key, value, offset, partition, headers, and timestamp is downloaded

### Requirement: CSV Export Formatting
Where the system offers CSV export of tabular data, the CSV output SHALL use configurable formatting: field separator, quote character, quote strategy, and line delimiter; designated internal columns SHALL be excluded, a header row SHALL be emitted from the exported objects' field names, and array values SHALL be joined with commas.

#### Scenario: Custom separator configured
- **WHEN** the CSV field separator is configured to a non-default character
- **THEN** exported CSV rows use that character between fields

### Requirement: Access Control for Message Operations
Browsing messages SHALL require a message-read permission on the topic and producing SHALL require a message-produce permission; operations SHALL be denied without them and SHALL be recorded in the audit trail.

#### Scenario: Produce without permission
- **WHEN** a user lacking message-produce permission attempts to send a message
- **THEN** the request is denied

# Application Configuration, Dynamic Config Wizard, and Application Info

## ADDED Requirements

### Requirement: Declarative cluster configuration
The application SHALL accept a declarative configuration (via configuration files, environment variables, or equivalent externalized settings) defining a list of managed clusters, where each cluster entry includes a name, bootstrap server addresses, and optional settings for TLS truststore (location, password, certificate-verification flag), broker authentication properties, a schema registry endpoint with its own basic or OAuth client-credentials authentication and keystore, a schema registry subject-name suffix for value topics, a streaming-SQL server endpoint with its own basic authentication and keystore, a list of connect-service clusters each with name, address, basic authentication, keystore, and consumer-name pattern, pluggable serialization configurations, metrics collection settings, data-masking rules, audit settings, a read-only flag, polling throttle rate, and free-form custom client properties (plus separate consumer-specific and producer-specific property maps).

#### Scenario: Multiple clusters configured
- **WHEN** the configuration declares several cluster entries with distinct names
- **THEN** the application connects to and exposes each configured cluster independently under its declared name

#### Scenario: Nested custom properties are flattened
- **WHEN** custom client properties are declared as nested maps
- **THEN** the application flattens them into dot-separated keys before passing them to the underlying clients

### Requirement: Cluster configuration validation at startup
The application SHALL validate the declared cluster configuration when it is loaded and refuse invalid configurations.

#### Scenario: Single unnamed cluster gets a default name
- **WHEN** exactly one cluster is configured without a name
- **THEN** the application assigns it a default name and starts normally

#### Scenario: Multiple clusters require unique names
- **WHEN** more than one cluster is configured and any cluster lacks a name or two clusters share the same name
- **THEN** the application rejects the configuration with an error stating that names must be provided and unique

#### Scenario: Mandatory cluster fields
- **WHEN** a cluster entry omits its name or bootstrap servers value (blank)
- **THEN** the configuration is rejected with a validation error identifying the missing field

### Requirement: Read-only cluster mode
The application SHALL support a per-cluster read-only flag; for read-only clusters all state-changing operations SHALL be rejected while read operations remain available.

#### Scenario: Mutating request against a read-only cluster
- **WHEN** a request using a state-changing method targets a cluster whose configuration marks it read-only
- **THEN** the request is rejected with an error indicating the cluster is in read-only mode

#### Scenario: Read requests unaffected
- **WHEN** a read-only cluster receives a read or preflight request
- **THEN** the request is processed normally

#### Scenario: Whitelisted analytical operations
- **WHEN** a state-changing request targets designated safe analytical endpoints (e.g. message-filter test execution or topic analysis) on a read-only cluster
- **THEN** the request is allowed despite read-only mode

### Requirement: Dynamic configuration feature toggle
The application SHALL support an opt-in dynamic-configuration mode, controlled by a dedicated setting, that allows the application configuration to be viewed and modified at runtime; when disabled, all runtime configuration-change operations SHALL be rejected with an explanatory error.

#### Scenario: Dynamic config disabled
- **WHEN** dynamic configuration is not enabled and a client attempts to read the editable config, persist a new config, or upload a config-related file
- **THEN** the operation fails with a validation error explaining that dynamic configuration is disabled and how to enable it

#### Scenario: Dynamic config file loaded at startup
- **WHEN** dynamic configuration is enabled and a dynamic configuration file exists at the configured (or default) path
- **THEN** the file's properties are loaded at startup with highest precedence over other configuration sources

#### Scenario: Missing dynamic config file tolerated
- **WHEN** dynamic configuration is enabled but the dynamic configuration file does not exist or is unreadable
- **THEN** the application logs a warning and starts with the remaining configuration sources

### Requirement: View current application configuration
The application SHALL expose an endpoint returning the current effective application configuration as a structured document containing the cluster definitions, access-control rules, authentication settings (type and identity-provider client settings), and HTTP-client tuning settings. Access SHALL require a configuration-view permission.

#### Scenario: Authorized configuration read
- **WHEN** a user with configuration-view permission requests the current configuration and dynamic configuration is enabled
- **THEN** the current values of cluster, access-control, authentication, and HTTP-client settings are returned as a structured document

#### Scenario: Unauthorized configuration read
- **WHEN** a user without configuration-view permission requests the current configuration
- **THEN** the request is denied

### Requirement: Apply configuration and restart
The application SHALL expose an endpoint that accepts a complete candidate configuration document, validates and persists it to the dynamic configuration file, and then restarts the application in-place so the new configuration takes effect. Access SHALL require a configuration-edit permission.

#### Scenario: Successful apply and restart
- **WHEN** an authorized user submits a new configuration document
- **THEN** the configuration is validated, defaults are applied, the document is serialized and written to the dynamic configuration file (creating parent directories as needed, overwriting any prior file), and the application restarts itself using the new configuration

#### Scenario: Invalid persist target
- **WHEN** the configured dynamic configuration path is a directory, or an existing file at that path is not readable and writable
- **THEN** the persist operation fails with a validation error and no restart occurs

#### Scenario: Invalid submitted configuration
- **WHEN** the submitted document fails structural validation (e.g. duplicate cluster names, missing mandatory fields, malformed HTTP-client buffer size)
- **THEN** the operation fails with a validation error and the running configuration is unchanged

### Requirement: Candidate configuration connectivity validation
The application SHALL expose an endpoint that accepts a candidate configuration and, without persisting it, actively tests connectivity to each declared cluster and its dependent services, returning a per-cluster validation report keyed by cluster name with independent results for the cluster connection, schema registry, each connect service (keyed by name), the streaming-SQL server, and the metrics store; each result indicates success or failure with an error message on failure. Access SHALL require a configuration-edit permission.

#### Scenario: Cluster connection test
- **WHEN** a candidate cluster configuration is validated
- **THEN** the application creates a short-lived administrative client with reduced retry and timeout settings, attempts a metadata listing, and reports success or a connection error for that cluster

#### Scenario: Dependent service tests
- **WHEN** the candidate cluster declares a schema registry, connect services, a streaming-SQL server, or a metrics store
- **THEN** each declared service is probed with a lightweight request (e.g. read global compatibility, list plugins, execute a trivial statement, run a trivial query) and its individual success/failure result is included in the report; undeclared services are omitted

#### Scenario: Truststore verified before connecting
- **WHEN** the candidate cluster configuration includes a truststore location and password
- **THEN** the truststore file is opened and loaded first, and a load failure is reported as a cluster validation error without attempting a connection

#### Scenario: Empty candidate
- **WHEN** the candidate configuration contains no clusters
- **THEN** an empty validation report is returned

### Requirement: Upload configuration-related files
The application SHALL expose an endpoint accepting a multipart file upload (e.g. truststores, keystores) that stores the file in a configurable uploads directory and returns the stored file's absolute path for referencing in configuration. Access SHALL require a configuration-edit permission and dynamic configuration to be enabled.

#### Scenario: Successful upload
- **WHEN** an authorized user uploads a file
- **THEN** the uploads directory is created if absent, the file is stored under its original name suffixed with a timestamp to avoid collisions, and the response contains the stored file's path

#### Scenario: Upload failure
- **WHEN** the uploads directory cannot be created or the file transfer fails
- **THEN** the operation fails with a file-upload error identifying the target path

### Requirement: Secret redaction in displayed configuration
The application SHALL mask the values of secret-like configuration keys with a fixed placeholder wherever component configuration is displayed, matching keys case-insensitively against a configurable pattern list that by default covers known client security keys plus general patterns (password, secret, token, key, credentials, passphrase, registry basic-auth user info, cloud access/secret/session keys, database connection URIs).

#### Scenario: Secret value masked
- **WHEN** a configuration entry whose key matches a sanitization pattern is displayed
- **THEN** its value is replaced by the fixed placeholder

#### Scenario: External secret references preserved
- **WHEN** a matching entry's value is an externalized config-provider reference of the form "${provider:[path:]key}"
- **THEN** the reference is returned unmasked so re-submitting the configuration does not destroy the indirection

#### Scenario: Sanitizer disabled or customized
- **WHEN** the sanitizer is disabled via its setting, or a custom pattern list is provided
- **THEN** no masking occurs, or the custom patterns fully replace the defaults, respectively

### Requirement: Application info endpoint
The application SHALL expose an unauthenticated-accessible info endpoint returning build information (version, short commit identifier, build time, whether the running version is the latest release), the list of enabled optional features (including a dynamic-configuration feature flag), and details of the latest published release (version tag, publication date, release-page link) when release checking is enabled.

#### Scenario: Info retrieval
- **WHEN** a client requests the application info
- **THEN** it receives build metadata, the enabled-features list, and latest-release details if available

#### Scenario: Dynamic config feature advertised
- **WHEN** dynamic configuration is enabled
- **THEN** the enabled-features list contains the dynamic-configuration feature identifier; otherwise it is absent

### Requirement: Latest-release check
The application SHALL periodically (at startup and on a configurable interval, default hourly) query the project's public release feed for the latest release, with a configurable timeout, and SHALL allow this check to be disabled via a setting; failures SHALL degrade gracefully to empty release info.

#### Scenario: Release check disabled
- **WHEN** the release-check setting is off
- **THEN** no external release queries are made, a warning about unsupported old versions is logged, and info responses omit latest-release data

#### Scenario: Release feed unavailable
- **WHEN** the release feed request fails or times out
- **THEN** the error is swallowed and the application continues with empty release information

### Requirement: Authentication settings endpoint
The application SHALL expose an endpoint describing how users authenticate, returning the configured authentication type (disabled, form login, LDAP, or OAuth) and, for OAuth, the list of available identity providers with display name and authorization-initiation path. Detailed authentication and RBAC behavior is governed by configuration but specified elsewhere.

#### Scenario: Authentication disabled
- **WHEN** no authentication type is configured
- **THEN** the endpoint reports authentication as disabled with an empty provider list

#### Scenario: OAuth providers listed
- **WHEN** OAuth authentication is configured with authorization-code providers
- **THEN** each such provider is listed with its client display name and the relative URI that starts its authorization flow

### Requirement: Health and monitoring endpoints
The application SHALL expose a health endpoint reporting application liveness and a metrics endpoint exposing internal application metrics in a standard scrape format, alongside the info endpoint.

#### Scenario: Health probe
- **WHEN** a monitoring system requests the health endpoint
- **THEN** an up/down health status is returned suitable for liveness/readiness probes

### Requirement: Cluster management UI (setup wizard)
When the dynamic-configuration feature is enabled, the web UI SHALL offer a cluster configuration form for registering a new cluster and for editing or deleting an existing one, organized into sections: cluster basics (name, read-only flag, bootstrap servers as host/port pairs, truststore upload), broker authentication (choice of security protocol and named authentication methods with method-specific fields, generating the appropriate client properties), schema registry, serialization plugins, connect services, streaming-SQL server, metrics, and data-masking rules; each dependent-service section supports its own credentials and keystore upload.

#### Scenario: Wizard entry visibility
- **WHEN** the application info reports the dynamic-configuration feature enabled
- **THEN** the UI's cluster dashboard shows per-cluster configuration actions and an option to configure a new cluster; otherwise these controls are hidden

#### Scenario: Validate before submit
- **WHEN** the user activates the form's validate action
- **THEN** the form contents are checked client-side, then sent to the connectivity-validation endpoint, and a success notice or per-service error feedback is shown without saving anything

#### Scenario: Submit new or edited cluster
- **WHEN** the user submits the form
- **THEN** the UI merges the cluster into the current configuration (appending a new cluster or replacing the one being edited by its original name), applies the configuration via the restart endpoint, and navigates back to the dashboard; failures show an error notification

#### Scenario: Delete cluster
- **WHEN** the user confirms deletion of an existing cluster
- **THEN** the UI removes that cluster from the configuration, applies the result via the restart endpoint, and navigates back to the dashboard

#### Scenario: Certificate file upload from the form
- **WHEN** the user selects a truststore/keystore file in the form
- **THEN** the file is uploaded through the config-file upload endpoint and the returned server-side path is placed into the corresponding location field, with an option to reset it

#### Scenario: Editing a cluster with unsupported custom authentication
- **WHEN** an existing cluster's configuration contains hand-written authentication properties that do not map onto the form's known methods
- **THEN** the form presents them in a raw custom-authentication section instead of the guided authentication section

### Requirement: Broker authentication methods in the wizard
The cluster form's authentication section SHALL support choosing a security protocol (SASL over TLS or plaintext) and an authentication method from at least: Kerberos-based, OAuth-bearer, plain username/password, salted-challenge (256 and 512 variants), delegation tokens, LDAP-backed plain, and cloud-provider IAM variants; the UI SHALL translate the chosen method and its fields into the corresponding standard client login-module configuration string.

#### Scenario: Method translated to client properties
- **WHEN** the user selects an authentication method and fills its fields
- **THEN** the submitted cluster properties contain the correct login-module class, its required options rendered as a single configuration line, and the matching mechanism/protocol settings

### Requirement: Configuration file auto-reload
The application SHALL support an optional setting that watches its loaded configuration files for changes and hot-reloads eligible settings (at minimum role-based access-control rules) from changed files without a restart.

#### Scenario: Watched file modified
- **WHEN** auto-reload is enabled and a watched configuration file changes on disk
- **THEN** the file is re-parsed, its properties replace the previous ones with highest precedence, and reloadable settings take effect immediately; parse errors are logged without disrupting the running application

### Requirement: HTTP-client tuning configuration
The application SHALL support configuring its outbound HTTP client behavior, including maximum in-memory response buffer size (expressed as a data-size string) and response timeout, and SHALL reject malformed buffer-size values at validation time.

#### Scenario: Malformed buffer size
- **WHEN** the buffer-size setting cannot be parsed as a data size
- **THEN** configuration validation fails with an error naming the offending setting

### Requirement: Web server base path and static UI serving
The application SHALL support deployment under a configurable base path prefix (for reverse-proxy setups), serving both API and UI beneath that prefix, and SHALL serve the single-page UI shell for UI routes and the application root.

#### Scenario: UI route requested
- **WHEN** a browser requests the root path or any UI route
- **THEN** the single-page application shell document is served

#### Scenario: Base path configured
- **WHEN** a base path is configured
- **THEN** all endpoints and UI assets are addressable only under that prefix

### Requirement: Cross-origin request support
The application SHALL answer cross-origin requests by echoing the request origin, allowing credentials, permitting the standard read/write methods, allowing the content-type header, and answering preflight requests directly with success.

#### Scenario: Preflight request
- **WHEN** a preflight (OPTIONS) request is received
- **THEN** the response carries the cross-origin allowance headers and a success status without invoking the underlying endpoint

### Requirement: Optional AI-assistant integration endpoint
The application SHALL optionally (behind a dedicated enable setting, off by default) expose a machine-consumable assistant integration endpoint implementing a standard model-context tool protocol over a server-sent-events transport, advertising the application's read and management operations as callable tools generated from the same operation set as the human-facing API.

#### Scenario: Assistant integration disabled by default
- **WHEN** the assistant-integration setting is not enabled
- **THEN** no assistant protocol endpoints are exposed

#### Scenario: Assistant integration enabled
- **WHEN** the setting is enabled
- **THEN** an assistant client can connect over the event-stream endpoint, list the available tools, and invoke them subject to the application's normal authorization rules

### Requirement: Interactive API documentation toggle
The application SHALL optionally serve its API specification and an interactive API explorer, controlled by a setting that is disabled by default.

#### Scenario: Explorer disabled
- **WHEN** the API-explorer setting is off (default)
- **THEN** the interactive documentation endpoints are not available

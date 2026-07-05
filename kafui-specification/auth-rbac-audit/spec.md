# Authentication, Role-Based Access Control, and Audit Logging

## ADDED Requirements

### Requirement: Configurable Authentication Method
The application SHALL support a single configurable authentication method selected via configuration, with the possible values: disabled, form login with configured users, LDAP/Active Directory, and OAuth2/OIDC. The default method SHALL be disabled.

#### Scenario: Authentication method selected by configuration
- **WHEN** the application starts with an authentication method configured
- **THEN** only that authentication method is active for the whole application

#### Scenario: No authentication method configured
- **WHEN** the application starts without an explicit authentication method setting
- **THEN** authentication is disabled and all access is unrestricted

#### Scenario: Deprecated authentication flag rejected
- **WHEN** the application starts and a legacy boolean "authentication enabled" configuration property is present
- **THEN** the application logs an error explaining the replacement setting and its valid values, and terminates with a non-zero exit code

### Requirement: Disabled Authentication Mode
When authentication is disabled, the application SHALL allow every request without credentials and SHALL log a warning at startup that access is unrestricted.

#### Scenario: Anonymous access with disabled authentication
- **WHEN** authentication is disabled and any endpoint is requested without credentials
- **THEN** the request is served without any authentication challenge

### Requirement: Form Login With Configured Users
When form-login authentication is enabled, the application SHALL authenticate users by username and password submitted to a login endpoint, validated against statically configured user credentials, and SHALL establish a server-side session on success.

#### Scenario: Successful form login
- **WHEN** a user submits a valid username and password as form-encoded fields to the login endpoint
- **THEN** a session is established and the response completes without forcing a browser redirect, allowing the client application to navigate itself

#### Scenario: Failed form login
- **WHEN** a user submits invalid credentials to the login endpoint
- **THEN** no session is established and the response indicates an authentication error that the login page can detect and display

#### Scenario: Unauthenticated request to protected endpoint
- **WHEN** authentication is enabled and a protected endpoint is requested without a valid session
- **THEN** the request is rejected and the user is directed to the login page

### Requirement: LDAP and Active Directory Authentication
When LDAP authentication is enabled, the application SHALL authenticate users against a configured directory server by binding with the submitted credentials, supporting configurable server URLs, base DN or user DN pattern, an administrative bind account, and user search base/filter. An Active Directory mode SHALL be supported that requires a configured domain and authenticates against the directory using domain semantics.

#### Scenario: LDAP bind authentication
- **WHEN** LDAP authentication is enabled and a user submits directory credentials through the login form
- **THEN** the application binds to the directory with those credentials and establishes a session on success

#### Scenario: User located via search filter
- **WHEN** a user search base and search filter are configured
- **THEN** the user entry is located via that search before authentication instead of a static DN pattern

#### Scenario: Active Directory mode without domain
- **WHEN** Active Directory mode is enabled but no domain is configured
- **THEN** the application fails to start with an error stating the domain is required

#### Scenario: Secure LDAP connections
- **WHEN** the configured directory URL uses the secure LDAP scheme
- **THEN** connections are made over TLS using the application's trust configuration

#### Scenario: Directory group resolution with RBAC enabled
- **WHEN** RBAC is enabled and a user authenticates via LDAP or Active Directory
- **THEN** the user's directory group memberships are resolved (via a configurable group search base and filter for LDAP, or native group attributes for Active Directory) and used for role assignment

### Requirement: OAuth2/OIDC Authentication With Multiple Providers
When OAuth2 authentication is enabled, the application SHALL support any number of concurrently configured OAuth2/OIDC providers, each with its own client id, client secret, client name, scopes, redirect URI, and either an issuer URI (OIDC discovery) or explicit authorization/token/user-info/key-set endpoints, plus an optional user-name attribute and arbitrary provider-specific custom parameters.

#### Scenario: Login via chosen provider
- **WHEN** a user starts the authorization flow for one of the configured providers
- **THEN** the application performs the authorization-code flow with that provider and establishes a session for the returned identity

#### Scenario: No providers configured
- **WHEN** OAuth2 authentication is enabled but no providers are configured
- **THEN** the application fails to start with a descriptive error

#### Scenario: Provider validation
- **WHEN** a configured provider lacks a client id or a provider type name
- **THEN** the application rejects the configuration at startup

#### Scenario: Outbound provider calls honor system proxy settings
- **WHEN** system-level HTTP/HTTPS proxy settings are present
- **THEN** all outbound calls to identity providers (token, user-info, key-set retrieval) are routed through the configured proxy

### Requirement: Bearer-Token Resource Server Mode
When OAuth2 authentication is enabled, the application SHALL optionally accept bearer tokens on API requests, validated either as signed JWTs against a configured key-set URI or as opaque tokens via a configured introspection endpoint with client credentials.

#### Scenario: JWT bearer token accepted
- **WHEN** resource-server JWT validation is configured and a request carries a valid signed bearer token
- **THEN** the request is authenticated without an interactive login

#### Scenario: Opaque token introspection
- **WHEN** opaque-token introspection is configured and a request carries a bearer token
- **THEN** the token is validated by calling the introspection endpoint authenticated with the configured client id and secret

### Requirement: Public Authentication Settings Endpoint
The application SHALL expose an unauthenticated endpoint that returns the active authentication method and, for OAuth2, the list of configured providers, each with its display name and the relative URI that starts its authorization flow.

#### Scenario: Settings fetched by the login page
- **WHEN** the authentication settings endpoint is requested without credentials
- **THEN** it returns the authentication type (disabled, form login, LDAP, or OAuth2) and the OAuth2 provider list (empty unless OAuth2 is active)

#### Scenario: Only interactive providers listed
- **WHEN** OAuth2 providers are configured with grant types other than authorization code
- **THEN** those providers are omitted from the returned provider list

### Requirement: Login Page
The application SHALL serve a login page that renders according to the active authentication method: a username/password form for form-login and LDAP modes, and one card per configured provider for OAuth2 mode.

#### Scenario: Credential form shown
- **WHEN** the authentication method is form login or LDAP and the login page loads
- **THEN** a sign-in form with username and password fields and a submit button is shown, with the submit button showing a progress state while authentication is in flight

#### Scenario: Provider cards shown
- **WHEN** the authentication method is OAuth2 and the login page loads
- **THEN** one card per provider is shown with the provider's name, a recognizable icon for well-known providers (with a generic fallback), and a button that navigates to that provider's authorization URI

#### Scenario: Invalid credentials feedback
- **WHEN** a form login attempt fails, or the browser returns to the login page with an error indicator after a failed OAuth2 flow
- **THEN** an error message such as "username or password entered incorrectly" or "invalid credentials" is displayed

### Requirement: Anonymous-Accessible Paths
The application SHALL exempt a fixed set of paths from authentication: static assets (index page, scripts, styles, images, fonts, favicon, manifest), health and monitoring endpoints, API documentation, the login and logout endpoints, the OAuth2 flow endpoints, the authentication settings endpoint, and the authorization/user-info endpoint.

#### Scenario: Static assets served without login
- **WHEN** a static asset or health endpoint is requested without credentials while authentication is enabled
- **THEN** the resource is served without an authentication challenge

### Requirement: Logout
The application SHALL provide a logout endpoint that terminates the user's session. For form-login and LDAP modes it SHALL accept logout via a simple navigation request and redirect to the login page with a logged-out indicator. For OAuth2 it SHALL perform provider-appropriate logout: OIDC RP-initiated logout by default, with pluggable provider-specific handlers.

#### Scenario: Session-based logout
- **WHEN** an authenticated form-login or LDAP user navigates to the logout endpoint
- **THEN** the session is invalidated and the browser is redirected to the login page marked as logged out

#### Scenario: OIDC logout
- **WHEN** an OAuth2-authenticated user logs out and no provider-specific handler applies
- **THEN** the application performs RP-initiated logout against the provider that authenticated the user

#### Scenario: Provider-specific logout
- **WHEN** the authenticating provider requires a custom logout flow (e.g. a hosted logout URL configured via a custom parameter)
- **THEN** the application invalidates the session and redirects the browser to that logout URL with the client id and the application's base URL as the post-logout return address, and fails configuration validation if the required logout URL parameter is absent

### Requirement: Current-User Authorization Endpoint
The application SHALL expose an endpoint returning whether RBAC is enabled and, for an authenticated user, the user's name and a flattened list of effective permissions, each entry containing the applicable cluster names, resource type, optional resource-name pattern, and permitted actions.

#### Scenario: Authenticated user info
- **WHEN** an authenticated user requests the authorization endpoint
- **THEN** the response contains the RBAC-enabled flag, the username, and the union of permissions from all configured roles whose names appear in the user's resolved group set

#### Scenario: Default-role fallback in user info
- **WHEN** an authenticated user matches no configured role and a default role is configured
- **THEN** the returned permissions are the default role's permissions applied to all known clusters

#### Scenario: Anonymous or RBAC-less response
- **WHEN** the endpoint is requested with no authenticated principal
- **THEN** the response contains the RBAC-enabled flag without user information

### Requirement: RBAC Activation
RBAC SHALL be considered enabled if and only if at least one role or a default role is defined in configuration. With RBAC disabled, every access check SHALL succeed.

#### Scenario: No roles configured
- **WHEN** no roles and no default role are configured
- **THEN** RBAC is disabled and all authenticated (or anonymous, if auth is disabled) users may perform every operation

#### Scenario: Roles configured without matching authentication
- **WHEN** roles are configured but the active authentication setup provides no identity providers to resolve them
- **THEN** the application logs an error warning that authentication may fail

### Requirement: Role Definition Model
Each role SHALL be defined in configuration with a unique name, a non-empty list of cluster names it applies to, a non-empty list of subjects that grant membership, and a list of permissions. Configuration SHALL be validated at startup, rejecting roles with empty clusters or subjects and permissions lacking a resource or actions.

#### Scenario: Invalid role rejected
- **WHEN** the application starts with a role missing clusters, subjects, a permission resource, or permission actions
- **THEN** startup fails with a validation error

### Requirement: Role Subjects
Each subject SHALL specify an identity-provider kind (generic OAuth, well-known OAuth providers such as a search company, a code-hosting service, and a cloud identity pool, LDAP, or Active Directory), a subject type meaningful for that provider, and a value. Values SHALL match case-insensitively by default, or as a regular expression when the subject is flagged as regex.

#### Scenario: Regex subject matching
- **WHEN** a subject is flagged as a regular expression
- **THEN** an identity attribute grants membership if it fully matches the pattern

#### Scenario: Literal subject matching
- **WHEN** a subject is not flagged as regex
- **THEN** an identity attribute grants membership only if it equals the value ignoring case

### Requirement: Role Resolution From Provider Identity
At login the application SHALL resolve the user's role memberships from provider-supplied identity data, using per-provider extraction rules, and attach the resulting role-name set to the session principal.

#### Scenario: Google-style provider extraction
- **WHEN** a user authenticates via the search-company provider
- **THEN** roles are matched by subject type "user" against the email claim and by subject type "domain" against the hosted-domain claim

#### Scenario: Code-hosting provider extraction
- **WHEN** a user authenticates via the code-hosting provider
- **THEN** roles are matched by subject type "user" against the login name, by subject type "organization" against the user's organization memberships fetched from the provider API with the access token, and by subject type "team" against "organization/team" strings fetched from the provider API

#### Scenario: Cloud identity-pool provider extraction
- **WHEN** a user authenticates via the cloud identity-pool provider
- **THEN** roles are matched by subject type "user" against the principal name and by subject type "group" against the token's groups claim, whose claim name defaults to the provider's standard groups claim and is overridable via a "roles-field" custom parameter

#### Scenario: Generic OAuth/OIDC extraction
- **WHEN** a user authenticates via a generic OAuth2/OIDC provider
- **THEN** roles are matched by subject type "user" against the principal name and by subject type "role" against the values of a claim named by the provider's "roles-field" custom parameter, accepting the claim as a string list or comma-separated string, and matching no group roles when the parameter is absent

#### Scenario: Provider matching by explicit type
- **WHEN** a provider's custom parameters declare a "type" naming one of the well-known extraction schemes
- **THEN** that extraction scheme is applied regardless of the provider's own name

#### Scenario: LDAP extraction
- **WHEN** a user authenticates via LDAP with RBAC enabled
- **THEN** roles with LDAP subjects are matched by subject type "user" against the username and by subject type "group" against the user's resolved directory groups, including nested group membership

#### Scenario: Active Directory extraction
- **WHEN** a user authenticates via Active Directory with RBAC enabled
- **THEN** roles with Active Directory subjects are matched by subject type "user" against the username and by subject type "group" against the user's directory groups

### Requirement: Permission Model
Each permission SHALL specify a resource type, an optional resource-name regular expression, and a non-empty list of action names. The resource types SHALL be: application configuration, cluster configuration, topic, consumer group, schema, connect cluster, connector, SQL query engine, ACL, audit, and client quotas. An "all" action keyword SHALL grant every action of the resource type.

#### Scenario: Name-pattern matching
- **WHEN** a permission has a resource-name pattern and an access check names a specific resource
- **THEN** the permission applies only if the name fully matches the pattern

#### Scenario: Unnamed resource checks
- **WHEN** an access check targets a resource type without a specific name (e.g. application configuration)
- **THEN** only permissions without a name pattern apply

#### Scenario: Unknown action rejected
- **WHEN** configuration names an action not applicable to the permission's resource type
- **THEN** startup fails with an error identifying the invalid action and resource

### Requirement: Per-Resource Action Vocabulary
The application SHALL support the following actions per resource type, each classified as read-only or altering: topics — view, create, edit, delete, read messages, produce messages, delete messages, view analysis, run analysis; consumer groups — view, delete, reset offsets; schemas — view, create, edit, delete, modify global compatibility; connect clusters — view, create, edit, operate, delete, reset offsets (with "restart" accepted as an alias of operate); connectors — view, create, edit, operate, delete, reset offsets; SQL query engine — execute; ACLs — view, edit; audit — view; client quotas — view, edit; cluster configuration — view, edit; application configuration — view, edit.

#### Scenario: Implied view permission
- **WHEN** a permission grants a non-view action on a resource type that has a view action
- **THEN** the corresponding view action (and, transitively, any further implied actions) is automatically granted

#### Scenario: Connector actions imply connect-cluster visibility
- **WHEN** a permission grants a connector action
- **THEN** the corresponding connect-cluster action needed to reach the connector is implied

### Requirement: Access Enforcement On Every Operation
Every API operation SHALL declare the cluster (when applicable) and the set of resource/action pairs it requires, and the application SHALL reject the operation with an access-denied error (HTTP 403) unless the user's effective permissions cover all requested actions on every requested resource.

#### Scenario: Denied operation
- **WHEN** RBAC is enabled and a user invokes an operation for which any requested resource/action pair is not covered
- **THEN** the operation is rejected with an access-denied error before any effect occurs

#### Scenario: Cluster-level gate
- **WHEN** an operation targets a cluster not covered by any of the user's roles and no default role is configured
- **THEN** the operation is denied regardless of resource-level permissions

#### Scenario: Multi-resource operations
- **WHEN** an operation touches several resources (e.g. copying between topics)
- **THEN** the required actions on every involved resource must all be granted

#### Scenario: Fallback permission paths
- **WHEN** an operation defines an alternative permission that also satisfies it
- **THEN** access is granted if either the primary or the fallback resource/action set is covered

### Requirement: Default Role
The application SHALL support an optional default role, consisting only of permissions, that applies to users who match no configured role, across all clusters.

#### Scenario: Default role applied
- **WHEN** RBAC is enabled, a default role is configured, and a user's resolved groups match no role for the target cluster
- **THEN** the default role's permissions are used for the access decision

### Requirement: Permission-Based Filtering of Listings
List endpoints SHALL only return resources the user is permitted to view: clusters are limited to those covered by the user's roles (or the default role), and topics, consumer groups, schemas, connect clusters, and connectors are filtered by view permission against their names.

#### Scenario: Topic list filtered
- **WHEN** a user lists topics with RBAC enabled
- **THEN** only topics whose names match one of the user's topic-view permissions are returned

#### Scenario: Connect cluster visible through connector permission
- **WHEN** a user has view permission on a connector within a connect cluster but not on the connect cluster itself
- **THEN** the connect cluster still appears in listings so its permitted connectors are reachable

#### Scenario: Connector-scoped permissions
- **WHEN** connector permissions are evaluated
- **THEN** the resource name is the composite "connect-cluster-name/connector-name", and both parts may be patterned

### Requirement: UI Permission Awareness
The web client SHALL fetch the current user's permission set once and use it to hide or disable any navigation entries, buttons, menu items, and forms whose underlying action the user is not permitted to perform, evaluating cluster, resource type, action, and resource-name pattern client-side.

#### Scenario: Forbidden action hidden or disabled
- **WHEN** RBAC is enabled and the user lacks a given action on a named resource
- **THEN** the corresponding UI control is hidden or rendered disabled

#### Scenario: Create actions checked cluster-wide
- **WHEN** the UI decides whether to offer a "create" action for a resource type
- **THEN** it requires only that some permission in the cluster grants create for that type, ignoring name patterns since the name is not yet known

#### Scenario: Unnamed resource types in the UI
- **WHEN** the UI checks permissions for resource types that have no per-name granularity (SQL engine, cluster configuration, application configuration, ACLs, audit)
- **THEN** the name-pattern check is skipped and only the action membership is evaluated

#### Scenario: RBAC disabled in the UI
- **WHEN** the RBAC-enabled flag from the server is false
- **THEN** all UI actions are treated as permitted

### Requirement: Per-Cluster Audit Configuration
The application SHALL support per-cluster audit configuration with independent switches for writing audit records to a message topic and/or to the application console log, and a level selecting whether all audited operations or only altering operations are recorded (defaulting to altering only). Auditing SHALL be inactive for a cluster when both outputs are disabled or no audit configuration is present.

#### Scenario: Console-only auditing
- **WHEN** console audit is enabled and topic audit is disabled for a cluster
- **THEN** audit records for that cluster are written only to a dedicated console log stream

#### Scenario: Alter-only level
- **WHEN** the audit level is "alter only" and an operation requests only read-type actions
- **THEN** no audit record is written for that operation

#### Scenario: All-operations level
- **WHEN** the audit level is "all"
- **THEN** read-only operations are audited as well

### Requirement: Audit Topic Provisioning
When topic auditing is enabled, the application SHALL write audit records to a configurable topic (with a documented default name) and SHALL create it at startup if absent, using a default of one partition and a 90-day deletion retention, both overridable along with arbitrary topic properties.

#### Scenario: Missing topic auto-created
- **WHEN** topic audit is enabled and the audit topic does not exist
- **THEN** the application creates it with the configured (or default) partition count and topic properties

#### Scenario: Topic initialization failure with strict mode
- **WHEN** audit topic creation or verification fails and the configuration requires the audit topic
- **THEN** application startup fails with the underlying error

#### Scenario: Topic initialization failure without strict mode
- **WHEN** audit topic initialization fails and the audit topic is not required
- **THEN** topic auditing is disabled for that cluster with a prominent error log, falling back to console-only auditing if console audit is enabled

#### Scenario: Compressed audit production
- **WHEN** audit records are produced to the topic
- **THEN** they are sent compressed, and production failures are logged without failing the audited operation

### Requirement: Audit Record Content
Each audit record SHALL be a JSON document containing: an ISO-8601 UTC timestamp, the acting username (or "Unknown" when no principal is resolvable), the cluster name (absent for application-level operations), the list of accessed resources (each with resource type, optional resource identifier, an "alter" flag, and the requested action names), the operation name, optional structured operation parameters, and the operation result — success, or a failure classified as access denied, validation error, execution error, or unrecognized error.

#### Scenario: Successful operation audited
- **WHEN** an audited operation completes successfully
- **THEN** a record with a success result is written after completion

#### Scenario: Failed operation audited
- **WHEN** an audited operation terminates with an error, including an authorization rejection
- **THEN** a record is written with a failure result classifying the error, so denied attempts are visible in the audit trail

### Requirement: Audit Coverage and Routing
All state-inspecting and state-changing API operations that declare an access context SHALL be audited, covering topics, messages, consumer groups, schemas, connect clusters and connectors, SQL engine execution, ACLs, client quotas, cluster configuration, and application configuration. Records for cluster-scoped operations SHALL be routed to that cluster's audit outputs; application-level operations without a cluster SHALL be written to the console audit log.

#### Scenario: Cluster operation routed to cluster writer
- **WHEN** an audited operation targets a cluster with auditing configured
- **THEN** its record is written to that cluster's configured outputs

#### Scenario: Application-level operation
- **WHEN** an audited operation has no target cluster (e.g. editing application configuration)
- **THEN** its record is written to the console audit log

#### Scenario: Audit failures are non-fatal
- **WHEN** writing an audit record throws an error
- **THEN** the error is logged and the user-facing operation outcome is unaffected

### Requirement: Audit Topic Access Protection
Reading the contents or details of a cluster's active audit topic SHALL require the audit "view" permission instead of ordinary topic permissions.

#### Scenario: Viewing the audit topic
- **WHEN** a user requests messages or details of the topic currently used for audit output with topic-writing enabled
- **THEN** access is granted only if the user holds the audit view permission for that cluster

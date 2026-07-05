# Web UI Shell and Cross-Cutting UX Behaviors

## ADDED Requirements

### Requirement: Single-Page Application Served by Backend
The system SHALL deliver the management console as a single-page web application whose static assets and entry document are served by the backend service, so that no separate web server is required.

#### Scenario: UI route request returns application shell
- **WHEN** a browser requests the application root, or any UI route path, directly from the backend
- **THEN** the backend responds with the application entry document so the client-side application can boot and render the requested view

#### Scenario: Direct navigation to a deep UI route
- **WHEN** a user opens or reloads a deep UI route URL (e.g. a specific resource detail page) in a fresh browser session
- **THEN** the backend serves the application shell and the client renders the view corresponding to that URL rather than returning a not-found error

### Requirement: Configurable Base Path
The system SHALL support being hosted under an operator-configurable context/base path (e.g. behind a reverse proxy at a sub-path), and all asset references, API calls, and internal links SHALL resolve correctly relative to that path.

#### Scenario: Entry document rewritten with base path
- **WHEN** the application is configured with a non-root base path and a browser requests the entry document
- **THEN** the served document has all asset URLs and its embedded base-path variable rewritten to include the configured path prefix

#### Scenario: API requests honor base path
- **WHEN** the client application issues API requests while hosted under a base path
- **THEN** all request URLs are prefixed with the configured base path

### Requirement: Cross-Origin Resource Sharing
The backend SHALL answer cross-origin requests with permissive CORS headers, echoing the request origin, allowing credentials, and allowing common HTTP methods, and SHALL short-circuit preflight requests.

#### Scenario: Preflight request
- **WHEN** the backend receives an HTTP OPTIONS preflight request
- **THEN** it responds immediately with a success status and CORS headers (allowed origin echoing the caller, credentials allowed, allowed methods including GET/PUT/POST/DELETE/OPTIONS, allowed content-type header, and a max-age)

### Requirement: Client-Side Routing with Deep-Linkable URLs
The client application SHALL use URL-based routing where every page, including per-cluster sections and nested resource views, has a stable, shareable URL containing the cluster identifier and resource identifiers (URL-encoded where necessary).

#### Scenario: Navigating within the application updates the URL
- **WHEN** a user navigates between views inside the application
- **THEN** the browser URL updates to a distinct path for the target view without a full page reload

#### Scenario: Unknown route fallback
- **WHEN** a user navigates to a URL that matches no known route
- **THEN** the application redirects to a not-found error page

### Requirement: Global Layout with Header and Sidebar
The application SHALL present a persistent layout consisting of a top header bar (product logo linking to the dashboard, version indicator, timezone selector, theme selector, external community links, and current-user menu) and a left navigation sidebar, with the routed page content rendered beside them.

#### Scenario: Header rendered on all main pages
- **WHEN** a user is on any page other than the standalone login page
- **THEN** the top header bar and sidebar layout are visible, and clicking the product logo navigates to the dashboard

### Requirement: Collapsible Responsive Sidebar
The sidebar SHALL be toggleable via a burger button in the header, SHALL default to open on large screens and closed on small screens, and on small screens SHALL overlay the content with a dismissible backdrop and close automatically after navigation.

#### Scenario: Toggling the sidebar
- **WHEN** the user clicks the burger button in the header
- **THEN** the sidebar visibility toggles between shown and hidden

#### Scenario: Sidebar auto-closes on small screens
- **WHEN** the viewport is small and the user navigates to another route or clicks the backdrop overlay
- **THEN** the sidebar closes

### Requirement: Sidebar Cluster Navigation Tree
The sidebar SHALL show a dashboard link followed by one expandable entry per configured cluster, each displaying the cluster name and an online/offline status indicator; expanding a cluster reveals links to that cluster's functional sections.

#### Scenario: Cluster list rendered
- **WHEN** the list of configured clusters has loaded
- **THEN** the sidebar shows one menu entry per cluster with its name and health status

#### Scenario: Default expansion
- **WHEN** exactly one cluster is configured, or the user is currently viewing a given cluster
- **THEN** that cluster's menu is expanded by default and scrolled into view

#### Scenario: Clicking a cluster name
- **WHEN** the user clicks a collapsed cluster's name
- **THEN** the menu expands and the application navigates to that cluster's default (brokers overview) section

#### Scenario: Active section highlighting
- **WHEN** the current URL falls under a section of a cluster
- **THEN** the corresponding sidebar menu item is highlighted as active

### Requirement: Feature-Conditional Sidebar Sections
Within each cluster's sidebar menu, core sections (brokers, topics, consumers) SHALL always be shown, while sections for optional integrations (schema registry, connector management, streaming SQL, access-control lists) SHALL be shown only when the corresponding integration/capability is enabled for that cluster.

#### Scenario: Optional integration not configured
- **WHEN** a cluster has no schema registry (or other optional integration) configured
- **THEN** the corresponding sidebar entry is absent for that cluster

#### Scenario: Optional integration configured
- **WHEN** a cluster reports an optional capability as enabled (including either view or edit capability for access control)
- **THEN** the corresponding sidebar entry is shown for that cluster

### Requirement: Persistent Sidebar Personalization
The application SHALL persist, per cluster, the expanded/collapsed state of the cluster menu and an optional user-chosen color label in browser-local storage, restoring them across sessions.

#### Scenario: Reopening the application
- **WHEN** the user previously expanded a cluster menu or assigned it a color and later reloads the application
- **THEN** the same expansion state and color are restored

### Requirement: Theme Selection with Persistence
The application SHALL offer three theme modes — automatic (follow the operating system's light/dark preference), light, and dark — selectable from the header, and SHALL persist the choice in browser-local storage; automatic mode is the default.

#### Scenario: Selecting dark theme
- **WHEN** the user selects the dark theme option
- **THEN** the entire application immediately re-renders using the dark color scheme and the choice survives page reloads

#### Scenario: Automatic mode
- **WHEN** the theme mode is automatic
- **THEN** the applied color scheme matches the operating system / browser dark-mode preference

### Requirement: Timezone Selection
The header SHALL provide a searchable timezone selector listing all available timezones with their UTC offsets (sorted by offset, plus a plain-UTC option); the selection SHALL be persisted locally, default to the system timezone, and be used when formatting timestamps throughout the application.

#### Scenario: Searching and selecting a timezone
- **WHEN** the user opens the timezone selector, filters by name or offset, and picks an entry
- **THEN** the selector displays the chosen UTC offset and subsequently rendered timestamps use that timezone

#### Scenario: No stored preference
- **WHEN** the user has never chosen a timezone
- **THEN** the application uses the browser's system timezone

### Requirement: Version Display and Outdated-Release Notice
The header SHALL display the running build's version information — the release tag when the build corresponds to the latest known release, otherwise the build timestamp — together with a link from the build commit identifier to the corresponding source-control commit, and SHALL show a warning indicator naming the latest available version when the running build is outdated.

#### Scenario: Running latest release
- **WHEN** the backend reports that the running build matches the latest published release
- **THEN** the header shows the release tag and no warning indicator

#### Scenario: Outdated build
- **WHEN** the backend reports the running build is not the latest release
- **THEN** the header shows a warning indicator whose tooltip states that the version is outdated and names the latest release tag

### Requirement: Global Error Notifications
Any failed API request SHALL automatically produce a non-blocking toast notification (bottom corner of the screen) showing the HTTP status, status text, and the server-provided error message when available, deduplicated per request URL and manually dismissible.

#### Scenario: Server returns an error
- **WHEN** any read or write API call fails with an HTTP error status
- **THEN** a dismissible error toast appears containing the status code and the error message from the response body, or a generic message if none is provided

#### Scenario: Network-level failure
- **WHEN** a request fails without an HTTP status (e.g. network unreachable)
- **THEN** a generic "something went wrong" error toast is shown

### Requirement: Success and Informational Notifications
The application SHALL provide a shared notification mechanism for success, warning, and informational messages using the same toast presentation as errors, with an optional title and dismiss control.

#### Scenario: Successful destructive or mutating action
- **WHEN** a user-initiated action completes successfully and the feature reports it
- **THEN** a success toast with a title and message appears and can be dismissed

### Requirement: Confirmation Dialog for Destructive Actions
The application SHALL provide a single global confirmation dialog used before executing destructive or irreversible actions, showing an action-specific message with Cancel and Confirm buttons; for dangerous actions the confirm button SHALL use a danger style, and while the confirmed action runs the button SHALL show an in-progress state.

#### Scenario: User confirms
- **WHEN** the user clicks Confirm
- **THEN** the associated action executes, the confirm button shows a busy indicator until completion, and the dialog then closes

#### Scenario: User cancels
- **WHEN** the user clicks Cancel or the backdrop
- **THEN** the dialog closes and no action is executed

### Requirement: Error Pages
The application SHALL provide full-page error views with a status-appropriate icon, title, explanatory text, and an action button: access denied (403) with "You do not have permission" messaging, resource not found (404) naming the missing resource type when known, and a generic unexpected-error page, with dedicated URLs for the 403 and 404 pages.

#### Scenario: Access denied
- **WHEN** the user is routed to the access-denied page
- **THEN** a 403-styled page states that access is denied and offers a recovery action button

#### Scenario: Not found
- **WHEN** the user is routed to the not-found page
- **THEN** a 404-styled page states the resource cannot be found

### Requirement: Loading Indicators
The application SHALL display a loading indicator (full-page spinner for initial shell/page loads) while route content or required data is being fetched asynchronously.

#### Scenario: Lazy page load
- **WHEN** the user navigates to a page whose code or data has not yet loaded
- **THEN** a loading spinner is shown until the content renders

### Requirement: Shared Table Pattern with URL-Persisted Sorting and Pagination
All list views SHALL use a shared table component supporting column sorting and pagination whose state (sort column, sort direction, page number, page size) is stored in the URL query string, so that sorted/paged views are shareable and survive reloads; both client-side and server-side data processing modes SHALL be supported, with a default page size of 25.

#### Scenario: Sorting a column
- **WHEN** the user clicks a sortable column header
- **THEN** rows re-order accordingly and the URL query string reflects the sort field and direction

#### Scenario: Changing page
- **WHEN** the user navigates to another page of results
- **THEN** the URL query string reflects the new page, and reloading the URL restores the same page

#### Scenario: Empty result set
- **WHEN** a table has no rows to display
- **THEN** a configurable empty-state message is shown in place of rows

### Requirement: Table Row Selection, Expansion, Filtering, and Resizing
The shared table SHALL optionally support: per-row checkboxes with a batch-actions bar for multi-row operations, expandable rows rendering inline detail content, per-column filters (optionally persisted to the URL query string, resetting pagination when applied), column visibility control, and drag-resizable columns with optionally persisted widths.

#### Scenario: Selecting multiple rows
- **WHEN** row selection is enabled and the user checks several rows
- **THEN** a batch actions bar appears offering operations on the selected rows, and changing page clears the selection

#### Scenario: Applying a column filter
- **WHEN** the user applies a column filter on a filterable table
- **THEN** the visible rows are narrowed accordingly and, where configured, the filter state is persisted in the URL and pagination resets to the first page

### Requirement: Code Editor with Syntax Highlighting
Wherever structured text (JSON-like schemas, protocol definitions, queries, configuration) is displayed or edited, the application SHALL use an embedded code editor providing syntax highlighting appropriate to the content type, line numbers, search within content, soft wrapping, and theme-consistent styling; read-only viewer and diff-comparison variants SHALL also be available.

#### Scenario: Editing structured content
- **WHEN** a user edits schema or query text
- **THEN** the editor highlights syntax according to the content type and supports in-editor search

### Requirement: Page Headings with Back Navigation
Detail and sub-pages SHALL use a shared page-heading pattern showing the current page title and, where the page is nested, a breadcrumb-style back link returning to the parent list view, with an area for page-level action buttons.

#### Scenario: Viewing a nested page
- **WHEN** the user is on a resource detail or creation page
- **THEN** the heading shows a back link to the parent section and the page title

### Requirement: First-Run Setup Redirect
When dynamic application configuration is enabled, no clusters are configured, and the current user has application-configuration permission, the application SHALL automatically redirect to the new-cluster configuration wizard.

#### Scenario: Fresh install with dynamic configuration
- **WHEN** the shell loads, dynamic configuration is enabled, the cluster list is empty, and the user may configure the application
- **THEN** the browser is redirected to the cluster setup page

### Requirement: Standalone Login Page and Session Controls
The application SHALL serve a standalone login page (outside the main shell layout) at a dedicated URL; when the backend indicates authentication is required, the client SHALL redirect there, and when a user is authenticated the header SHALL display the username with a menu offering log-out.

#### Scenario: Unauthenticated access
- **WHEN** the application-info request indicates a redirect to login is required
- **THEN** the client navigates to the login page, which renders without the header and sidebar

#### Scenario: Logging out
- **WHEN** an authenticated user opens the user menu and selects log out
- **THEN** the browser navigates to the logout endpoint (honoring the configured base path)

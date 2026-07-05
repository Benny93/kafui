package mainpage

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// Connector / Connect-cluster column keys. Each resource uses its own dedicated
// column layout instead of the shared four-column mapping.
const (
	colConnName     = "conn_name"
	colConnCluster  = "conn_cluster"
	colConnType     = "conn_type"
	colConnPlugin   = "conn_plugin"
	colConnTopics   = "conn_topics"
	colConnState    = "conn_state"
	colConnGroup    = "conn_group"
	colConnTasks    = "conn_tasks"
	colCCName       = "cc_name"
	colCCVersion    = "cc_version"
	colCCConnectors = "cc_connectors"
	colCCTasks      = "cc_tasks"
)

// connectorStateStyle colours a connector/task state string by semantic role.
func connectorStateStyle(state string) lipgloss.Style {
	switch strings.ToUpper(state) {
	case api.ConnectorStateRunning:
		return lipgloss.NewStyle().Foreground(stylesPkg.Success)
	case api.ConnectorStateFailed:
		return lipgloss.NewStyle().Foreground(stylesPkg.Error)
	case api.ConnectorStatePaused, api.ConnectorStateRestarting:
		return lipgloss.NewStyle().Foreground(stylesPkg.Warning)
	default: // STOPPED, UNASSIGNED, DESTROYED, unknown
		return lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
	}
}

// shortClass returns the trailing dotted segment of a connector class
// (io.confluent.connect.s3.S3SinkConnector -> S3SinkConnector).
func shortClass(class string) string {
	if i := strings.LastIndex(class, "."); i >= 0 && i < len(class)-1 {
		return class[i+1:]
	}
	return class
}

// ---- Connectors resource (aggregated listing) ----

// ConnectorResource lists connectors aggregated across all configured Connect
// clusters via GetConnectors.
type ConnectorResource struct {
	BaseResource
}

// NewConnectorResource creates a new aggregated-connectors resource.
func NewConnectorResource(dataSource api.KafkaDataSource) *ConnectorResource {
	return &ConnectorResource{
		BaseResource: BaseResource{
			resourceType: ConnectorResourceType,
			name:         "Connectors",
			dataSource:   dataSource,
		},
	}
}

// GetData fetches all connectors across the configured Connect clusters.
func (cr *ConnectorResource) GetData() ([]ResourceItem, error) {
	connectors, err := cr.dataSource.GetConnectors()
	if err != nil {
		return nil, err
	}
	items := make([]ResourceItem, 0, len(connectors))
	for _, c := range connectors {
		items = append(items, &ConnectorResourceItem{conn: c})
	}
	return items, nil
}

// ConnectorResourceItem represents a single connector row.
type ConnectorResourceItem struct {
	conn api.Connector
}

// Connector returns the underlying api.Connector.
func (c *ConnectorResourceItem) Connector() api.Connector { return c.conn }

// GetID uniquely identifies the connector by cluster + name.
func (c *ConnectorResourceItem) GetID() string {
	return c.conn.ConnectCluster + "/" + c.conn.Name
}

// GetValues returns unstyled column values.
func (c *ConnectorResourceItem) GetValues() []string {
	return []string{
		c.conn.Name,
		c.conn.ConnectCluster,
		string(c.conn.Type),
		shortClass(c.conn.Class),
		strings.Join(c.conn.Topics, ","),
		c.conn.State,
		c.conn.ConsumerGroup,
		c.tasksText(),
	}
}

func (c *ConnectorResourceItem) tasksText() string {
	running := c.conn.TaskCount - c.conn.FailedTaskCount
	return strconv.Itoa(running) + "/" + strconv.Itoa(c.conn.TaskCount)
}

// GetDetails returns sidebar detail fields.
func (c *ConnectorResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name":           c.conn.Name,
		"Connect":        c.conn.ConnectCluster,
		"Type":           string(c.conn.Type),
		"Class":          c.conn.Class,
		"State":          c.conn.State,
		"Consumer Group": c.conn.ConsumerGroup,
		"Tasks":          c.tasksText(),
	}
}

// connectorItemFrom unwraps a connector resource item from the list wrappers.
func connectorItemFrom(item interface{}) (*ConnectorResourceItem, string, bool) {
	switch v := item.(type) {
	case shared.ResourceListItem:
		ci, ok := v.ResourceItem.(*ConnectorResourceItem)
		return ci, "", ok
	case shared.HighlightedResourceListItem:
		ci, ok := v.ResourceItem.(*ConnectorResourceItem)
		return ci, v.SearchQuery, ok
	case *ConnectorResourceItem:
		return v, "", true
	default:
		return nil, "", false
	}
}

// connectorRowData builds a bubble-table row for a connector, colouring the
// state cell by role and highlighting the tasks cell when tasks have failed.
func connectorRowData(c *ConnectorResourceItem, searchQuery string) table.RowData {
	name := c.conn.Name
	if searchQuery != "" {
		name = shared.HighlightSearchMatches(name, searchQuery)
	}
	tasks := c.tasksText()
	if c.conn.FailedTaskCount > 0 {
		tasks = lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(tasks)
	}
	return table.RowData{
		colConnName:    name,
		colConnCluster: c.conn.ConnectCluster,
		colConnType:    string(c.conn.Type),
		colConnPlugin:  shortClass(c.conn.Class),
		colConnTopics:  strings.Join(c.conn.Topics, ","),
		colConnState:   connectorStateStyle(c.conn.State).Render(c.conn.State),
		colConnGroup:   c.conn.ConsumerGroup,
		colConnTasks:   tasks,
	}
}

// ---- Connect clusters resource (overview) ----

// ConnectClusterResource lists the configured Connect clusters with aggregated
// statistics via GetConnectClusters(withStats=true).
type ConnectClusterResource struct {
	BaseResource
}

// NewConnectClusterResource creates a new Connect-clusters overview resource.
func NewConnectClusterResource(dataSource api.KafkaDataSource) *ConnectClusterResource {
	return &ConnectClusterResource{
		BaseResource: BaseResource{
			resourceType: ConnectClusterResourceType,
			name:         "Connect Clusters",
			dataSource:   dataSource,
		},
	}
}

// GetData fetches the Connect clusters with stats.
func (cr *ConnectClusterResource) GetData() ([]ResourceItem, error) {
	clusters, err := cr.dataSource.GetConnectClusters(true)
	if err != nil {
		return nil, err
	}
	items := make([]ResourceItem, 0, len(clusters))
	for _, c := range clusters {
		items = append(items, &ConnectClusterResourceItem{cluster: c})
	}
	return items, nil
}

// ConnectClusterResourceItem represents a single Connect-cluster row.
type ConnectClusterResourceItem struct {
	cluster api.ConnectCluster
}

// Cluster returns the underlying api.ConnectCluster.
func (c *ConnectClusterResourceItem) Cluster() api.ConnectCluster { return c.cluster }

// GetID returns the Connect cluster name.
func (c *ConnectClusterResourceItem) GetID() string { return c.cluster.Name }

func (c *ConnectClusterResourceItem) versionText() string {
	if !c.cluster.Reachable {
		return "unreachable"
	}
	return c.cluster.Version
}

// GetValues returns unstyled column values.
func (c *ConnectClusterResourceItem) GetValues() []string {
	if !c.cluster.Reachable {
		return []string{c.cluster.Name, "unreachable", "-", "-"}
	}
	runningTasks := c.cluster.TaskCount - c.cluster.FailedTaskCount
	return []string{
		c.cluster.Name,
		c.cluster.Version,
		strconv.Itoa(c.cluster.ConnectorCount),
		strconv.Itoa(runningTasks),
	}
}

// GetDetails returns sidebar detail fields.
func (c *ConnectClusterResourceItem) GetDetails() map[string]string {
	return map[string]string{
		"Name":              c.cluster.Name,
		"Address":           c.cluster.Address,
		"Version":           c.versionText(),
		"Connectors":        strconv.Itoa(c.cluster.ConnectorCount),
		"Failed Connectors": strconv.Itoa(c.cluster.FailedConnectorCount),
		"Tasks":             strconv.Itoa(c.cluster.TaskCount),
		"Failed Tasks":      strconv.Itoa(c.cluster.FailedTaskCount),
	}
}

// connectClusterItemFrom unwraps a Connect-cluster item from the list wrappers.
func connectClusterItemFrom(item interface{}) (*ConnectClusterResourceItem, bool) {
	switch v := item.(type) {
	case shared.ResourceListItem:
		ci, ok := v.ResourceItem.(*ConnectClusterResourceItem)
		return ci, ok
	case shared.HighlightedResourceListItem:
		ci, ok := v.ResourceItem.(*ConnectClusterResourceItem)
		return ci, ok
	case *ConnectClusterResourceItem:
		return v, true
	default:
		return nil, false
	}
}

// connectClusterRowData builds a bubble-table row for a Connect cluster,
// styling failed counts and unreachable clusters with the error/muted roles.
func connectClusterRowData(c *ConnectClusterResourceItem) table.RowData {
	if !c.cluster.Reachable {
		muted := lipgloss.NewStyle().Foreground(stylesPkg.FgMuted)
		return table.RowData{
			colCCName:       c.cluster.Name,
			colCCVersion:    muted.Render("unreachable"),
			colCCConnectors: muted.Render("-"),
			colCCTasks:      muted.Render("-"),
		}
	}
	connectors := strconv.Itoa(c.cluster.ConnectorCount)
	if c.cluster.FailedConnectorCount > 0 {
		connectors = lipgloss.NewStyle().Foreground(stylesPkg.Error).
			Render(connectors + " (" + strconv.Itoa(c.cluster.FailedConnectorCount) + " failed)")
	}
	runningTasks := c.cluster.TaskCount - c.cluster.FailedTaskCount
	tasks := strconv.Itoa(runningTasks) + "/" + strconv.Itoa(c.cluster.TaskCount)
	if c.cluster.FailedTaskCount > 0 {
		tasks = lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(tasks)
	}
	return table.RowData{
		colCCName:       c.cluster.Name,
		colCCVersion:    c.cluster.Version,
		colCCConnectors: connectors,
		colCCTasks:      tasks,
	}
}

// connectorMatchesQuery matches a connector against a filter query. Bare terms
// substring-match across name/status/type/plugin; a term with a known prefix
// (status:, type:, connect:, plugin:) matches that field only. All terms
// (space-separated) must match (AND semantics).
func connectorMatchesQuery(c api.Connector, query string) bool {
	for _, term := range strings.Fields(strings.ToLower(query)) {
		field, val, hasPrefix := strings.Cut(term, ":")
		if hasPrefix {
			var target string
			switch field {
			case "status", "state":
				target = strings.ToLower(c.State)
			case "type":
				target = strings.ToLower(string(c.Type))
			case "connect", "cluster":
				target = strings.ToLower(c.ConnectCluster)
			case "plugin", "class":
				target = strings.ToLower(c.Class)
			default:
				// Unknown prefix: fall back to a bare substring match on the whole term.
				if !connectorTermMatches(c, term) {
					return false
				}
				continue
			}
			if !strings.Contains(target, val) {
				return false
			}
			continue
		}
		if !connectorTermMatches(c, term) {
			return false
		}
	}
	return true
}

func connectorTermMatches(c api.Connector, term string) bool {
	hay := strings.ToLower(strings.Join([]string{
		c.Name, c.State, string(c.Type), c.Class, c.ConnectCluster,
	}, " "))
	return strings.Contains(hay, term)
}

// isConnectorResource reports whether the connectors resource is active.
func (k *KafuiContentProvider) isConnectorResource() bool {
	return k.currentResource != nil && k.currentResource.GetType() == ConnectorResourceType
}

// isConnectClusterResource reports whether the Connect-clusters resource is active.
func (k *KafuiContentProvider) isConnectClusterResource() bool {
	return k.currentResource != nil && k.currentResource.GetType() == ConnectClusterResourceType
}

// ---- create-connector flow (KC-17) ----

// openCreateConnectorForm builds and shows the create-connector overlay. The
// connect-cluster selector is skipped (defaulted) when exactly one reachable
// cluster is configured. Config is entered as single-line JSON.
//
// ponytail: multi-line textarea / $EDITOR handoff (tea.ExecProcess) for large
// JSON configs is deferred — a single-line JSON Text field covers the flow.
func (k *KafuiContentProvider) openCreateConnectorForm() tea.Cmd {
	if k.common != nil && k.common.IsReadOnly() {
		return core.NewNotification(core.StatusWarning, "Create connector", "cluster is read-only")
	}
	clusters, err := k.dataSource.GetConnectClusters(false)
	if err != nil {
		return core.NotifyError("Create connector", err)
	}
	var reachable []string
	for _, c := range clusters {
		if c.Reachable {
			reachable = append(reachable, c.Name)
		}
	}
	if len(reachable) == 0 {
		return core.NewNotification(core.StatusWarning, "Create connector", "no reachable Connect cluster")
	}

	// Offer the installed plugin classes of the first cluster as a convenience.
	var pluginOpts []string
	if plugins, perr := k.dataSource.GetConnectorPlugins(reachable[0]); perr == nil {
		for _, p := range plugins {
			pluginOpts = append(pluginOpts, p.Class)
		}
	}

	fields := []form.Field{}
	if len(reachable) > 1 {
		fields = append(fields, form.Field{Name: "connect", Label: "Connect cluster", Type: form.Select, Options: reachable, Default: reachable[0]})
	} else {
		fields = append(fields, form.Field{Name: "connect", Label: "Connect cluster", Type: form.Text, Default: reachable[0]})
	}
	fields = append(fields,
		form.Field{Name: "name", Label: "Name", Type: form.Text, Required: true},
	)
	if len(pluginOpts) > 0 {
		fields = append(fields, form.Field{Name: "plugin", Label: "Plugin class", Type: form.Select, Options: pluginOpts, Default: pluginOpts[0]})
	} else {
		fields = append(fields, form.Field{Name: "plugin", Label: "Plugin class", Type: form.Text, Required: true})
	}
	fields = append(fields, form.Field{Name: "config", Label: "Config (JSON)", Type: form.Text, Default: "{}"})

	k.connectForm = form.New(fields)
	k.showConnectForm = true
	return k.connectForm.Focus()
}

// handleConnectorFormSubmit validates the submitted config against the plugin,
// then creates the connector. Validation/parse errors keep the form open.
func (k *KafuiContentProvider) handleConnectorFormSubmit(values map[string]string) tea.Cmd {
	connect := values["connect"]
	name := values["name"]
	class := values["plugin"]

	config := map[string]string{}
	if raw := strings.TrimSpace(values["config"]); raw != "" && raw != "{}" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			return core.NotifyError("Invalid JSON config", err)
		}
	}
	config["name"] = name
	if class != "" {
		config["connector.class"] = class
	}

	ds := k.dataSource
	return func() tea.Msg {
		result, err := ds.ValidateConnectorConfig(connect, config["connector.class"], config)
		if err != nil {
			return connectorCreatedMsg{connect: connect, name: name, err: err}
		}
		if result.ErrorCount > 0 {
			return connectorCreatedMsg{connect: connect, name: name, err: fmt.Errorf("validation failed: %s", firstValidationError(result))}
		}
		if _, cerr := ds.CreateConnector(connect, name, config); cerr != nil {
			return connectorCreatedMsg{connect: connect, name: name, err: cerr}
		}
		return connectorCreatedMsg{connect: connect, name: name}
	}
}

// firstValidationError returns the first per-field error message from a result.
func firstValidationError(r api.ConnectorValidationResult) string {
	for _, c := range r.Configs {
		if len(c.Errors) > 0 {
			return c.Errors[0]
		}
	}
	return "connector configuration is invalid"
}

// handleConnectorCreated closes the form and navigates on success; on failure
// (e.g. duplicate name, validation error) it keeps the form open and reports it.
func (k *KafuiContentProvider) handleConnectorCreated(msg connectorCreatedMsg) tea.Cmd {
	if msg.err != nil {
		return core.NotifyError("Create connector failed", msg.err)
	}
	k.showConnectForm = false
	k.connectForm = nil
	return tea.Batch(
		core.NewNotification(core.StatusSuccess, "Connector created", msg.name),
		k.loadCurrentResource(),
		core.NewPageChangeMsg("connector:"+msg.connect+":"+msg.name, map[string]interface{}{
			"connect": msg.connect,
			"name":    msg.name,
		}),
	)
}

// ---- CSV export (KC-19) ----

// exportConnectorsCSV writes the currently filtered/sorted connector rows to a
// timestamped CSV file and reports the path.
func (k *KafuiContentProvider) exportConnectorsCSV() tea.Cmd {
	items := k.allItems
	if k.isFiltered {
		items = k.filteredItems
	}
	rows := [][]string{{"name", "connect", "type", "plugin", "topics", "status", "consumer_group", "tasks_running", "tasks_total"}}
	for _, it := range items {
		ci, _, ok := connectorItemFrom(it)
		if !ok {
			continue
		}
		c := ci.conn
		rows = append(rows, []string{
			c.Name, c.ConnectCluster, string(c.Type), c.Class,
			strings.Join(c.Topics, ";"), c.State, c.ConsumerGroup,
			strconv.Itoa(c.TaskCount - c.FailedTaskCount), strconv.Itoa(c.TaskCount),
		})
	}
	return writeCSVFile("kafui-connectors", rows)
}

// exportConnectClustersCSV writes the Connect-clusters overview rows to CSV.
func (k *KafuiContentProvider) exportConnectClustersCSV() tea.Cmd {
	items := k.allItems
	if k.isFiltered {
		items = k.filteredItems
	}
	rows := [][]string{{"name", "address", "version", "reachable", "connectors", "failed_connectors", "tasks", "failed_tasks"}}
	for _, it := range items {
		cci, ok := connectClusterItemFrom(it)
		if !ok {
			continue
		}
		c := cci.cluster
		rows = append(rows, []string{
			c.Name, c.Address, c.Version, strconv.FormatBool(c.Reachable),
			strconv.Itoa(c.ConnectorCount), strconv.Itoa(c.FailedConnectorCount),
			strconv.Itoa(c.TaskCount), strconv.Itoa(c.FailedTaskCount),
		})
	}
	return writeCSVFile("kafui-connect-clusters", rows)
}

// writeCSVFile writes rows to ./<prefix>-<timestamp>.csv and returns a status cmd.
func writeCSVFile(prefix string, rows [][]string) tea.Cmd {
	if len(rows) <= 1 {
		return core.NewNotification(core.StatusWarning, "CSV export", "nothing to export")
	}
	filename := fmt.Sprintf("%s-%s.csv", prefix, time.Now().Format("20060102-150405"))
	f, err := os.Create(filename)
	if err != nil {
		return core.NotifyError("CSV export failed", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.WriteAll(rows); err != nil {
		return core.NotifyError("CSV export failed", err)
	}
	w.Flush()
	abs, _ := filepath.Abs(filename)
	return core.NewNotification(core.StatusInfo, "Exported", abs)
}

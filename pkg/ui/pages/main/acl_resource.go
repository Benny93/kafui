package mainpage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/Benny93/kafui/pkg/ui/shared/aclcsv"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// ACL-specific column keys (AQ-13). ACLs use their own six-column layout
// (principal/resource/pattern/host/operation/permission).
const (
	colACLPrincipal  = "acl_principal"
	colACLResource   = "acl_resource"
	colACLPattern    = "acl_pattern"
	colACLHost       = "acl_host"
	colACLOperation  = "acl_operation"
	colACLPermission = "acl_permission"
)

// enumerable value sets for the ACL create form / filter cycles.
var (
	aclResourceTypes = []string{"Topic", "Group", "Cluster", "TransactionalID", "DelegationToken"}
	aclPatternTypes  = []string{"Literal", "Prefixed"}
	aclOperations    = []string{"Read", "Write", "Create", "Delete", "Alter", "Describe", "ClusterAction", "DescribeConfigs", "AlterConfigs", "IdempotentWrite", "All"}
	aclPermissions   = []string{"Allow", "Deny"}
)

// aclResourceTypeFilterCycle / aclPatternFilterCycle drive the header filter
// cycle selectors ("" = Any). ponytail: single-value cycle stands in for the
// spec's multi-select column filters (see TUI adaptation notes).
var (
	aclResourceTypeFilterCycle = append([]string{""}, aclResourceTypes...)
	aclPatternFilterCycle      = append([]string{""}, aclPatternTypes...)
)

// isACLResource reports whether the ACLs resource is currently active.
func (k *KafuiContentProvider) isACLResource() bool {
	return k.currentResource != nil && k.currentResource.GetType() == ACLResourceType
}

// aclItemFrom unwraps an *ACLResourceItem from the list-item wrappers, returning
// the item plus any active search query (for principal highlighting).
func aclItemFrom(item interface{}) (*ACLResourceItem, string, bool) {
	switch v := item.(type) {
	case shared.ResourceListItem:
		ari, ok := v.ResourceItem.(*ACLResourceItem)
		return ari, "", ok
	case shared.HighlightedResourceListItem:
		ari, ok := v.ResourceItem.(*ACLResourceItem)
		return ari, v.SearchQuery, ok
	case *ACLResourceItem:
		return v, "", true
	default:
		return nil, "", false
	}
}

// patternBadge renders the pattern type as a coloured [LITERAL]/[PREFIXED] badge.
func patternBadge(pattern string) string {
	if pattern == "" {
		pattern = "Literal"
	}
	style := lipgloss.NewStyle().Foreground(stylesPkg.Info)
	if strings.EqualFold(pattern, "Prefixed") {
		style = lipgloss.NewStyle().Foreground(stylesPkg.Warning)
	}
	return style.Render("[" + strings.ToUpper(pattern) + "]")
}

// permissionCell styles the permission by role: Allow → success, Deny → error.
func permissionCell(permission string) string {
	switch {
	case strings.EqualFold(permission, "Allow"):
		return lipgloss.NewStyle().Foreground(stylesPkg.Success).Render(permission)
	case strings.EqualFold(permission, "Deny"):
		return lipgloss.NewStyle().Foreground(stylesPkg.Error).Render(permission)
	default:
		return permission
	}
}

// aclRowData builds a bubble-table row for an ACL binding (AQ-13).
func aclRowData(a *ACLResourceItem, searchQuery string) table.RowData {
	principal := a.principal
	if searchQuery != "" {
		principal = shared.HighlightSearchMatches(principal, searchQuery)
	}
	pattern := a.patternType
	if pattern == "" {
		pattern = "Literal"
	}
	host := a.host
	if host == "" {
		host = "*"
	}
	return table.RowData{
		colACLPrincipal:  principal,
		colACLResource:   a.resourceType + ":" + a.resourceName,
		colACLPattern:    patternBadge(pattern),
		colACLHost:       host,
		colACLOperation:  a.operation,
		colACLPermission: permissionCell(a.permission),
	}
}

// --- filtering (AQ-14) ---
//
// ponytail: only the enumerable dimensions (resource type, pattern type) get
// cycle filters, wired server-side through GetACLsFiltered; free-text resource
// name/host filters and the config-defaulted fuzzy-principal-search toggle are
// deferred — the cycle filters + principal substring search cover the spec's
// primary scenarios.

// aclResource returns the active *ACLResource (or nil when ACLs are not active).
func (k *KafuiContentProvider) aclResource() *ACLResource {
	if ar, ok := k.currentResource.(*ACLResource); ok {
		return ar
	}
	return nil
}

// cycleACLResourceTypeFilter advances the resource-type filter and reloads via
// GetACLsFiltered.
func (k *KafuiContentProvider) cycleACLResourceTypeFilter() tea.Cmd {
	ar := k.aclResource()
	if ar == nil {
		return nil
	}
	f := ar.Filter()
	f.ResourceType = nextInCycle(aclResourceTypeFilterCycle, f.ResourceType)
	ar.SetFilter(f)
	return k.reloadACLs()
}

// cycleACLPatternFilter advances the pattern-type filter and reloads.
func (k *KafuiContentProvider) cycleACLPatternFilter() tea.Cmd {
	ar := k.aclResource()
	if ar == nil {
		return nil
	}
	f := ar.Filter()
	f.PatternType = nextInCycle(aclPatternFilterCycle, f.PatternType)
	ar.SetFilter(f)
	return k.reloadACLs()
}

// nextInCycle returns the element after cur in the cycle (wrapping); if cur is
// not found it returns the second element (first is the "" any-sentinel).
func nextInCycle(cycle []string, cur string) string {
	for i, v := range cycle {
		if v == cur {
			return cycle[(i+1)%len(cycle)]
		}
	}
	if len(cycle) > 1 {
		return cycle[1]
	}
	return ""
}

// reloadACLs re-fetches the ACL list with the active filter, resetting to page 1.
func (k *KafuiContentProvider) reloadACLs() tea.Cmd {
	k.pendingReset = true
	return k.loadCurrentResource()
}

// aclFilterSummary describes the active ACL filters for the header, or "" when
// none are set.
func (k *KafuiContentProvider) aclFilterSummary() string {
	ar := k.aclResource()
	if ar == nil {
		return ""
	}
	f := ar.Filter()
	var parts []string
	if f.ResourceType != "" {
		parts = append(parts, "type="+f.ResourceType)
	}
	if f.PatternType != "" {
		parts = append(parts, "pattern="+f.PatternType)
	}
	return strings.Join(parts, " ")
}

// --- delete (AQ-15) ---

// deleteSelectedACL shows a confirmation modal summarizing the binding, then
// deletes it on confirm. Gated on the ACL edit capability.
func (k *KafuiContentProvider) deleteSelectedACL() tea.Cmd {
	if !k.canEditACL() {
		return aclEditDisabledHint()
	}
	ari, _, ok := aclItemFrom(k.GetSelectedResourceItem())
	if !ok {
		return nil
	}
	entry := ari.Entry()
	ds := k.dataSource
	summary := fmt.Sprintf("%s  %s:%s [%s]  %s %s  host=%s",
		entry.Principal, entry.ResourceType, entry.ResourceName, entry.PatternType,
		entry.Operation, entry.Permission, ari.host)
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete ACL binding",
			Message:      "Delete this ACL binding? This cannot be undone.\n" + summary,
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				return aclDeletedMsg{summary: summary, err: ds.DeleteACL(entry)}
			},
		}
	}
}

// --- create / convenience form (AQ-16 / AQ-17) ---

// splitList parses a comma-separated field into a trimmed, non-empty slice.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// buildACLForm builds the unified ACL creation form. A "type" selector switches
// between the Custom binding and the Consumer/Producer/Stream-app convenience
// flows; irrelevant fields are simply ignored on submit for the chosen type.
// ponytail: a single form with a type selector stands in for per-variant panels
// (the form component is static); unused fields are left blank. Topic/group
// autocomplete suggestions (GetTopicNames/GetConsumerGroups) are deferred — the
// convenience fields take comma-separated names typed by the user instead.
func buildACLForm() *form.Form {
	return form.New([]form.Field{
		{Name: "type", Label: "Form type", Type: form.Select, Options: []string{"Custom", "Consumer", "Producer", "StreamApp"}, Default: "Custom"},
		{Name: "principal", Label: "Principal (e.g. User:alice)", Type: form.Text, Required: true, Validator: principalValidator},
		{Name: "host", Label: "Host", Type: form.Text, Default: "*"},
		// Custom
		{Name: "resource_type", Label: "[Custom] Resource type", Type: form.Select, Options: aclResourceTypes},
		{Name: "resource_name", Label: "[Custom] Resource name", Type: form.Text},
		{Name: "pattern_type", Label: "[Custom] Pattern type", Type: form.Select, Options: aclPatternTypes},
		{Name: "operation", Label: "[Custom] Operation", Type: form.Select, Options: aclOperations},
		{Name: "permission", Label: "[Custom] Permission", Type: form.Select, Options: aclPermissions},
		// Consumer / Producer
		{Name: "topics", Label: "[Consumer/Producer] Topics (comma-separated)", Type: form.Text},
		{Name: "topic_prefix", Label: "[Consumer/Producer] Topic prefix", Type: form.Text},
		// Consumer
		{Name: "groups", Label: "[Consumer] Groups (comma-separated)", Type: form.Text},
		{Name: "group_prefix", Label: "[Consumer] Group prefix", Type: form.Text},
		// Producer
		{Name: "transactional_id", Label: "[Producer] Transactional ID", Type: form.Text},
		{Name: "tx_prefix", Label: "[Producer] Transactional ID prefix", Type: form.Text},
		{Name: "idempotent", Label: "[Producer] Idempotent", Type: form.Bool},
		// Stream app
		{Name: "app_id", Label: "[StreamApp] Application ID", Type: form.Text},
		{Name: "input_topics", Label: "[StreamApp] Input topics (comma-separated)", Type: form.Text},
		{Name: "output_topics", Label: "[StreamApp] Output topics (comma-separated)", Type: form.Text},
	})
}

func principalValidator(v string) error {
	return api.ValidatePrincipal(v)
}

// openCreateACLForm opens the ACL creation overlay (custom + convenience flows).
func (k *KafuiContentProvider) openCreateACLForm() tea.Cmd {
	if !k.canEditACL() {
		return aclEditDisabledHint()
	}
	k.aclForm = buildACLForm()
	k.showACLForm = true
	return k.aclForm.Focus()
}

// handleACLFormSubmit builds the requested bindings and creates them, reporting
// per-binding failures. Custom → one binding; convenience → expanded set.
func (k *KafuiContentProvider) handleACLFormSubmit(v map[string]string) tea.Cmd {
	ds := k.dataSource
	principal := strings.TrimSpace(v["principal"])
	host := strings.TrimSpace(v["host"])

	var entries []api.ACLEntry
	var err error
	switch v["type"] {
	case "Consumer":
		entries, err = api.ExpandConsumerACLs(principal, host, splitList(v["topics"]), splitList(v["groups"]), strings.TrimSpace(v["topic_prefix"]), strings.TrimSpace(v["group_prefix"]))
	case "Producer":
		entries, err = api.ExpandProducerACLs(principal, host, splitList(v["topics"]), strings.TrimSpace(v["topic_prefix"]), strings.TrimSpace(v["transactional_id"]), strings.TrimSpace(v["tx_prefix"]), v["idempotent"] == "true")
	case "StreamApp":
		entries, err = api.ExpandStreamAppACLs(principal, host, strings.TrimSpace(v["app_id"]), splitList(v["input_topics"]), splitList(v["output_topics"]))
	default: // Custom
		entry := api.ACLEntry{
			Principal:    principal,
			Host:         host,
			ResourceType: v["resource_type"],
			ResourceName: strings.TrimSpace(v["resource_name"]),
			PatternType:  v["pattern_type"],
			Operation:    v["operation"],
			Permission:   v["permission"],
		}
		if verr := api.ValidateACLEntry(entry); verr != nil {
			err = verr
		} else {
			entries = []api.ACLEntry{entry}
		}
	}

	if err != nil {
		// Keep the form open so the user can correct the input.
		return func() tea.Msg { return aclCreatedMsg{err: err} }
	}
	if len(entries) == 0 {
		return func() tea.Msg {
			return aclCreatedMsg{err: api.ACLValidationError{Field: "form", Reason: "no bindings to create — fill in the fields for the selected type"}}
		}
	}

	return func() tea.Msg {
		var failures []string
		created := 0
		for _, e := range entries {
			if ce := ds.CreateACL(e); ce != nil {
				failures = append(failures, e.Principal+" "+e.ResourceType+":"+e.ResourceName+": "+ce.Error())
			} else {
				created++
			}
		}
		return aclCreatedMsg{created: created, failures: failures}
	}
}

// --- CSV export (AQ-18) ---

// exportACLsCSV serializes the currently filtered/searched ACL list via aclcsv
// and writes it to a timestamped file, announcing the absolute path.
func (k *KafuiContentProvider) exportACLsCSV() tea.Cmd {
	items := k.activeItems()
	entries := make([]api.ACLEntry, 0, len(items))
	for _, item := range items {
		if ari, _, ok := aclItemFrom(item); ok {
			entries = append(entries, ari.Entry())
		}
	}
	if len(entries) == 0 {
		return nil
	}
	ctx := k.dataSource.GetContext()
	filename := fmt.Sprintf("kafui-acls-%s-%s.csv", ctx, time.Now().Format("20060102-150405"))
	f, err := os.Create(filename)
	if err != nil {
		return core.NotifyError("CSV export failed", err)
	}
	defer f.Close()
	if _, werr := f.WriteString(aclcsv.Marshal(entries)); werr != nil {
		return core.NotifyError("CSV export failed", werr)
	}
	abs, _ := filepath.Abs(filename)
	return core.NewNotification(core.StatusInfo, "ACLs exported", abs)
}

// --- declarative CSV sync (AQ-18) ---

// openACLSyncForm opens a one-field path prompt for the declarative sync flow.
func (k *KafuiContentProvider) openACLSyncForm() tea.Cmd {
	if !k.canEditACL() {
		return aclEditDisabledHint()
	}
	k.aclSyncForm = form.New([]form.Field{
		{Name: "path", Label: "Path to ACL CSV file", Type: form.Text, Required: true},
	})
	k.showACLSyncForm = true
	return k.aclSyncForm.Focus()
}

// handleACLSyncSubmit reads + parses the CSV, computes the sync plan, and shows
// a confirmation summarizing the additions/removals before applying anything.
// Parse errors abort before any datasource mutation.
func (k *KafuiContentProvider) handleACLSyncSubmit(path string) tea.Cmd {
	ds := k.dataSource
	data, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return core.NotifyError("ACL sync failed", err)
	}
	desired, err := aclcsv.Parse(string(data))
	if err != nil {
		return core.NotifyError("ACL sync: malformed CSV", err)
	}
	plan, err := aclcsv.SyncACLs(ds, desired)
	if err != nil {
		return core.NotifyError("ACL sync failed", err)
	}
	shared.Log.Info("ACL sync plan", "create", len(plan.ToCreate), "delete", len(plan.ToDelete))
	if plan.Empty() {
		return core.NewNotification(core.StatusInfo, "ACL sync", "already in sync — nothing to do")
	}
	create, del := len(plan.ToCreate), len(plan.ToDelete)
	msg := fmt.Sprintf("Apply ACL sync plan?\n%d binding(s) to create, %d to delete.\n%s", create, del, aclPlanPreview(plan))
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Sync ACLs from CSV",
			Message:      msg,
			Danger:       del > 0,
			ConfirmLabel: "Apply",
			OnConfirm: func() tea.Msg {
				return aclSyncedMsg{created: create, deleted: del, err: plan.Apply(ds)}
			},
		}
	}
}

// aclPlanPreview lists the first few create/delete bindings for the confirm view.
func aclPlanPreview(plan aclcsv.SyncPlan) string {
	var lines []string
	add := func(prefix string, es []api.ACLEntry) {
		for i, e := range es {
			if i >= 5 {
				lines = append(lines, fmt.Sprintf("%s … (+%d more)", prefix, len(es)-5))
				break
			}
			lines = append(lines, fmt.Sprintf("%s %s %s:%s [%s] %s %s", prefix, e.Principal, e.ResourceType, e.ResourceName, e.PatternType, e.Operation, e.Permission))
		}
	}
	add("+", plan.ToCreate)
	add("-", plan.ToDelete)
	return strings.Join(lines, "\n")
}

// --- capability gating (AQ-21) ---

// canEditACL reports whether ACL mutations are permitted for the active cluster.
// When capabilities are unknown (no Common / collector not ready) it defaults to
// enabled so actions are not blocked before the first collection cycle.
func (k *KafuiContentProvider) canEditACL() bool {
	if k.common == nil {
		return true
	}
	return k.common.HasCapability(api.CapACLEdit)
}

// aclEditDisabledHint surfaces the disabled-edit explanation in the status bar.
func aclEditDisabledHint() tea.Cmd {
	return core.NewNotification(core.StatusInfo, "ACLs", "ACL editing is not available on this cluster (no ALTER on cluster).")
}

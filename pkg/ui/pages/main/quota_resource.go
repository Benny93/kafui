package mainpage

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

// Quota-specific column keys (AQ-19).
const (
	colQuotaUser   = "quota_user"
	colQuotaClient = "quota_client"
	colQuotaIP     = "quota_ip"
	colQuotaValues = "quota_values"
)

// isQuotaResource reports whether the quotas resource is currently active.
func (k *KafuiContentProvider) isQuotaResource() bool {
	return k.currentResource != nil && k.currentResource.GetType() == QuotaResourceType
}

// quotaItemFrom unwraps a *QuotaResourceItem from the list-item wrappers.
func quotaItemFrom(item interface{}) (*QuotaResourceItem, bool) {
	switch v := item.(type) {
	case shared.ResourceListItem:
		qri, ok := v.ResourceItem.(*QuotaResourceItem)
		return qri, ok
	case shared.HighlightedResourceListItem:
		qri, ok := v.ResourceItem.(*QuotaResourceItem)
		return qri, ok
	case *QuotaResourceItem:
		return v, true
	default:
		return nil, false
	}
}

// quotaRowData builds a bubble-table row for a client-quota entry.
func quotaRowData(q *QuotaResourceItem) table.RowData {
	vals := q.GetValues()
	return table.RowData{
		colQuotaUser:   vals[0],
		colQuotaClient: vals[1],
		colQuotaIP:     vals[2],
		colQuotaValues: vals[3],
	}
}

// --- upsert / delete form (AQ-20) ---
//
// ponytail: the property editor is a single comma-separated "name=value" text
// field rather than add/edit/remove rows; it still enforces replace semantics
// and numeric validation. Create mode maps a blank identifier field to nil
// (absent); the explicit <default> (empty-string) entity is not creatable from
// the UI (edit mode preserves it since the entity is fixed).

// openQuotaForm opens the quota editor. When an existing row is highlighted it
// pre-fills the entity (fixed) and its properties; otherwise it starts blank.
func (k *KafuiContentProvider) openQuotaForm(edit bool) tea.Cmd {
	var fields []form.Field
	k.quotaEditEntity = nil

	if edit {
		qri, ok := quotaItemFrom(k.GetSelectedResourceItem())
		if !ok {
			return nil
		}
		ent := qri.Entity()
		k.quotaEditEntity = &ent
		fields = []form.Field{
			{Name: "user", Label: "User", Type: form.Text, Default: ptrVal(ent.User)},
			{Name: "client_id", Label: "Client ID", Type: form.Text, Default: ptrVal(ent.ClientID)},
			{Name: "ip", Label: "IP", Type: form.Text, Default: ptrVal(ent.IP)},
			{Name: "quotas", Label: "Quotas (name=value, comma-separated; empty = delete)", Type: form.Text, Default: quotaPropsToString(qri.Quotas())},
		}
	} else {
		fields = []form.Field{
			{Name: "user", Label: "User (at least one of user/client/ip required)", Type: form.Text},
			{Name: "client_id", Label: "Client ID", Type: form.Text},
			{Name: "ip", Label: "IP", Type: form.Text},
			{Name: "quotas", Label: "Quotas (name=value, comma-separated)", Type: form.Text},
		}
	}
	k.quotaForm = form.New(fields)
	k.showQuotaForm = true
	return k.quotaForm.Focus()
}

func ptrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// quotaPropsToString renders a quota map as sorted "name=value" comma-list.
func quotaPropsToString(m map[string]float64) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+strconv.FormatFloat(m[k], 'f', -1, 64))
	}
	return strings.Join(parts, ", ")
}

// parseQuotaProps parses a "name=value, ..." string into a quota map, rejecting
// malformed pairs and non-numeric values.
func parseQuotaProps(s string) (map[string]float64, error) {
	out := map[string]float64{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || strings.TrimSpace(kv[0]) == "" {
			return nil, fmt.Errorf("invalid quota %q (want name=value)", part)
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(kv[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("quota %q: value must be numeric", strings.TrimSpace(kv[0]))
		}
		out[strings.TrimSpace(kv[0])] = v
	}
	return out, nil
}

// entityFromForm builds a ClientQuotaEntity from the form fields. For edit mode
// the fixed entity is reused; for create mode a blank field means "absent" (nil).
func (k *KafuiContentProvider) entityFromForm(v map[string]string) api.ClientQuotaEntity {
	if k.quotaEditEntity != nil {
		return *k.quotaEditEntity
	}
	strOrNil := func(s string) *string {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		return &s
	}
	return api.ClientQuotaEntity{
		User:     strOrNil(v["user"]),
		ClientID: strOrNil(v["client_id"]),
		IP:       strOrNil(v["ip"]),
	}
}

// handleQuotaFormSubmit validates the entity + properties and calls
// AlterClientQuotas (replace semantics). An empty property set is the delete
// path and requires a confirmation modal before the call.
func (k *KafuiContentProvider) handleQuotaFormSubmit(v map[string]string) tea.Cmd {
	entity := k.entityFromForm(v)
	if err := api.ValidateQuotaEntity(entity); err != nil {
		return func() tea.Msg { return quotaAlteredMsg{err: err} }
	}
	quotas, err := parseQuotaProps(v["quotas"])
	if err != nil {
		return func() tea.Msg { return quotaAlteredMsg{err: err} }
	}

	ds := k.dataSource
	editing := k.quotaEditEntity != nil

	// Empty property set → delete path, gated by confirmation.
	if len(quotas) == 0 {
		label := describeEntity(entity)
		return func() tea.Msg {
			return core.ShowConfirmMsg{
				Title:        "Delete client quota",
				Message:      fmt.Sprintf("Delete all quotas for %s? This cannot be undone.", label),
				Danger:       true,
				ConfirmLabel: "Delete",
				OnConfirm: func() tea.Msg {
					return quotaAlteredMsg{action: "deleted", err: ds.AlterClientQuotas(entity, nil)}
				},
			}
		}
	}

	action := "created"
	if editing {
		action = "updated"
	}
	return func() tea.Msg {
		return quotaAlteredMsg{action: action, err: ds.AlterClientQuotas(entity, quotas)}
	}
}

// deleteSelectedQuota deletes the highlighted quota entity (sets an empty quota
// set) behind a confirmation modal.
func (k *KafuiContentProvider) deleteSelectedQuota() tea.Cmd {
	qri, ok := quotaItemFrom(k.GetSelectedResourceItem())
	if !ok {
		return nil
	}
	entity := qri.Entity()
	ds := k.dataSource
	label := describeEntity(entity)
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete client quota",
			Message:      fmt.Sprintf("Delete all quotas for %s? This cannot be undone.", label),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm: func() tea.Msg {
				return quotaAlteredMsg{action: "deleted", err: ds.AlterClientQuotas(entity, nil)}
			},
		}
	}
}

// describeEntity renders a human-readable entity label for confirmation text.
func describeEntity(e api.ClientQuotaEntity) string {
	var parts []string
	if e.User != nil {
		parts = append(parts, "user="+quotaIDStr(e.User))
	}
	if e.ClientID != nil {
		parts = append(parts, "client-id="+quotaIDStr(e.ClientID))
	}
	if e.IP != nil {
		parts = append(parts, "ip="+quotaIDStr(e.IP))
	}
	return strings.Join(parts, " ")
}

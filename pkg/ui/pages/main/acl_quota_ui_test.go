package mainpage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/Benny93/kafui/pkg/ui/shared/aclcsv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- AQ-13: ACL row columns include pattern type + host ---

func TestACLResourceItem_GetValues(t *testing.T) {
	tests := []struct {
		name string
		item *ACLResourceItem
		want []string
	}{
		{
			"literal binding",
			&ACLResourceItem{principal: "User:alice", host: "*", resourceType: "Topic", resourceName: "orders", patternType: "Literal", operation: "Read", permission: "Allow"},
			[]string{"User:alice", "Topic:orders", "Literal", "*", "Read", "Allow"},
		},
		{
			"prefixed binding, empty pattern defaults to Literal, empty host to *",
			&ACLResourceItem{principal: "User:bob", resourceType: "Group", resourceName: "grp-", patternType: "Prefixed", operation: "Describe", permission: "Deny"},
			[]string{"User:bob", "Group:grp-", "Prefixed", "*", "Describe", "Deny"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.GetValues())
		})
	}
}

func TestACLResourceItem_GetDetails_IncludesPatternAndHost(t *testing.T) {
	item := &ACLResourceItem{principal: "User:alice", host: "10.0.0.1", resourceType: "Topic", resourceName: "orders", patternType: "Prefixed", operation: "Read", permission: "Allow"}
	d := item.GetDetails()
	assert.Equal(t, "Prefixed", d["PatternType"])
	assert.Equal(t, "10.0.0.1", d["Host"])
	assert.Equal(t, "Topic:orders", d["Resource"])
}

// --- AQ-19: quota row rendering including absent identifiers + sorted props ---

func TestQuotaResourceItem_GetValuesAndDetails(t *testing.T) {
	u := "alice"
	item := &QuotaResourceItem{entry: api.ClientQuotaEntry{
		Entity: api.ClientQuotaEntity{User: &u}, // clientID + ip absent
		Quotas: map[string]float64{"producer_byte_rate": 1048576, "consumer_byte_rate": 2097152},
	}}
	vals := item.GetValues()
	assert.Equal(t, []string{"alice", "<any>", "<any>", "consumer_byte_rate=2097152, producer_byte_rate=1048576"}, vals)

	// default (empty-string) identifier renders <default>.
	def := ""
	dItem := &QuotaResourceItem{entry: api.ClientQuotaEntry{Entity: api.ClientQuotaEntity{User: &def}, Quotas: map[string]float64{"consumer_byte_rate": 1024}}}
	assert.Equal(t, "<default>", dItem.GetValues()[0])

	d := item.GetDetails()
	assert.Equal(t, "alice", d["User"])
	assert.Equal(t, "<any>", d["ClientID"])
}

// --- test helpers ---

func newACLProvider(t *testing.T) (*KafuiContentProvider, *mock.KafkaDataSourceMock) {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	k := NewKafuiContentProvider(ds)
	k.switchResource(SwitchResourceMsg(ACLResourceType))
	return k, ds
}

// loadACLs loads the ACL list into the provider so GetSelectedResourceItem works.
func loadACLs(t *testing.T, k *KafuiContentProvider) {
	t.Helper()
	cmd := k.loadCurrentResource()
	msg := cmd()
	list, ok := msg.(CurrentResourceListMsg)
	require.True(t, ok)
	k.handleResourceList(list)
	k.pagination.SetTotalItems(len(k.allItems))
	k.updateTableForCurrentPage()
}

// --- AQ-16: custom create form composes the entry and calls CreateACL ---

func TestACLCreateForm_Custom_CallsCreateACL(t *testing.T) {
	k, ds := newACLProvider(t)
	cmd := k.handleACLFormSubmit(map[string]string{
		"type":          "Custom",
		"principal":     "User:carol",
		"host":          "*",
		"resource_type": "Topic",
		"resource_name": "payments",
		"pattern_type":  "Literal",
		"operation":     "Write",
		"permission":    "Allow",
	})
	msg := cmd()
	res, ok := msg.(aclCreatedMsg)
	require.True(t, ok)
	require.NoError(t, res.err)
	assert.Equal(t, 1, res.created)

	acls, _ := ds.GetACLs()
	found := false
	for _, a := range acls {
		if a.Principal == "User:carol" && a.ResourceName == "payments" && a.Operation == "Write" {
			found = true
		}
	}
	assert.True(t, found, "created binding should be listed")
}

func TestACLCreateForm_InvalidPrincipal_ReturnsError(t *testing.T) {
	k, _ := newACLProvider(t)
	cmd := k.handleACLFormSubmit(map[string]string{
		"type":          "Custom",
		"principal":     "no-colon",
		"resource_type": "Topic",
		"resource_name": "t",
		"pattern_type":  "Literal",
		"operation":     "Read",
		"permission":    "Allow",
	})
	res, ok := cmd().(aclCreatedMsg)
	require.True(t, ok)
	assert.Error(t, res.err)

	// Feeding the error message back keeps the form open for correction.
	k.showACLForm = true
	k.HandleContentUpdate(res)
	assert.True(t, k.showACLForm, "form stays open on validation error")
}

// --- AQ-17: consumer convenience flow expands the expected bindings ---

func TestACLConvenienceFlow_Consumer_CreatesExpandedBindings(t *testing.T) {
	k, ds := newACLProvider(t)
	cmd := k.handleACLFormSubmit(map[string]string{
		"type":      "Consumer",
		"principal": "User:svc",
		"host":      "*",
		"topics":    "orders",
		"groups":    "grp1",
	})
	res, ok := cmd().(aclCreatedMsg)
	require.True(t, ok)
	require.NoError(t, res.err)
	// READ+DESCRIBE on topic + READ+DESCRIBE on group = 4 bindings.
	assert.Equal(t, 4, res.created)

	want, err := api.ExpandConsumerACLs("User:svc", "*", []string{"orders"}, []string{"grp1"}, "", "")
	require.NoError(t, err)
	acls, _ := ds.GetACLs()
	for _, w := range want {
		assert.Contains(t, acls, w)
	}
}

// --- AQ-15: delete emits ShowConfirmMsg and calls DeleteACL once on confirm ---

func TestDeleteACL_ConfirmThenDelete(t *testing.T) {
	k, ds := newACLProvider(t)
	loadACLs(t, k)
	before, _ := ds.GetACLs()
	require.NotEmpty(t, before)
	target, _, ok := aclItemFrom(k.GetSelectedResourceItem())
	require.True(t, ok)
	entry := target.Entry()

	cmd := k.deleteSelectedACL()
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok, "delete must go through a confirmation")
	require.NotNil(t, confirm.OnConfirm)
	assert.True(t, confirm.Danger)

	// Esc/cancel path: not invoking OnConfirm makes no datasource call.
	afterCancel, _ := ds.GetACLs()
	assert.Len(t, afterCancel, len(before))

	// Confirm path deletes exactly the selected binding.
	res, ok := confirm.OnConfirm().(aclDeletedMsg)
	require.True(t, ok)
	require.NoError(t, res.err)
	after, _ := ds.GetACLs()
	assert.Len(t, after, len(before)-1)
	assert.NotContains(t, after, entry)
}

// --- AQ-18: CSV export golden + malformed import aborts + in-sync no-op ---

func TestExportACLs_MarshalGolden(t *testing.T) {
	entries := []api.ACLEntry{
		{Principal: "User:alice", Host: "*", ResourceType: "Topic", ResourceName: "orders", PatternType: "Literal", Operation: "Read", Permission: "Allow"},
	}
	got := aclcsv.Marshal(entries)
	want := "Principal,ResourceType,PatternType,ResourceName,Operation,PermissionType,Host\n" +
		"User:alice,Topic,Literal,orders,Read,Allow,*\n"
	assert.Equal(t, want, got)
}

func TestACLSync_MalformedFile_AbortsBeforeMutation(t *testing.T) {
	k, ds := newACLProvider(t)
	before, _ := ds.GetACLs()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.csv")
	require.NoError(t, os.WriteFile(path, []byte("User:alice,Topic\n"), 0o600)) // too few columns

	msg := k.handleACLSyncSubmit(path)()
	note, ok := msg.(core.NotificationMsg)
	require.True(t, ok)
	assert.Equal(t, core.StatusError, note.Severity)

	after, _ := ds.GetACLs()
	assert.Len(t, after, len(before), "malformed CSV must not mutate the cluster")
}

func TestACLSync_PlanConfirmAppliesChanges(t *testing.T) {
	k, ds := newACLProvider(t)
	// Desired = a brand-new binding on top of the seeded set → plan has a create.
	desired := "Principal,ResourceType,PatternType,ResourceName,Operation,PermissionType,Host\n" +
		"User:newbie,Topic,Literal,brand-new,Read,Allow,*\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "acls.csv")
	require.NoError(t, os.WriteFile(path, []byte(desired), 0o600))

	msg := k.handleACLSyncSubmit(path)()
	confirm, ok := msg.(core.ShowConfirmMsg)
	require.True(t, ok, "non-empty plan must be confirmed")
	require.NotNil(t, confirm.OnConfirm)

	res, ok := confirm.OnConfirm().(aclSyncedMsg)
	require.True(t, ok)
	require.NoError(t, res.err)
	assert.Equal(t, 1, res.created)

	acls, _ := ds.GetACLs()
	found := false
	for _, a := range acls {
		if a.ResourceName == "brand-new" {
			found = true
		}
	}
	assert.True(t, found)
}

// --- AQ-20: quota edit calls AlterClientQuotas; empty set gated by confirmation ---

func newQuotaProvider(t *testing.T) (*KafuiContentProvider, *mock.KafkaDataSourceMock) {
	t.Helper()
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")
	k := NewKafuiContentProvider(ds)
	k.switchResource(SwitchResourceMsg(QuotaResourceType))
	return k, ds
}

func TestQuotaForm_Create_CallsAlterClientQuotas(t *testing.T) {
	k, ds := newQuotaProvider(t)
	k.quotaEditEntity = nil // create mode
	cmd := k.handleQuotaFormSubmit(map[string]string{
		"user":      "dave",
		"client_id": "",
		"ip":        "",
		"quotas":    "producer_byte_rate=1000, consumer_byte_rate=2000",
	})
	res, ok := cmd().(quotaAlteredMsg)
	require.True(t, ok)
	require.NoError(t, res.err)
	assert.Equal(t, "created", res.action)

	quotas, _ := ds.GetClientQuotas()
	found := false
	for _, q := range quotas {
		if q.Entity.User != nil && *q.Entity.User == "dave" {
			found = true
			assert.Equal(t, float64(1000), q.Quotas["producer_byte_rate"])
			assert.Equal(t, float64(2000), q.Quotas["consumer_byte_rate"])
		}
	}
	assert.True(t, found)
}

func TestQuotaForm_MissingEntity_ReturnsValidationError(t *testing.T) {
	k, _ := newQuotaProvider(t)
	k.quotaEditEntity = nil
	cmd := k.handleQuotaFormSubmit(map[string]string{"user": "", "client_id": "", "ip": "", "quotas": "x=1"})
	res := cmd().(quotaAlteredMsg)
	assert.Error(t, res.err)
}

func TestQuotaForm_EmptyProps_GatedByConfirmation(t *testing.T) {
	k, ds := newQuotaProvider(t)
	k.quotaEditEntity = nil
	cmd := k.handleQuotaFormSubmit(map[string]string{"user": "alice", "quotas": ""})
	confirm, ok := cmd().(core.ShowConfirmMsg)
	require.True(t, ok, "empty (delete) submission must be confirmed")
	assert.True(t, confirm.Danger)

	res, ok := confirm.OnConfirm().(quotaAlteredMsg)
	require.True(t, ok)
	require.NoError(t, res.err)
	assert.Equal(t, "deleted", res.action)

	// alice's seeded quota should be gone.
	quotas, _ := ds.GetClientQuotas()
	for _, q := range quotas {
		if q.Entity.User != nil && *q.Entity.User == "alice" {
			t.Fatal("alice's quota should have been deleted")
		}
	}
}

// --- AQ-21: ACL edit actions gated on the CapACLEdit capability ---

// capSpyDS lets a test control the advertised cluster capabilities.
type capSpyDS struct {
	*mock.KafkaDataSourceMock
	caps []api.Capability
}

func (s *capSpyDS) GetClusterCapabilities(_ context.Context, _ string) ([]api.Capability, error) {
	return s.caps, nil
}

func newGatedProvider(t *testing.T, caps []api.Capability) *KafuiContentProvider {
	t.Helper()
	ds := &capSpyDS{KafkaDataSourceMock: &mock.KafkaDataSourceMock{}, caps: caps}
	ds.Init("")
	col := cluster.New(ds, time.Minute, nil)
	col.CollectAll(context.Background())
	common := &core.Common{
		DataSource: ds,
		Styles:     nil,
		Collector:  col,
		Config:     &core.UIConfig{},
	}
	k := NewKafuiContentProviderWithCommon(common)
	k.switchResource(SwitchResourceMsg(ACLResourceType))
	return k
}

func TestACLEditGating(t *testing.T) {
	t.Run("edit allowed when CapACLEdit present", func(t *testing.T) {
		k := newGatedProvider(t, []api.Capability{api.CapACLView, api.CapACLEdit})
		assert.True(t, k.canEditACL())
		cmd := k.openCreateACLForm()
		assert.True(t, k.showACLForm, "form opens when editing is allowed")
		_ = cmd
	})

	t.Run("edit hidden when CapACLEdit absent", func(t *testing.T) {
		k := newGatedProvider(t, []api.Capability{api.CapACLView})
		assert.False(t, k.canEditACL())

		// Create form does not open; a hint notification is returned instead.
		cmd := k.openCreateACLForm()
		assert.False(t, k.showACLForm, "form must not open when editing is disabled")
		note, ok := cmd().(core.NotificationMsg)
		require.True(t, ok)
		assert.Equal(t, core.StatusInfo, note.Severity)

		// Delete is likewise gated (no confirmation dialog emitted).
		loadACLs(t, k)
		dcmd := k.deleteSelectedACL()
		_, isConfirm := dcmd().(core.ShowConfirmMsg)
		assert.False(t, isConfirm, "delete must be gated when editing is disabled")
	})
}

package consumergroup

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func i64(v int64) *int64 { return &v }

// withWd runs fn inside dir, restoring the previous working directory after.
// exportCSV writes a relative filename, so the CWD is what the test controls.
func withWd(t *testing.T, dir string, fn func()) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(wd) })
	fn()
}

// With nothing displayed there is nothing to export.
func TestExportCSV_NoRowsIsNoop(t *testing.T) {
	m := newPage(t, newSpy(), "grp")
	m.topicRows = nil
	assert.Nil(t, m.exportCSV(), "no visible rows means no export command")
}

// A populated table writes an aggregate row plus one line per partition, and
// the group id is sanitized into the filename.
func TestExportCSV_WritesRowsAndSanitizesName(t *testing.T) {
	m := newPage(t, newSpy(), "orders/beta: prod 1")
	m.topicRows = []topicRow{{
		topic:  "t1",
		aggLag: i64(42),
		partitions: []api.PartitionOffset{
			{Partition: 0, CommittedOffset: i64(10), EndOffset: 20, Lag: i64(10), MemberID: "m1", MemberHost: "h1"},
			{Partition: 1, CommittedOffset: nil, EndOffset: 5, Lag: nil, MemberID: "", MemberHost: ""},
		},
	}}

	var msg = func() any {
		var out any
		withWd(t, t.TempDir(), func() { out = m.exportCSV()() })
		return out
	}()

	n, ok := msg.(core.NotificationMsg)
	require.True(t, ok, "a successful export reports a notification")
	assert.Equal(t, core.StatusInfo, n.Severity)
	assert.Equal(t, "Group exported", n.Title)

	// Every path-hostile character in the group id became an underscore.
	base := filepath.Base(n.Message)
	assert.Contains(t, base, "consumer-group-orders_beta__prod_1-")
	assert.True(t, filepath.IsAbs(n.Message), "the status shows the absolute path")

	f, err := os.Open(n.Message)
	require.NoError(t, err)
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 4, "header + aggregate row + two partition rows")
	assert.Equal(t, []string{"Topic", "Partition", "Committed", "End", "Lag", "Consumer", "Host"}, records[0])
	// Aggregate row: only topic and summed lag are filled.
	assert.Equal(t, []string{"t1", "", "", "", "42", "", ""}, records[1])
	assert.Equal(t, []string{"t1", "0", "10", "20", "10", "m1", "h1"}, records[2])
	// Uncommitted / lagless partition: empty committed, em-dash lag.
	assert.Equal(t, []string{"t1", "1", "", "5", "—", "", ""}, records[3])
}

// A group id with no hostile characters is passed through untouched.
func TestSanitizeLeavesSafeNames(t *testing.T) {
	assert.Equal(t, "order-processor_v2", sanitize("order-processor_v2"))
	assert.Equal(t, "a_b_c_d", sanitize(`a/b\c:d`))
}

// When the file cannot be created the command surfaces an error notification
// rather than a panic or a silent success.
func TestExportCSV_WriteFailure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o0500)) // read + execute, no write
	t.Cleanup(func() { _ = os.Chmod(dir, 0o0700) })

	m := newPage(t, newSpy(), "grp")
	m.topicRows = []topicRow{{topic: "t1"}}

	withWd(t, dir, func() {
		// A process that bypasses the permission bits (root) cannot exercise
		// the failure path; skip rather than assert a false negative.
		if f, err := os.Create("probe"); err == nil {
			_ = f.Close()
			_ = os.Remove("probe")
			t.Skip("cannot force a write failure in this environment")
		}

		msg := m.exportCSV()()
		n, ok := msg.(core.NotificationMsg)
		require.True(t, ok)
		assert.Equal(t, core.StatusError, n.Severity)
		assert.Equal(t, "CSV export failed", n.Title)
	})
}

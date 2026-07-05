package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Result
	}{
		{"nil is success", nil, ResultSuccess},
		{"access denied", api.AccessDeniedError{Resource: "topic", Action: "delete"}, ResultAccessDenied},
		{"read-only is denied", api.ClusterReadOnlyError{Cluster: "prod"}, ResultAccessDenied},
		{"acl validation", api.ACLValidationError{Field: "principal", Reason: "x"}, ResultValidationError},
		{"topic validation", api.TopicValidationError{TopicName: "t", Reason: "bad"}, ResultValidationError},
		{"generic is execution error", api.TopicNotFoundError{TopicName: "t"}, ResultExecutionError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Classify(tt.err))
		})
	}
}

func TestFileWriterAppendsJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "audit.log")
	w, err := NewFileWriter(path)
	require.NoError(t, err)

	require.NoError(t, w.Write(Record{User: "alice", Operation: "DeleteTopic", Result: ResultSuccess}))
	require.NoError(t, w.Write(Record{User: "alice", Operation: "CreateTopic", Result: ResultAccessDenied}))
	require.NoError(t, w.Close())

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var lines []Record
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r Record
		require.NoError(t, json.Unmarshal(sc.Bytes(), &r))
		lines = append(lines, r)
	}
	require.Len(t, lines, 2, "one JSON line per record")
	assert.Equal(t, "DeleteTopic", lines[0].Operation)
	assert.Equal(t, ResultAccessDenied, lines[1].Result)
}

// countingWriter records how many times Write was called and can fail.
type countingWriter struct {
	n       int
	fail    bool
	records []Record
}

func (c *countingWriter) Write(r Record) error {
	c.n++
	c.records = append(c.records, r)
	if c.fail {
		return assert.AnError
	}
	return nil
}

func alterRecord() Record {
	return Record{Operation: "DeleteTopic", Resources: []Resource{{Type: "topic", Alter: true, Actions: []string{"delete"}}}}
}

func readRecord() Record {
	return Record{Operation: "GetTopics", Resources: []Resource{{Type: "topic", Alter: false, Actions: []string{"view"}}}}
}

func TestServiceDisabledWritesNothing(t *testing.T) {
	cw := &countingWriter{}
	s := NewService(false, LevelAll, cw, nil)
	s.Record(alterRecord())
	assert.Equal(t, 0, cw.n)
	assert.False(t, s.Enabled())
}

func TestServiceLevelFiltering(t *testing.T) {
	t.Run("alter_only skips reads", func(t *testing.T) {
		cw := &countingWriter{}
		s := NewService(true, LevelAlterOnly, cw, nil)
		s.Record(readRecord())
		assert.Equal(t, 0, cw.n, "read-only op skipped at alter_only")
		s.Record(alterRecord())
		assert.Equal(t, 1, cw.n, "altering op recorded")
	})
	t.Run("all records reads too", func(t *testing.T) {
		cw := &countingWriter{}
		s := NewService(true, LevelAll, cw, nil)
		s.Record(readRecord())
		s.Record(alterRecord())
		assert.Equal(t, 2, cw.n)
	})
}

func TestServiceStampsTimestampAndUser(t *testing.T) {
	cw := &countingWriter{}
	s := NewService(true, LevelAll, cw, nil)
	s.Record(alterRecord())
	require.Len(t, cw.records, 1)
	assert.NotEmpty(t, cw.records[0].Timestamp)
	assert.NotEmpty(t, cw.records[0].User)
}

func TestServiceWriteFailureDoesNotPropagate(t *testing.T) {
	cw := &countingWriter{fail: true}
	s := NewService(true, LevelAll, cw, nil)
	// Must not panic or block; there's no error to observe by contract.
	s.Record(alterRecord())
	assert.Equal(t, 1, cw.n)
}

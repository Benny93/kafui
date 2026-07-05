package cmd

import (
	"encoding/csv"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/datasource/mock"
)

// captureStdout runs fn while capturing anything written to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestRunGetBrokersCSV(t *testing.T) {
	ds := &mock.KafkaDataSourceMock{}
	ds.Init("")

	out := captureStdout(t, func() {
		if err := runGetBrokersCSV(ds); err != nil {
			t.Fatalf("runGetBrokersCSV: %v", err)
		}
	})

	records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	// header + 3 brokers
	if len(records) != 4 {
		t.Fatalf("records = %d, want 4", len(records))
	}
	if records[0][0] != "ID" {
		t.Errorf("header[0] = %q, want ID", records[0][0])
	}
	if !strings.Contains(records[1][0], "(Active)") {
		t.Errorf("broker 1 ID = %q, want controller annotation", records[1][0])
	}
	if records[3][4] != "N/A" {
		t.Errorf("broker 3 disk = %q, want N/A", records[3][4])
	}
}

func TestGetBrokersCommand_UnsupportedFormat(t *testing.T) {
	cmd := newGetBrokersCommand()
	cmd.SetArgs([]string{"--mock", "--format", "json"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for unsupported format")
	}
}

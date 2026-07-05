package shared

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
)

// TopicCSVRow is one topic's exported row. MessageCount/Size are negative when
// unknown (rendered "N/A").
type TopicCSVRow struct {
	Name              string
	Partitions        int32
	ReplicationFactor int16
	MessageCount      int64
	OutOfSync         int
	Size              int64
	Internal          bool
}

// TopicCSVHeader is the column header row written by WriteTopicCSV.
var TopicCSVHeader = []string{
	"Name", "Partitions", "Replication Factor", "Message Count", "Out Of Sync", "Size", "Internal",
}

// WriteTopicCSV writes topics as CSV to w. Rows are ordered by name for
// deterministic output. Unknown message counts/sizes (negative) render "N/A";
// sizes are formatted human-readably via FormatBytes2dp.
func WriteTopicCSV(w io.Writer, topics []TopicCSVRow) error {
	sorted := make([]TopicCSVRow, len(topics))
	copy(sorted, topics)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	cw := csv.NewWriter(w)
	if err := cw.Write(TopicCSVHeader); err != nil {
		return err
	}
	for _, t := range sorted {
		msgCount := "N/A"
		if t.MessageCount >= 0 {
			msgCount = strconv.FormatInt(t.MessageCount, 10)
		}
		size := "N/A"
		if t.Size >= 0 {
			size = FormatBytes2dp(t.Size)
		}
		row := []string{
			t.Name,
			strconv.FormatInt(int64(t.Partitions), 10),
			strconv.FormatInt(int64(t.ReplicationFactor), 10),
			msgCount,
			strconv.Itoa(t.OutOfSync),
			size,
			strconv.FormatBool(t.Internal),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

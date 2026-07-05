package shared

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"

	"github.com/Benny93/kafui/pkg/api"
)

// BrokerCSVHeader is the column header row written by WriteBrokerCSV. It is
// shared so the in-app export and the `kafui get brokers --format csv`
// subcommand always agree on layout.
var BrokerCSVHeader = []string{
	"ID", "Host", "Port", "Rack", "Disk Usage", "Leaders", "Replicas", "ISR", "Leader Skew", "Replica Skew",
}

// WriteBrokerCSV writes brokers as CSV to w, enriched with stats when available.
// The ID column is annotated "(Active)" for the controller. Disk usage and skew
// columns reuse the shared list-cell formatters so UI and CSV output match.
// Rows are ordered by broker ID for deterministic output.
func WriteBrokerCSV(w io.Writer, brokers []api.BrokerInfo, stats map[int32]api.BrokerStats) error {
	sorted := make([]api.BrokerInfo, len(brokers))
	copy(sorted, brokers)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	cw := csv.NewWriter(w)
	if err := cw.Write(BrokerCSVHeader); err != nil {
		return err
	}
	for _, b := range sorted {
		id := strconv.FormatInt(int64(b.ID), 10)
		if b.IsController {
			id += " (Active)"
		}
		disk, leaders, replicas, isr, leaderSkew, replicaSkew := "N/A", "", "", "", "-", "-"
		if s, ok := stats[b.ID]; ok {
			disk = FormatDiskUsage(s.SegmentSize, s.SegmentCount)
			leaders = strconv.Itoa(s.LeaderCount)
			replicas = strconv.Itoa(s.ReplicaCount)
			isr, _ = FormatISR(s.InSyncReplicaCount, s.ReplicaCount)
			leaderSkew = FormatSkew(s.LeaderSkew)
			replicaSkew = FormatSkew(s.ReplicaSkew)
		}
		row := []string{
			id, b.Host, strconv.FormatInt(int64(b.Port), 10), b.Rack,
			disk, leaders, replicas, isr, leaderSkew, replicaSkew,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

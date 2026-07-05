package shared

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"

	"github.com/Benny93/kafui/pkg/api"
)

// ConsumerGroupCSVHeader is the column header row written by WriteConsumerGroupCSV.
var ConsumerGroupCSVHeader = []string{
	"Group ID", "Members", "Topics", "Lag", "Coordinator", "State",
}

// WriteConsumerGroupCSV writes consumer groups as CSV to w. Undefined lag is
// written as an empty cell (never 0); unknown coordinator (<0) is empty too.
// Rows are ordered by group id for deterministic output.
func WriteConsumerGroupCSV(w io.Writer, groups []api.ConsumerGroup) error {
	sorted := make([]api.ConsumerGroup, len(groups))
	copy(sorted, groups)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	cw := csv.NewWriter(w)
	if err := cw.Write(ConsumerGroupCSVHeader); err != nil {
		return err
	}
	for _, g := range sorted {
		lag := ""
		if g.Lag != nil {
			lag = strconv.FormatInt(*g.Lag, 10)
		}
		coord := ""
		if g.CoordinatorID >= 0 {
			coord = strconv.FormatInt(int64(g.CoordinatorID), 10)
		}
		row := []string{
			g.Name,
			strconv.Itoa(g.MemberCount),
			strconv.Itoa(g.TopicCount),
			lag,
			coord,
			g.State,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

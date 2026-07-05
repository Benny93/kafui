package cmd

import (
	"fmt"
	"os"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/spf13/cobra"
)

// newGetCommand adds `kafui get …`: non-interactive, machine-readable listings.
// Currently only `get brokers --format csv` is implemented (the CLI adaptation
// of the brokers machine-readable endpoint).
func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Non-interactive resource listings (machine-readable)",
	}
	cmd.AddCommand(newGetBrokersCommand())
	return cmd
}

func newGetBrokersCommand() *cobra.Command {
	var useMock bool
	var format string
	cmd := &cobra.Command{
		Use:   "brokers",
		Short: "List brokers (with stats) to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "csv" {
				return fmt.Errorf("unsupported format %q (only csv)", format)
			}
			ds, err := newHealthDataSource(useMock)
			if err != nil {
				return err
			}
			return runGetBrokersCSV(ds)
		},
	}
	cmd.Flags().BoolVar(&useMock, "mock", false, "use the mock datasource")
	cmd.Flags().StringVar(&format, "format", "csv", "output format (csv)")
	return cmd
}

// runGetBrokersCSV writes the broker list, enriched with stats, as CSV to stdout
// using the same writer the in-app export uses so both agree.
func runGetBrokersCSV(ds api.KafkaDataSource) error {
	brokers, err := ds.GetBrokers()
	if err != nil {
		return fmt.Errorf("listing brokers: %w", err)
	}
	stats, _, err := ds.GetBrokerStats()
	if err != nil {
		// Stats are best-effort; still emit the broker list.
		stats = map[int32]api.BrokerStats{}
	}
	return shared.WriteBrokerCSV(os.Stdout, brokers, stats)
}

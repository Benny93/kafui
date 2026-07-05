package cmd

import (
	"fmt"
	"os"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/spf13/cobra"
)

// newHealthCommand adds `kafui health`: a liveness/connectivity probe that exits
// 0 when all probed services are reachable, 1 otherwise. This is the CLI-world
// equivalent of a health endpoint.
func newHealthCommand() *cobra.Command {
	var useMock bool
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Probe cluster (and schema registry) connectivity; exit 0 if healthy",
		Run: func(cmd *cobra.Command, args []string) {
			ds, err := newHealthDataSource(useMock)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			os.Exit(runHealth(ds))
		},
	}
	cmd.Flags().BoolVar(&useMock, "mock", false, "probe the mock datasource")
	return cmd
}

// newHealthDataSource builds a datasource for a one-shot probe (not the TUI).
// It returns a ClusterNotFoundError if --cluster/-c names an unknown context,
// instead of silently falling back to localhost:9092.
func newHealthDataSource(useMock bool) (api.KafkaDataSource, error) {
	if useMock {
		ds := &mock.KafkaDataSourceMock{}
		ds.Init("")
		return ds, nil
	}
	kafds.SetOverrides(brokersFlag, schemaRegistryURL, clusterFlag, verboseFlag)
	ds := kafds.NewKafkaDataSourceKaf()
	ds.Init(cfgFile)
	if err := api.ValidateClusterOverride(ds, clusterFlag); err != nil {
		return nil, err
	}
	return ds, nil
}

// runHealth probes broker (via a metadata listing) and, when configured, the
// schema registry. It prints one OK/FAIL line per service and returns an exit code.
func runHealth(ds api.KafkaDataSource) int {
	failed := false

	// Broker connectivity: a topic-names listing is a lightweight metadata request.
	if _, err := ds.GetTopicNames(); err != nil {
		fmt.Printf("broker           FAIL  %v\n", err)
		failed = true
	} else {
		fmt.Println("broker           OK")
	}

	// Schema registry, only when configured for the active cluster.
	if info, err := ds.GetClusterDetails(ds.GetContext()); err == nil && info.SchemaRegistryURL != "" {
		if _, err := ds.GetSchemas(); err != nil {
			fmt.Printf("schema-registry  FAIL  %v\n", err)
			failed = true
		} else {
			fmt.Println("schema-registry  OK")
		}
	}

	if failed {
		return 1
	}
	return 0
}

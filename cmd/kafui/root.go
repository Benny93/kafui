package cmd

import (
	"fmt"
	"os"

	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	cfgFile           string
	brokersFlag       []string
	schemaRegistryURL string
	clusterFlag       string
	verboseFlag       bool
	readOnlyFlag      bool
	topicFlag         string
	resourceFlag      string
	metricsListenFlag string
)

// dynamicConfigEnabled reports whether the kafui config file enables in-UI editing.
func dynamicConfigEnabled() bool {
	cfg, err := appconfig.Load(appconfig.DefaultPath())
	if err != nil {
		return false
	}
	return cfg.DynamicConfigEnabled
}

// KafuiInitFunc is a function type for kafui initialization
type KafuiInitFunc func(opts ui.InitOptions)

// defaultKafuiInit is the default implementation
var defaultKafuiInit KafuiInitFunc = ui.Init

// OsExit is os.Exit, swappable in tests: a real exit call kills the test
// binary outright (Go's testing package cannot recover from it), so tests
// that need to exercise the error path replace this with a panic instead.
var OsExit = os.Exit

// CreateRootCommand creates and returns the root cobra command
func CreateRootCommand(initFunc KafuiInitFunc) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kafui",
		Short: "k9s style kafka explorer",
		Long:  "Explore different kafka broker in a k9s fashion with quick switches between topics, consumer groups and brokers",
		Run: func(cmd *cobra.Command, args []string) {
			mock, _ := cmd.Flags().GetBool("mock")
			readOnly, _ := cmd.Flags().GetBool("read-only")
			initFunc(ui.InitOptions{
				ConfigFile:     cfgFile,
				Mock:           mock,
				Brokers:        brokersFlag,
				SchemaRegistry: schemaRegistryURL,
				Cluster:        clusterFlag,
				Verbose:        verboseFlag,
				ReadOnly:       readOnly,
				Topic:          topicFlag,
				Resource:       resourceFlag,
				MetricsListen:  metricsListenFlag,
			})
		},
	}

	// Add flags to root command
	rootCmd.PersistentFlags().Bool("mock", false, "Enable mock mode: Display mock data to test various functions without a real kafka broker")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kaf/config)")
	rootCmd.PersistentFlags().StringSliceVarP(&brokersFlag, "brokers", "b", nil, "Comma-separated list of broker host:port pairs (overrides config)")
	rootCmd.PersistentFlags().StringVar(&schemaRegistryURL, "schema-registry", "", "Schema registry URL (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&clusterFlag, "cluster", "c", "", "Set the active cluster/context by name")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose sarama logging")
	rootCmd.PersistentFlags().BoolVar(&readOnlyFlag, "read-only", false, "Treat every cluster as read-only: deny all altering operations")
	rootCmd.PersistentFlags().StringVar(&topicFlag, "topic", "", "Open the given topic directly on startup")
	rootCmd.PersistentFlags().StringVar(&resourceFlag, "resource", "", "Open the main page pre-switched to a resource (topics|consumer-groups|schemas|contexts|brokers|acls|quotas|connectors)")
	rootCmd.PersistentFlags().StringVar(&metricsListenFlag, "metrics-listen", "", "Serve the current metrics snapshot in Prometheus exposition format on this address (e.g. :9090); default off")

	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newHealthCommand())
	rootCmd.AddCommand(newGetCommand())

	// Errors are reported by DoExecute (once, without a stack trace or usage
	// dump); cobra's own printing is silenced to avoid a duplicate message.
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	return rootCmd
}

func DoExecute() {
	rootCmd := CreateRootCommand(defaultKafuiInit)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		OsExit(1)
	}
}

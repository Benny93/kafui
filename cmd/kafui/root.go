package cmd

import (
	"github.com/Benny93/kafui/pkg/kafui"
	"github.com/spf13/cobra"
)

var cfgFile string

// KafuiInitFunc is a function type for kafui initialization
type KafuiInitFunc func(configFile string, mock bool)

// defaultKafuiInit is the default implementation
var defaultKafuiInit KafuiInitFunc = kafui.Init

// CreateRootCommand creates and returns the root cobra command
func CreateRootCommand(initFunc KafuiInitFunc) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kafui",
		Short: "k9s style kafka explorer",
		Long:  "Explore different kafka broker in a k9s fashion with quick switches between topics, consumer groups and brokers",
		Run: func(cmd *cobra.Command, args []string) {
			mock, _ := cmd.Flags().GetBool("mock")
			initFunc(cfgFile, mock)
		},
	}

	// Add flags to root command
	rootCmd.PersistentFlags().Bool("mock", false, "Enable mock mode: Display mock data to test various functions without a real kafka broker")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kaf/config)")
	
	return rootCmd
}

func DoExecute() {
	rootCmd := CreateRootCommand(defaultKafuiInit)
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

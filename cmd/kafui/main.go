package main

import (
	"com/emptystate/kafui/pkg/kafui"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kafui",
		Short: "k9s style kafka explorer",
		Long:  "Explore different kafka broker in a k9s fashion with quick switches between topics, consumer groups and brokers",
		Run: func(cmd *cobra.Command, args []string) {
			mock, _ := cmd.Flags().GetBool("mock")
			kafui.Init(mock)
		},
	}

	// Add flags to root command
	rootCmd.PersistentFlags().Bool("mock", false, "Enable mock mode: Display mock data to test various functions without a real kafka broker")

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

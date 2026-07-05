package cmd

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/version"
	"github.com/spf13/cobra"
)

// newVersionCommand prints build metadata and the enabled feature list.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kafui version and build information",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.Get(enabledFeatures()...)
			fmt.Println(info.String())
			if len(info.Features) > 0 {
				fmt.Printf("features: %v\n", info.Features)
			}
		},
	}
}

// enabledFeatures reports compile/config feature flags advertised by `version`.
// The dynamic-config flag is resolved at runtime from the kafui config file.
func enabledFeatures() []string {
	feats := []string{}
	if dynamicConfigEnabled() {
		feats = append(feats, "dynamic-config")
	}
	return feats
}

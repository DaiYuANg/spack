package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var runtime *fx.App

var rootCmd = &cobra.Command{
	Use: "sproxy",
	PreRun: func(cmd *cobra.Command, args []string) {
		runtime = container()
	},
	Run: func(cmd *cobra.Command, args []string) {
		runtime.Run()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

package cmd

import (
	"github.com/daiyuang/spack/internal/cache"
	"github.com/daiyuang/spack/internal/printer"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var printCmdRuntime *fx.App

var printCmd = &cobra.Command{
	Use: "print",
	PreRun: func(cmd *cobra.Command, args []string) {
		printCmdRuntime = createContainer(
			registry.Module,
			cache.Module,
			printer.Module,
		)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return printCmdRuntime.Start(cmd.Context())
	},
}

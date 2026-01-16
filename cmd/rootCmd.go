package cmd

import (
	"github.com/daiyuang/spack/internal/cache"
	"github.com/daiyuang/spack/internal/finder"
	"github.com/daiyuang/spack/internal/http"
	"github.com/daiyuang/spack/internal/metrics"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var container *fx.App

var rootCmd = &cobra.Command{
	Use: "spack",
	PreRun: func(cmd *cobra.Command, args []string) {
		container = createContainer(
			cache.Module,
			registry.Module,
			scanner.Module,
			metrics.Module,
			finder.Module,
			http.Module,
		)
	},
	Run: func(cmd *cobra.Command, args []string) {
		container.Run()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

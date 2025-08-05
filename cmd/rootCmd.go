package cmd

import (
	"github.com/daiyuang/spack/internal/cache"
	"github.com/daiyuang/spack/internal/http"
	"github.com/daiyuang/spack/internal/metrics"
	"github.com/daiyuang/spack/internal/preprocessor"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var runtime *fx.App

var rootCmd = &cobra.Command{
	Use: "spack",
	PreRun: func(cmd *cobra.Command, args []string) {
		runtime = createContainer(
			cache.Module,
			registry.Module,
			preprocessor.Module,
			http.Module,
			metrics.Module,
		)
	},
	Run: func(cmd *cobra.Command, args []string) {
		runtime.Run()
	},
}

func init() {
	rootCmd.AddCommand(printCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

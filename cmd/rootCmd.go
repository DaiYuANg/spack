package cmd

import (
	"fmt"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/event"
	"github.com/daiyuang/spack/internal/metrics"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/daiyuang/spack/internal/server"
	"github.com/daiyuang/spack/internal/source"
	"github.com/spf13/cobra"
)

var container *dix.App

var rootCmd = &cobra.Command{
	Use: "spack",
	PreRun: func(cmd *cobra.Command, args []string) {
		container = createContainer(
			metrics.Module,
			event.Module,
			source.Module,
			artifact.Module,
			assetcache.Module,
			pipeline.Module,
			resolver.Module,
			server.Module,
		)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return container.Run()
	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("execute root command: %w", err)
	}
	return nil
}

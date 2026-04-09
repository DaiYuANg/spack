package metrics

import "runtime/debug"

// NewBuildInfoMetricsForTest exposes build-info metric construction with explicit build info.
func NewBuildInfoMetricsForTest(appName string, info *debug.BuildInfo) *BuildInfoMetrics {
	return newBuildInfoMetrics(appName, info)
}

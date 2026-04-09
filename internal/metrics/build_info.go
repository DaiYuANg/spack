package metrics

import (
	"runtime/debug"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type BuildInfoMetrics struct {
	collector prometheus.Collector
}

func NewBuildInfoMetrics(appName string) *BuildInfoMetrics {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = &debug.BuildInfo{}
	}
	return newBuildInfoMetrics(appName, info)
}

func newBuildInfoMetrics(appName string, info *debug.BuildInfo) *BuildInfoMetrics {
	if info == nil {
		info = &debug.BuildInfo{}
	}

	labels := prometheus.Labels{
		"app":          normalizeBuildInfoLabel(appName, "spack"),
		"version":      normalizeBuildInfoLabel(info.Main.Version, "unknown"),
		"go_version":   normalizeBuildInfoLabel(info.GoVersion, "unknown"),
		"module_path":  normalizeBuildInfoLabel(info.Path, "unknown"),
		"vcs_revision": "unknown",
		"vcs_time":     "unknown",
		"vcs_modified": "unknown",
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			labels["vcs_revision"] = normalizeBuildInfoLabel(setting.Value, "unknown")
		case "vcs.time":
			labels["vcs_time"] = normalizeBuildInfoLabel(setting.Value, "unknown")
		case "vcs.modified":
			labels["vcs_modified"] = normalizeBuildInfoLabel(setting.Value, "unknown")
		}
	}

	return &BuildInfoMetrics{
		collector: prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name:        "spack_build_info",
			Help:        "Build metadata for the current spack runtime.",
			ConstLabels: labels,
		}, func() float64 {
			return 1
		}),
	}
}

func (m *BuildInfoMetrics) Collectors() []prometheus.Collector {
	if m == nil || m.collector == nil {
		return nil
	}
	return []prometheus.Collector{m.collector}
}

func normalizeBuildInfoLabel(value, fallback string) string {
	clean := strings.TrimSpace(value)
	if clean == "" || clean == "(devel)" {
		return fallback
	}
	return clean
}

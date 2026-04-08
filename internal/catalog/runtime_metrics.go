package catalog

import "github.com/prometheus/client_golang/prometheus"

type RuntimeMetrics struct {
	AssetsCurrent      prometheus.Gauge
	VariantsCurrent    prometheus.Gauge
	SourceBytesCurrent prometheus.Gauge
}

func NewRuntimeMetrics() *RuntimeMetrics {
	return &RuntimeMetrics{
		AssetsCurrent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "spack_catalog_assets_current",
			Help: "Current number of assets tracked by the in-memory catalog",
		}),
		VariantsCurrent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "spack_catalog_variants_current",
			Help: "Current number of variants tracked by the in-memory catalog",
		}),
		SourceBytesCurrent: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "spack_catalog_source_bytes_current",
			Help: "Current total source bytes observed during the latest catalog scan",
		}),
	}
}

func (m *RuntimeMetrics) Collectors() []prometheus.Collector {
	if m == nil {
		return nil
	}
	return []prometheus.Collector{
		m.AssetsCurrent,
		m.VariantsCurrent,
		m.SourceBytesCurrent,
	}
}

func (m *RuntimeMetrics) SyncCatalog(cat Catalog) {
	if m == nil || cat == nil {
		return
	}
	m.AssetsCurrent.Set(float64(cat.AssetCount()))
	m.VariantsCurrent.Set(float64(cat.VariantCount()))
}

func (m *RuntimeMetrics) SetSourceBytes(totalBytes int64) {
	if m == nil {
		return
	}
	m.SourceBytesCurrent.Set(float64(totalBytes))
}

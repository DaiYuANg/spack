package cmd

import (
	"github.com/daiyuang/spack/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	configFiles   []string
	configFlagSet = newConfigFlagSet()
)

func init() {
	bindConfigFlags(rootCmd)
}

func bindConfigFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()
	flags.StringSliceVarP(&configFiles, "config", "c", nil, "Config file path(s). Later files override earlier ones.")
	flags.AddFlagSet(configFlagSet)
}

func configLoadOptions() config.LoadOptions {
	return config.LoadOptions{
		Files:   append([]string(nil), configFiles...),
		FlagSet: configFlagSet,
	}
}

func newConfigFlagSet() *pflag.FlagSet {
	defaults := config.DefaultConfig()
	flags := pflag.NewFlagSet("config", pflag.ContinueOnError)

	bindHTTPFlags(flags, defaults.HTTP)
	bindAssetFlags(flags, defaults.Assets)
	bindAsyncFlags(flags, defaults.Async)
	bindDebugFlags(flags, defaults.Debug)
	bindImageFlags(flags, defaults.Image)
	bindMetricsFlags(flags, defaults.Metrics)
	bindLoggerFlags(flags, defaults.Logger)
	bindRobotsFlags(flags, defaults.Robots)
	bindCompressionFlags(flags, defaults.Compression)

	return flags
}

func bindHTTPFlags(flags *pflag.FlagSet, defaults config.HTTP) {
	flags.Int("http.port", defaults.Port, "HTTP listen port.")
	flags.Bool("http.low_memory", defaults.LowMemory, "Reduce Fiber memory usage.")
	flags.Bool("http.memory_cache.enable", defaults.MemoryCache.Enable, "Enable in-memory asset cache.")
	flags.Bool("http.memory_cache.warmup", defaults.MemoryCache.Warmup, "Preload in-memory asset cache at startup.")
	flags.Int("http.memory_cache.max_entries", defaults.MemoryCache.MaxEntries, "Maximum number of in-memory asset cache entries.")
	flags.Int64("http.memory_cache.max_file_size", defaults.MemoryCache.MaxFileSize, "Maximum asset size in bytes eligible for in-memory cache.")
	flags.String("http.memory_cache.ttl", defaults.MemoryCache.TTL, "TTL for in-memory asset cache entries.")
}

func bindAssetFlags(flags *pflag.FlagSet, defaults config.Assets) {
	flags.String("assets.backend", string(defaults.NormalizedBackend()), "Asset source backend.")
	flags.String("assets.path", defaults.Path, "HTTP mount path for assets.")
	flags.String("assets.root", defaults.Root, "Filesystem root containing static assets.")
	flags.String("assets.entry", defaults.Entry, "Default entry file for directory requests.")
	flags.String("assets.fallback.on", string(defaults.Fallback.On), "Fallback trigger mode.")
	flags.String("assets.fallback.target", defaults.Fallback.Target, "Fallback asset path.")
}

func bindAsyncFlags(flags *pflag.FlagSet, defaults config.Async) {
	flags.Int("async.workers", defaults.NormalizedWorkers(), "Shared async worker pool size.")
}

func bindDebugFlags(flags *pflag.FlagSet, defaults config.Debug) {
	flags.Bool("debug.enable", defaults.Enable, "Enable debug HTTP runtime.")
	flags.String("debug.pprof_prefix", defaults.PprofPrefix, "Mount prefix for Fiber pprof handlers.")
	flags.Int("debug.live_port", defaults.LivePort, "Debug runtime listen port.")
}

func bindImageFlags(flags *pflag.FlagSet, defaults config.Image) {
	flags.Bool("image.enable", defaults.Enable, "Enable image variant pipeline.")
	flags.String("image.widths", defaults.Widths, "Comma-separated responsive image widths.")
	flags.String("image.formats", defaults.Formats, "Comma-separated additional image output formats for warmup and default generation.")
	flags.Int("image.jpeg_quality", defaults.JPEGQuality, "JPEG encoding quality for generated variants.")
}

func bindMetricsFlags(flags *pflag.FlagSet, defaults config.Metrics) {
	flags.String("metrics.prefix", defaults.Prefix, "Metrics endpoint path.")
}

func bindLoggerFlags(flags *pflag.FlagSet, defaults config.Logger) {
	flags.String("logger.level", defaults.Level, "Logger level.")
	flags.Bool("logger.console.enabled", defaults.Console.Enabled, "Enable console logging.")
	flags.Bool("logger.file.enabled", defaults.File.Enabled, "Enable file logging.")
	flags.String("logger.file.path", defaults.File.Path, "Log file path.")
	flags.Int("logger.file.max_size", defaults.File.MaxSize, "Maximum log file size before rotation.")
	flags.Int("logger.file.max_age", defaults.File.MaxAge, "Maximum age in days for rotated log files.")
	flags.Int("logger.file.max_files", defaults.File.MaxFiles, "Maximum number of rotated log files to retain.")
}

func bindRobotsFlags(flags *pflag.FlagSet, defaults config.Robots) {
	flags.Bool("robots.enable", defaults.Enable, "Enable built-in robots.txt route generation.")
	flags.Bool("robots.override", defaults.Override, "Prefer generated robots.txt over a scanned robots.txt asset.")
	flags.String("robots.user_agent", defaults.UserAgent, "Generated robots.txt User-agent value.")
	flags.String("robots.allow", defaults.Allow, "Generated robots.txt Allow value.")
	flags.String("robots.disallow", defaults.Disallow, "Generated robots.txt Disallow value.")
	flags.String("robots.sitemap", defaults.Sitemap, "Generated robots.txt Sitemap value.")
	flags.String("robots.host", defaults.Host, "Generated robots.txt Host value.")
}

func bindCompressionFlags(flags *pflag.FlagSet, defaults config.Compression) {
	flags.Bool("compression.enable", defaults.Enable, "Enable compression pipeline.")
	flags.String("compression.mode", defaults.Mode, "Compression mode: off, lazy, or warmup.")
	flags.String("compression.cache_dir", defaults.CacheDir, "Compression artifact cache directory.")
	flags.Int64("compression.min_size", defaults.MinSize, "Minimum asset size in bytes eligible for compression.")
	flags.Int("compression.workers", defaults.Workers, "Compression worker count.")
	flags.Int("compression.queue_size", defaults.QueueSize, "Compression queue capacity.")
	flags.String("compression.encodings", defaults.Encodings, "Comma-separated supported compression encodings in preference order.")
	flags.String("compression.cleanup_every", defaults.CleanupEvery, "Compression cache cleanup interval.")
	flags.String("compression.max_age", defaults.MaxAge, "Default cache max-age for compressed responses.")
	flags.String("compression.image_max_age", defaults.ImageMaxAge, "Cache max-age for generated image variants.")
	flags.String("compression.encoding_max_age", defaults.EncodingMaxAge, "Cache max-age for precompressed variants.")
	flags.Int64("compression.max_cache_bytes", defaults.MaxCacheBytes, "Maximum bytes allowed in compression cache.")
	flags.Int("compression.brotli_quality", defaults.BrotliQuality, "Brotli compression quality.")
	flags.Int("compression.zstd_level", defaults.ZstdLevel, "Zstd compression level.")
	flags.Int("compression.gzip_level", defaults.GzipLevel, "Gzip compression level.")
}

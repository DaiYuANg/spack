package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/daiyuang/spack/internal/config"
	"github.com/spf13/pflag"
)

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if !ok {
			if err := os.Unsetenv(key); err != nil {
				t.Fatalf("cleanup unset %s: %v", key, err)
			}
			return
		}
		t.Setenv(key, value)
	})
}

func TestLoadIntoDefaultConfigPreservesNestedDefaultsWithPartialDotenv(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	envBody := "APP_ASSETS_ROOT=/tmp/assets\nAPP_ASSETS_PATH=/\nAPP_COMPRESSION_ENABLE=true\n"
	if err := os.WriteFile(envPath, []byte(envBody), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfigForTest()
	if err := configx.Load(
		&cfg,
		configx.WithEnvPrefix("APP"),
		configx.WithIgnoreDotenvError(false),
		configx.WithDotenv(envPath),
	); err != nil {
		t.Fatal(err)
	}

	if cfg.Assets.Entry != "index.html" {
		t.Fatalf("expected default assets entry to be preserved, got %q", cfg.Assets.Entry)
	}
	if cfg.Assets.NormalizedBackend() != config.SourceBackendLocal {
		t.Fatalf("expected default assets backend %q, got %q", config.SourceBackendLocal, cfg.Assets.NormalizedBackend())
	}
	if cfg.Assets.Fallback.On != config.FallbackOnNotFound {
		t.Fatalf("expected default fallback mode %q, got %q", config.FallbackOnNotFound, cfg.Assets.Fallback.On)
	}
	if cfg.Debug.LivePort != 8080 {
		t.Fatalf("expected default debug live port 8080, got %d", cfg.Debug.LivePort)
	}
	if cfg.Compression.CacheDir == "" {
		t.Fatal("expected default compression cache dir to be preserved")
	}
	if cfg.Compression.Workers != 2 {
		t.Fatalf("expected default compression workers 2, got %d", cfg.Compression.Workers)
	}
}

func TestLoadWithOptions_PrioritizesFlagsOverEnvOverFiles(t *testing.T) {
	t.Helper()

	unsetEnvForTest(t, "SPACK_HTTP_PORT")
	unsetEnvForTest(t, "SPACK_HTTP_LOW_MEMORY")
	unsetEnvForTest(t, "SPACK_ASSETS_PATH")
	unsetEnvForTest(t, "SPACK_ASSETS_BACKEND")
	unsetEnvForTest(t, "SPACK_LOGGER_LEVEL")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "spack.yaml")
	configBody := "" +
		"http:\n" +
		"  port: 7001\n" +
		"assets:\n" +
		"  path: /from-file\n" +
		"  root: /file-root\n" +
		"logger:\n" +
		"  level: warn\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SPACK_ASSETS_ROOT", "/env-root")

	flags := pflag.NewFlagSet("spack-test", pflag.ContinueOnError)
	flags.Int("http.port", 0, "")
	flags.Bool("http.low_memory", true, "")
	flags.String("assets.backend", "", "")
	flags.String("logger.level", "", "")
	if err := flags.Parse([]string{"--http.port=8088", "--http.low_memory=false", "--assets.backend=local", "--logger.level=debug"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadWithOptions(config.LoadOptions{
		Files:   []string{configPath},
		FlagSet: flags,
	})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.HTTP.Port != 8088 {
		t.Fatalf("expected flag to override http.port to 8088, got %d", cfg.HTTP.Port)
	}
	if cfg.HTTP.LowMemory {
		t.Fatal("expected flag to override http.low_memory to false")
	}
	if cfg.Assets.Path != "/from-file" {
		t.Fatalf("expected config file to set assets.path, got %q", cfg.Assets.Path)
	}
	if cfg.Assets.Root != "/env-root" {
		t.Fatalf("expected env to override assets.root, got %q", cfg.Assets.Root)
	}
	if cfg.Assets.NormalizedBackend() != config.SourceBackendLocal {
		t.Fatalf("expected flag to set assets.backend to %q, got %q", config.SourceBackendLocal, cfg.Assets.NormalizedBackend())
	}
	if cfg.Logger.Level != "debug" {
		t.Fatalf("expected flag to override logger.level, got %q", cfg.Logger.Level)
	}
}

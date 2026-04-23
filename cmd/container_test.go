package cmd_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/arcgolabs/dix"
	obsprom "github.com/arcgolabs/observabilityx/prometheus"
	"github.com/daiyuang/spack/cmd"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/asyncx"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/contentcoding"
	"github.com/daiyuang/spack/internal/event"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/daiyuang/spack/internal/server"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/internal/sourcecatalog"
)

func TestCreateContainerBuildPublishesDixMetrics(t *testing.T) {
	t.Setenv("SPACK_ASSETS_ROOT", t.TempDir())
	t.Setenv("SPACK_LOGGER_CONSOLE_ENABLED", "false")

	app, err := cmd.CreateContainerForTest(
		config.LoadOptions{},
		asyncx.Module,
		event.Module,
		source.Module,
		sourcecatalog.Module,
		artifact.Module,
		contentcoding.Module,
		assetcache.Module,
		pipeline.Module,
		resolver.Module,
		server.Module,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := app.RunStopTimeout(); got != dix.DefaultRunStopTimeout {
		t.Fatalf("expected run stop timeout %s, got %s", dix.DefaultRunStopTimeout, got)
	}

	rt, err := app.Build()
	if err != nil {
		t.Fatal(err)
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Fatal("expected build info to be available")
	}
	if got := rt.Meta().Version; got != info.Main.Version {
		t.Fatalf("expected runtime version %q, got %q", info.Main.Version, got)
	}

	adapter, err := dix.ResolveAs[*obsprom.Adapter](rt.Container())
	if err != nil {
		t.Fatal(err)
	}

	body := ""
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/prometheus", http.NoBody)
		response := httptest.NewRecorder()
		adapter.Handler().ServeHTTP(response, request)

		body = response.Body.String()
		if strings.Contains(body, "spack_dix_build_total") && strings.Contains(body, `app="spack"`) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !strings.Contains(body, "spack_dix_build_total") {
		t.Fatalf("expected dix build metric to be exported, got body:\n%s", body)
	}
	if !strings.Contains(body, `app="spack"`) {
		t.Fatalf("expected dix metrics to include app label, got body:\n%s", body)
	}
}

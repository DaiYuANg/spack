package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/dix"
	obsprom "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
	"github.com/daiyuang/spack/internal/artifact"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/contentcoding"
	"github.com/daiyuang/spack/internal/event"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/daiyuang/spack/internal/server"
	"github.com/daiyuang/spack/internal/source"
	"github.com/daiyuang/spack/internal/sourcecatalog"
	"github.com/daiyuang/spack/internal/workerpool"
)

func TestCreateContainerBuildPublishesDixMetrics(t *testing.T) {
	t.Setenv("SPACK_ASSETS_ROOT", t.TempDir())
	t.Setenv("SPACK_LOGGER_CONSOLE_ENABLED", "false")

	app, err := createContainer(
		config.LoadOptions{},
		workerpool.Module,
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

	rt, err := app.Build()
	if err != nil {
		t.Fatal(err)
	}

	adapter, err := dix.ResolveAs[*obsprom.Adapter](rt.Container())
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/prometheus", http.NoBody)
	response := httptest.NewRecorder()
	adapter.Handler().ServeHTTP(response, request)

	body := response.Body.String()
	if !strings.Contains(body, "spack_dix_build_total") {
		t.Fatalf("expected dix build metric to be exported, got body:\n%s", body)
	}
	if !strings.Contains(body, `app="spack"`) {
		t.Fatalf("expected dix metrics to include app label, got body:\n%s", body)
	}
}

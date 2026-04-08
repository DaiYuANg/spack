package server_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
)

type healthResponse struct {
	Kind    string             `json:"kind"`
	Healthy bool               `json:"healthy"`
	Checks  map[string]*string `json:"checks"`
}

func TestHealthRoutesReturnHealthyReports(t *testing.T) {
	root := t.TempDir()

	cfg := config.DefaultConfigForTest()
	cfg.Debug.Enable = false
	cfg.Assets.Root = root

	cat := catalog.NewInMemoryCatalog()
	app := newHTTPTestApp(
		t,
		&cfg,
		slog.New(slog.DiscardHandler),
		cat,
		assetcache.NewCacheForTest(cfg.HTTP.MemoryCache, slog.New(slog.DiscardHandler)),
		resolver.NewResolverForTest(&cfg.Assets, cat, slog.New(slog.DiscardHandler)),
	)

	assertHealthResponse(t, app, "/healthz", http.StatusOK, "general", "catalog", "")
	assertHealthResponse(t, app, "/livez", http.StatusOK, "liveness", "server", "")
	assertHealthResponse(t, app, "/readyz", http.StatusOK, "readiness", "assets_root", "")
}

func TestReadinessRouteReturnsUnavailableWhenAssetsRootIsMissing(t *testing.T) {
	cfg := config.DefaultConfigForTest()
	cfg.Debug.Enable = false
	cfg.Assets.Root = t.TempDir() + "/missing"

	cat := catalog.NewInMemoryCatalog()
	app := newHTTPTestApp(
		t,
		&cfg,
		slog.New(slog.DiscardHandler),
		cat,
		assetcache.NewCacheForTest(cfg.HTTP.MemoryCache, slog.New(slog.DiscardHandler)),
		resolver.NewResolverForTest(&cfg.Assets, cat, slog.New(slog.DiscardHandler)),
	)

	assertHealthResponse(t, app, "/readyz", http.StatusServiceUnavailable, "readiness", "assets_root", "stat assets root")
}

func assertHealthResponse(
	t *testing.T,
	app *fiber.App,
	path string,
	status int,
	kind string,
	checkName string,
	checkContains string,
) {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer closeHTTPBody(t, response)

	if response.StatusCode != status {
		t.Fatalf("expected %s to return %d, got %d", path, status, response.StatusCode)
	}

	var payload healthResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Kind != kind {
		t.Fatalf("expected %s kind %q, got %q", path, kind, payload.Kind)
	}

	checkMessage, ok := payload.Checks[checkName]
	if !ok {
		t.Fatalf("expected %s check %q to be present", path, checkName)
	}
	if checkContains == "" {
		if checkMessage != nil {
			t.Fatalf("expected %s check %q to be healthy, got %q", path, checkName, *checkMessage)
		}
		return
	}
	if checkMessage == nil || !strings.Contains(*checkMessage, checkContains) {
		t.Fatalf("expected %s check %q to contain %q, got %v", path, checkName, checkContains, checkMessage)
	}
}

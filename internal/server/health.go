package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/daiyuang/spack/internal/catalog"
	"github.com/daiyuang/spack/internal/config"
	"github.com/gofiber/fiber/v3"
)

const (
	healthEndpoint    = "/healthz"
	livenessEndpoint  = "/livez"
	readinessEndpoint = "/readyz"
)

type healthCheckDefinition struct {
	Name  string
	Kind  dix.HealthKind
	Check dix.HealthCheckFunc
}

func newHealthCheckDefinitions(cfg *config.Config, cat catalog.Catalog) collectionx.List[healthCheckDefinition] {
	return collectionx.NewList(
		newHealthCheckDefinition(dix.HealthKindGeneral, "catalog", func(context.Context) error {
			return checkCatalog(cat)
		}),
		newHealthCheckDefinition(dix.HealthKindLiveness, "server", func(context.Context) error {
			return nil
		}),
		newHealthCheckDefinition(dix.HealthKindReadiness, "assets_root", func(context.Context) error {
			return checkAssetsRoot(cfg.Assets.Root)
		}),
	)
}

func newHealthCheckDefinition(kind dix.HealthKind, name string, check dix.HealthCheckFunc) healthCheckDefinition {
	return healthCheckDefinition{
		Name:  strings.TrimSpace(name),
		Kind:  kind,
		Check: check,
	}
}

func registerHealthCheckSetup(container *dix.Container, _ dix.Lifecycle) error {
	checks, err := dix.ResolveAs[collectionx.List[healthCheckDefinition]](container)
	if err != nil {
		return err
	}

	checks.Range(func(_ int, check healthCheckDefinition) bool {
		if check.Name == "" || check.Check == nil {
			return true
		}

		switch check.Kind {
		case dix.HealthKindGeneral:
			container.RegisterHealthCheck(check.Name, check.Check)
		case dix.HealthKindLiveness:
			container.RegisterLivenessCheck(check.Name, check.Check)
		case dix.HealthKindReadiness:
			container.RegisterReadinessCheck(check.Name, check.Check)
		default:
			container.RegisterHealthCheck(check.Name, check.Check)
		}
		return true
	})
	return nil
}

func registerHealthRoutes(
	app *fiber.App,
	cat catalog.Catalog,
	checks collectionx.List[healthCheckDefinition],
) {
	app.Get(healthEndpoint, healthHandler(dix.HealthKindGeneral, checks))
	app.Get(livenessEndpoint, healthHandler(dix.HealthKindLiveness, checks))
	app.Get(readinessEndpoint, healthHandler(dix.HealthKindReadiness, checks))
	app.Get("/catalog", func(c fiber.Ctx) error {
		return c.JSON(cat.Snapshot())
	})
}

func healthHandler(kind dix.HealthKind, checks collectionx.List[healthCheckDefinition]) fiber.Handler {
	return func(c fiber.Ctx) error {
		report := runHealthChecks(c.Context(), kind, checks)
		status := fiber.StatusOK
		if !report.Healthy() {
			status = fiber.StatusServiceUnavailable
		}
		c.Status(status)
		return c.JSON(report)
	}
}

func runHealthChecks(ctx context.Context, kind dix.HealthKind, checks collectionx.List[healthCheckDefinition]) dix.HealthReport {
	matched := checks.Where(func(_ int, check healthCheckDefinition) bool {
		return check.Kind == kind && check.Name != "" && check.Check != nil
	})
	report := dix.HealthReport{
		Kind:   kind,
		Checks: collectionx.NewMapWithCapacity[string, error](matched.Len()),
	}
	matched.Range(func(_ int, check healthCheckDefinition) bool {
		report.Checks.Set(check.Name, check.Check(ctx))
		return true
	})
	return report
}

func checkCatalog(cat catalog.Catalog) error {
	if cat == nil {
		return errors.New("catalog is not configured")
	}
	if cat.Snapshot() == nil {
		return errors.New("catalog snapshot is unavailable")
	}
	return nil
}

func checkAssetsRoot(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return errors.New("assets root is not configured")
	}

	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat assets root %q: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("assets root %q is not a directory", root)
	}
	return nil
}

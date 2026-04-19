package server

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/daiyuang/spack/internal/assetcache"
	"github.com/daiyuang/spack/internal/cachepolicy"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/media"
	"github.com/daiyuang/spack/internal/pipeline"
	"github.com/daiyuang/spack/internal/requestpath"
	"github.com/daiyuang/spack/internal/resolver"
	"github.com/gofiber/fiber/v3"
)

const maxVariantFallbackAttempts = 3

type assetDeliveryRuntime struct {
	mountPath      string
	responsePolicy cachepolicy.ResponsePolicy
	logger         *slog.Logger
	assetResolver  *resolver.Resolver
	pipelineSvc    *pipeline.Service
	bodyCache      *assetcache.Cache
	bus            eventx.BusRuntime
	trackDelivery  bool
}

func registerAssetRoute(
	app *fiber.App,
	cfg *config.Config,
	logger *slog.Logger,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	bodyCache *assetcache.Cache,
	bus eventx.BusRuntime,
	trackDelivery bool,
) {
	runtime := newAssetDeliveryRuntime(cfg, logger, assetResolver, pipelineSvc, bodyCache, bus, trackDelivery)
	app.Use(routePattern(cfg.Assets.Path), runtime.handle)
}

func newAssetDeliveryRuntime(
	cfg *config.Config,
	logger *slog.Logger,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	bodyCache *assetcache.Cache,
	bus eventx.BusRuntime,
	trackDelivery bool,
) *assetDeliveryRuntime {
	return &assetDeliveryRuntime{
		mountPath:      cfg.Assets.Path,
		responsePolicy: cachepolicy.NewResponsePolicy(&cfg.Compression),
		logger:         logger,
		assetResolver:  assetResolver,
		pipelineSvc:    pipelineSvc,
		bodyCache:      bodyCache,
		bus:            bus,
		trackDelivery:  trackDelivery,
	}
}

func (r *assetDeliveryRuntime) handle(c fiber.Ctx) error {
	requestedFormat := media.NormalizeImageFormat(c.Query("format"))
	request := buildResolverRequest(c, r.mountPath, requestedFormat)
	result, err := r.assetResolver.Resolve(request)
	if err != nil {
		return fiber.ErrNotFound
	}

	r.enqueuePipelineResult(result)
	delivery, resolvedResult, deliveryErr := r.sendResolvedAssetWithVariantFallback(c, request, result, requestedFormat)
	if deliveryErr != nil {
		return deliveryErr
	}
	if delivery != "" {
		if r.trackDelivery {
			setAssetDelivery(c, delivery)
		}
		publishVariantServed(c.Context(), resolvedResult, r.bus, r.logger)
	}
	return nil
}

func (r *assetDeliveryRuntime) sendResolvedAssetWithVariantFallback(
	c fiber.Ctx,
	request resolver.Request,
	result *resolver.Result,
	requestedFormat string,
) (string, *resolver.Result, error) {
	delivery, err := r.sendResolvedAsset(c, request, result, requestedFormat)
	if err == nil {
		return delivery, result, nil
	}

	missingErr := asMissingResolvedVariantError(err)
	if missingErr == nil {
		return "", result, err
	}

	return r.retryResolvedAssetDelivery(c, request, result, requestedFormat, missingErr)
}

func (r *assetDeliveryRuntime) retryResolvedAssetDelivery(
	c fiber.Ctx,
	request resolver.Request,
	result *resolver.Result,
	requestedFormat string,
	missingErr *missingResolvedVariantError,
) (string, *resolver.Result, error) {
	for range maxVariantFallbackAttempts {
		nextResult, resolveErr := r.resolveAfterVariantArtifactMiss(request, result, missingErr)
		if resolveErr != nil {
			return "", result, resolveErr
		}
		r.enqueuePipelineResult(nextResult)

		delivery, err := r.sendResolvedAsset(c, request, nextResult, requestedFormat)
		if err == nil {
			return delivery, nextResult, nil
		}

		missingErr = asMissingResolvedVariantError(err)
		if missingErr == nil {
			return "", nextResult, err
		}
		result = nextResult
	}

	return "", result, fiber.ErrInternalServerError
}

func (r *assetDeliveryRuntime) resolveAfterVariantArtifactMiss(
	request resolver.Request,
	result *resolver.Result,
	missingErr *missingResolvedVariantError,
) (*resolver.Result, error) {
	if r.bodyCache != nil {
		r.bodyCache.Delete(missingErr.artifactPath)
	}

	nextResult, err := r.assetResolver.ResolveAfterVariantArtifactMiss(request, result.Variant)
	if err != nil {
		return nil, fiber.ErrNotFound
	}
	return nextResult, nil
}

func asMissingResolvedVariantError(err error) *missingResolvedVariantError {
	if missingErr, ok := errors.AsType[*missingResolvedVariantError](err); ok {
		return missingErr
	}
	return nil
}

func buildResolverRequest(c fiber.Ctx, mountPath, requestedFormat string) resolver.Request {
	cleanedPath := requestpath.CleanMounted(c.Path(), mountPath)
	return resolver.Request{
		Path:           cleanedPath.Value,
		Accept:         c.Get(fiber.HeaderAccept),
		AcceptEncoding: c.Get(fiber.HeaderAcceptEncoding),
		Width:          parsePositiveInt(c.Query("w")),
		Format:         requestedFormat,
		RangeRequested: strings.TrimSpace(c.Get(fiber.HeaderRange)) != "",
	}
}

func (r *assetDeliveryRuntime) enqueuePipelineResult(result *resolver.Result) {
	if r.pipelineSvc == nil || result == nil || result.Asset == nil || !hasPreferredPipelineRequests(result) {
		return
	}

	r.pipelineSvc.Enqueue(pipeline.Request{
		AssetPath:          result.Asset.Path,
		PreferredEncodings: result.PreferredEncodings,
		PreferredWidths:    result.PreferredWidths,
		PreferredFormats:   result.PreferredFormats,
	})
}

func hasPreferredPipelineRequests(result *resolver.Result) bool {
	if result == nil {
		return false
	}
	return preferredListLen(result.PreferredEncodings) > 0 ||
		preferredListLen(result.PreferredWidths) > 0 ||
		preferredListLen(result.PreferredFormats) > 0
}

func preferredListLen[T any](values collectionx.List[T]) int {
	if values == nil {
		return 0
	}
	return values.Len()
}

func parsePositiveInt(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

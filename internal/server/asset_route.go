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
	responsePolicy := cachepolicy.NewResponsePolicy(&cfg.Compression)
	app.Use(routePattern(cfg.Assets.Path), func(c fiber.Ctx) error {
		requestedFormat := media.NormalizeImageFormat(c.Query("format"))
		request := buildResolverRequest(c, cfg.Assets.Path, requestedFormat)
		result, err := assetResolver.Resolve(request)
		if err != nil {
			return fiber.ErrNotFound
		}

		enqueuePipelineResult(result, pipelineSvc)
		delivery, resolvedResult, deliveryErr := sendResolvedAssetWithVariantFallback(
			c,
			responsePolicy,
			request,
			result,
			requestedFormat,
			logger,
			bodyCache,
			assetResolver,
			pipelineSvc,
		)
		if deliveryErr != nil {
			return deliveryErr
		}
		if delivery != "" {
			if trackDelivery {
				setAssetDelivery(c, delivery)
			}
			publishVariantServed(c.Context(), resolvedResult, bus, logger)
		}
		return nil
	})
}

func sendResolvedAssetWithVariantFallback(
	c fiber.Ctx,
	responsePolicy cachepolicy.ResponsePolicy,
	request resolver.Request,
	result *resolver.Result,
	requestedFormat string,
	logger *slog.Logger,
	bodyCache *assetcache.Cache,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
) (string, *resolver.Result, error) {
	delivery, err := sendResolvedAsset(c, responsePolicy, request, result, requestedFormat, logger, bodyCache)
	if err == nil {
		return delivery, result, nil
	}

	missingErr := asMissingResolvedVariantError(err)
	if missingErr == nil {
		return "", result, err
	}

	return retryResolvedAssetDelivery(
		c,
		responsePolicy,
		request,
		result,
		requestedFormat,
		logger,
		bodyCache,
		assetResolver,
		pipelineSvc,
		missingErr,
	)
}

func retryResolvedAssetDelivery(
	c fiber.Ctx,
	responsePolicy cachepolicy.ResponsePolicy,
	request resolver.Request,
	result *resolver.Result,
	requestedFormat string,
	logger *slog.Logger,
	bodyCache *assetcache.Cache,
	assetResolver *resolver.Resolver,
	pipelineSvc *pipeline.Service,
	missingErr *missingResolvedVariantError,
) (string, *resolver.Result, error) {
	for range maxVariantFallbackAttempts {
		nextResult, resolveErr := resolveAfterVariantArtifactMiss(bodyCache, assetResolver, request, result, missingErr)
		if resolveErr != nil {
			return "", result, resolveErr
		}
		enqueuePipelineResult(nextResult, pipelineSvc)

		delivery, err := sendResolvedAsset(c, responsePolicy, request, nextResult, requestedFormat, logger, bodyCache)
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

func resolveAfterVariantArtifactMiss(
	bodyCache *assetcache.Cache,
	assetResolver *resolver.Resolver,
	request resolver.Request,
	result *resolver.Result,
	missingErr *missingResolvedVariantError,
) (*resolver.Result, error) {
	if bodyCache != nil {
		bodyCache.Delete(missingErr.artifactPath)
	}

	nextResult, err := assetResolver.ResolveAfterVariantArtifactMiss(request, result.Variant)
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

func enqueuePipelineResult(result *resolver.Result, pipelineSvc *pipeline.Service) {
	if pipelineSvc == nil || result == nil || result.Asset == nil || !hasPreferredPipelineRequests(result) {
		return
	}

	pipelineSvc.Enqueue(pipeline.Request{
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

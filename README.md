# SPACK

SPACK is a container-first static asset runtime for SPA and frontend build outputs.

It is intentionally narrower than Nginx:

- one process serves one asset mount
- configuration is environment-variable driven
- optimized for container images and runtime base image usage
- built-in asset optimization pipeline instead of generic web server features

Current scope:

- SPA/static asset serving
- `index.html` fallback for client-side routing
- runtime asset catalog
- `gzip` and `brotli` variant generation
- on-demand image width/format variants via query or `Accept` negotiation
- lazy or warmup compression modes
- debug and metrics endpoints for container diagnostics

Out of scope:

- reverse proxy
- dynamic rewrite DSL
- TLS termination
- scripting plugins
- Nginx-style complex `location` semantics

## Runtime Model

The current architecture is:

1. `source`
   Reads files from the mounted asset directory.
2. `catalog`
   Stores assets and generated variants as runtime metadata.
3. `pipeline`
   Generates optimized variants such as compressed assets.
4. `resolver`
   Maps an HTTP request to the best asset or variant.
5. `server`
   Handles HTTP, headers, fallback, and observability.

## Quick Start

```dockerfile
FROM daiyuang/spack:latest

COPY ./dist /app

ENV SPACK_ASSETS_ROOT=/app
ENV SPACK_ASSETS_PATH=/
ENV SPACK_ASSETS_FALLBACK_TARGET=index.html
ENV SPACK_LOGGER_LEVEL=info
ENV SPACK_COMPRESSION_ENABLE=true
ENV SPACK_COMPRESSION_MODE=lazy
ENV SPACK_IMAGE_ENABLE=true
```

Then run:

```powershell
go run .
```

Important endpoints:

- `/healthz`
- `/catalog`
- `/prometheus` when debug runtime is enabled
- `/debug/statsviz` on the debug runtime port

## Configuration

Required:

- `SPACK_ASSETS_ROOT`

Common:

- `SPACK_ASSETS_PATH=/`
- `SPACK_ASSETS_ENTRY=index.html`
- `SPACK_ASSETS_FALLBACK_ON=not_found`
- `SPACK_ASSETS_FALLBACK_TARGET=index.html`
- `SPACK_HTTP_PORT=80`
- `SPACK_LOGGER_LEVEL=info`

Compression:

- `SPACK_COMPRESSION_ENABLE=true`
- `SPACK_COMPRESSION_MODE=lazy|warmup|off`
- `SPACK_COMPRESSION_CACHE_DIR=<path>`
- `SPACK_COMPRESSION_MIN_SIZE=1024`
- `SPACK_COMPRESSION_WORKERS=2`
- `SPACK_COMPRESSION_QUEUE_SIZE=128`
- `SPACK_COMPRESSION_CLEANUP_EVERY=5m`
- `SPACK_COMPRESSION_MAX_AGE=168h`
- `SPACK_COMPRESSION_IMAGE_MAX_AGE=336h`
- `SPACK_COMPRESSION_ENCODING_MAX_AGE=168h`
- `SPACK_COMPRESSION_MAX_CACHE_BYTES=1073741824`

Images:

- `SPACK_IMAGE_ENABLE=true`
- `SPACK_IMAGE_WIDTHS=640,1280,1920`
- `SPACK_IMAGE_JPEG_QUALITY=78`
- request width variants with `?w=<width>`
- request format variants with `?format=jpeg|png`
- format can also be negotiated from `Accept: image/jpeg,image/png`
- combine both as `?w=640&format=jpeg`

Debug runtime:

- `SPACK_DEBUG_ENABLE=true`
- `SPACK_DEBUG_LIVE_PORT=8080`
- `SPACK_METRICS_PREFIX=/prometheus`
- includes HTTP and pipeline runtime metrics (queue length, enqueue drop/dedupe, cleanup activity)

## Development

Run tests:

```powershell
go test ./...
```

Use the SPA fixture:

```powershell
pnpm -C test build
$env:SPACK_ASSETS_ROOT = (Resolve-Path .\test\build\dist).Path
go run .
```

## Next

The pipeline abstraction is in place for future stages such as:

- responsive image variants
- additional cache policy strategies (beyond TTL and max-size eviction)

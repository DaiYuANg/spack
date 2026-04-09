FROM --platform=$BUILDPLATFORM golang:bookworm AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0

USER root

WORKDIR /app

ARG UPX_VERSION=5.1.1
ARG BUILDARCH
ARG TARGETOS
ARG TARGETARCH

RUN apt-get update \
    && apt-get upgrade -y -fix-missing \
    && apt-get install -y -fix-missing --no-install-recommends curl xz-utils ca-certificates \
    && ARCH="${BUILDARCH}" && \
    case "$ARCH" in \
        amd64)   UPX_ARCH=amd64 ;; \
        arm64)   UPX_ARCH=arm64 ;; \
        *)       echo "Unsupported arch: $ARCH" && exit 1 ;; \
    esac && \
    curl -sSL "https://github.com/upx/upx/releases/download/v${UPX_VERSION}/upx-${UPX_VERSION}-${UPX_ARCH}_linux.tar.xz" \
    | tar -xJ && mv upx-${UPX_VERSION}-${UPX_ARCH}_linux/upx /usr/local/bin/ \
    && rm -rf upx-${UPX_VERSION}-${UPX_ARCH}_linux* \
    && curl -sSfL https://taskfile.dev/install.sh | sh -s -- -d \
    && rm -rf /var/lib/apt/lists/*

COPY . .

RUN GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" ./bin/task build \
    || GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -trimpath -ldflags="-s -w" -o dist/spack .

RUN upx --best --lzma dist/spack

FROM alpine:latest AS alpine

RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates curl dumb-init \
    && adduser -D -g '' appuser

WORKDIR /opt
COPY --from=builder /app/dist/spack /opt/spack

RUN chmod +x /opt/spack

USER appuser

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

EXPOSE 80
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -fsS http://127.0.0.1/livez || exit 1

CMD ["sh", "-c", "/opt/spack"]

FROM debian:stable-slim AS debian

WORKDIR /opt

COPY --from=builder /app/dist/spack /opt/spack

RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends ca-certificates curl dumb-init \
    && rm -rf /var/lib/apt/lists/*

RUN chmod +x /opt/spack

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

EXPOSE 80
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -fsS http://127.0.0.1/livez || exit 1

CMD ["sh", "-c", "/opt/spack"]

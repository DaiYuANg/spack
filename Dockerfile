# 构建阶段（使用 Go 官方 slim 版本）
FROM golang:bookworm AS builder

# 启用 Go module，关闭 CGO
ENV GO111MODULE=on \
    CGO_ENABLED=1

USER root

WORKDIR /app

ARG UPX_VERSION=5.0.1

RUN apt update && apt install -y curl xz-utils ca-certificates dumb-init libwebp-dev   \
    && ARCH=$(dpkg --print-architecture) && \
    case "$ARCH" in \
        amd64)   UPX_ARCH=amd64 ;; \
        arm64)   UPX_ARCH=arm64 ;; \
        *)       echo "Unsupported arch: $ARCH" && exit 1 ;; \
    esac && \
    curl -sSL "https://github.com/upx/upx/releases/download/v${UPX_VERSION}/upx-${UPX_VERSION}-${UPX_ARCH}_linux.tar.xz" \
    | tar -xJ && mv upx-${UPX_VERSION}-${UPX_ARCH}_linux/upx /usr/local/bin/ \
    && rm -rf upx-${UPX_VERSION}-${UPX_ARCH}_linux* \
    && curl -sSfL https://taskfile.dev/install.sh | sh -s -- -d

COPY . .

# 使用 task 或者 go build 构建静态二进制
RUN ./bin/task build || go build -trimpath -ldflags="-s -w" -o dist/spack .

RUN upx --best --lzma dist/spack

FROM frolvlad/alpine-glibc AS alpine
RUN adduser -D -g '' appuser

WORKDIR /app
COPY --from=builder /app/dist/spack /app/spack

USER root

RUN apk add --no-registry dumb-init libwebp

USER appuser

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["sh", "-c", "/app/spack"]

FROM debian:stable-slim AS debian

WORKDIR /app

COPY --from=builder /usr/bin/dumb-init /usr/bin/dumb-init

COPY --from=builder /app/dist/spack /app/spack

RUN apt-get update && apt-get install -y ca-certificates libwebp7 && rm -rf /var/lib/apt/lists/*

RUN chmod +x /app/spack

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["sh", "-c", "/app/spack"]
# 构建阶段（使用 Go 官方 slim 版本）
FROM golang:1.24-alpine AS builder

# 启用 Go module，关闭 CGO
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0

WORKDIR /app

# 安装 Taskfile（可选，如果你不需要就移除）
RUN apk add --no-cache curl && \
    sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

COPY . .

# 使用 task 或者 go build 构建静态二进制
RUN ./bin/task build || go build -trimpath -ldflags="-s -w" -o dist/sproxy .

FROM gcr.io/distroless/static:nonroot  AS distroless

WORKDIR /app

COPY --from=builder /app/dist/sproxy /app/sproxy

USER nonroot:nonroot

ENTRYPOINT ["/app/sproxy"]

FROM scratch AS scratch

COPY --from=builder /app/dist/sproxy /sproxy

ENTRYPOINT ["/sproxy"]

FROM alpine:latest AS alpine
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/dist/sproxy /app/sproxy
USER appuser
ENTRYPOINT ["/app/sproxy"]

FROM debian:stable-slim AS debian
WORKDIR /app
COPY --from=builder /app/dist/sproxy /app/sproxy
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
RUN chmod +x /app/sproxy
ENTRYPOINT ["/app/sproxy"]
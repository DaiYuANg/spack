# 构建阶段（使用 Go 官方 slim 版本）
FROM golang:bookworm AS builder

# 启用 Go module，关闭 CGO
ENV GO111MODULE=on \
    CGO_ENABLED=0

WORKDIR /app

# 安装 Taskfile
RUN apk add --no-cache curl upx && \
    sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

COPY . .

# 使用 task 或者 go build 构建静态二进制
RUN ./bin/task build || go build -trimpath -ldflags="-s -w" -o dist/sproxy .

RUN upx --best --lzma dist/sproxy

FROM gcr.io/distroless/base-debian12  AS distroless

WORKDIR /app

COPY --from=builder /app/dist/sproxy /app/sproxy
COPY --from=builder /usr/bin/dumb-init /usr/bin/dumb-init

USER nonroot:nonroot

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["sh", "-c", "/app/sproxy"]

FROM alpine:latest AS alpine
RUN adduser -D -g '' appuser

WORKDIR /app
COPY --from=builder /app/dist/sproxy /app/sproxy

USER appuser

RUN apk add --no-cache dumb-init

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["sh", "-c", "/app/sproxy"]

FROM debian:stable-slim AS debian

WORKDIR /app

COPY --from=builder /usr/bin/dumb-init /usr/bin/dumb-init

COPY --from=builder /app/dist/sproxy /app/sproxy

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

RUN chmod +x /app/sproxy

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["sh", "-c", "/app/sproxy"]
# 构建阶段
FROM golang:1.24-alpine AS builder

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache git curl build-base libwebp-dev

WORKDIR /app

RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

COPY . .

RUN ./bin/task build

# 运行阶段
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /app/dist/sproxy /app/sproxy

USER nonroot:nonroot

ENTRYPOINT ["/app/sproxy"]

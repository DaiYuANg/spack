FROM frolvlad/alpine-glibc AS alpine
RUN adduser -D -g '' appuser

WORKDIR /opt
COPY --from=builder /app/spack /opt/spack

USER root

RUN apk add --no-cache dumb-init curl

RUN chmod +x /opt/spack

USER appuser

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

EXPOSE 80
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD curl -fsS http://127.0.0.1/livez || exit 1

CMD ["sh", "-c", "/opt/spack"]

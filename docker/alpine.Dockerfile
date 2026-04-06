FROM alpine:3.22

RUN adduser -D -g '' appuser \
    && apk add --no-cache ca-certificates dumb-init

WORKDIR /opt

COPY spack /opt/spack

RUN chmod +x /opt/spack

USER appuser

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

EXPOSE 80
EXPOSE 8080

CMD ["/opt/spack"]

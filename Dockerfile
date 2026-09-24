FROM alpine:3.23@sha256:85fe1e81d6758c208f3e1eed4338a1997e19d4be002d4dd32d3100c9a8c010a0

ARG SOURCE
ARG REVISION

LABEL org.opencontainers.image.source="$SOURCE" \
      org.opencontainers.image.revision="$REVISION"

RUN mkdir /work && chown 65532:65532 /work

COPY --chown=65532:65532 --chmod=0755 dist/kapellmeister-linux-amd64 /usr/local/bin/kapellmeister

USER 65532:65532
WORKDIR /work

ENTRYPOINT ["/usr/local/bin/kapellmeister"]

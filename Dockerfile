FROM alpine:3.23

ARG SOURCE
ARG REVISION

LABEL org.opencontainers.image.source=$SOURCE \
      org.opencontainers.image.revision=$REVISION

RUN apk add --no-cache ca-certificates

COPY --chmod=0755 dist/kapellmeister /usr/local/bin/kapellmeister

USER 65532:65532
WORKDIR /work

ENTRYPOINT ["/usr/local/bin/kapellmeister"]

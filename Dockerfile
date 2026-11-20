FROM docker.io/library/golang:1.27.4-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder
ENV CGO_ENABLED=0
RUN apk add --no-cache git tini-static
WORKDIR /build
COPY . .
RUN go run github.com/valyala/quicktemplate/qtc@latest -dir=request/templates \
&& go build -o crtsh-web -ldflags "-X github.com/crtsh/crtsh-web/config.BuildTimestamp=`date --utc +%Y-%m-%dT%H:%M:%SZ`" /build/.

FROM gcr.io/distroless/static:nonroot@sha256:1c2c046bc09ed40fad370b599a0b1ae7987f55b01e247cf27a7c27cd97e5bbc7
USER nonroot:nonroot
COPY --from=builder --chown=nonroot:nonroot /build/crtsh-web /app/crtsh-web
COPY --from=builder --chown=nonroot:nonroot /sbin/tini-static /sbin/tini
VOLUME ["/config"]
ENTRYPOINT [ "/sbin/tini", "--", "/app/crtsh-web" ]

LABEL \
    org.opencontainers.image.base.name="gcr.io/distroless/static:nonroot" \
    org.opencontainers.image.title="crtsh-web" \
    org.opencontainers.image.source="https://github.com/crtsh/crtsh-web"

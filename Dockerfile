FROM docker.io/library/golang:1.26.5-alpine AS builder
ENV CGO_ENABLED=0 \
    GOARCH=amd64
RUN apk add --no-cache git tini-static
WORKDIR /build
COPY . .
RUN go run github.com/valyala/quicktemplate/qtc@latest -dir=request/templates \
&& go build -o crtsh-web -ldflags "-X github.com/crtsh/crtsh-web/config.BuildTimestamp=`date --utc +%Y-%m-%dT%H:%M:%SZ`" /build/.

FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot
COPY --from=builder --chown=nonroot:nonroot /build/crtsh-web /app/crtsh-web
COPY --from=builder --chown=nonroot:nonroot /sbin/tini-static /sbin/tini
VOLUME ["/config"]
ENTRYPOINT [ "/sbin/tini", "--", "/app/crtsh-web" ]

LABEL \
    org.opencontainers.image.base.name="gcr.io/distroless/static:nonroot" \
    org.opencontainers.image.title="crtsh-web" \
    org.opencontainers.image.source="https://github.com/crtsh/crtsh-web"

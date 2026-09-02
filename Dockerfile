FROM docker.io/library/golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder
ENV CGO_ENABLED=0
RUN apk add --no-cache git tini-static
WORKDIR /build
COPY . .
RUN go run github.com/valyala/quicktemplate/qtc@latest -dir=request/templates \
&& go build -o crtsh-web -ldflags "-X github.com/crtsh/crtsh-web/config.BuildTimestamp=`date --utc +%Y-%m-%dT%H:%M:%SZ`" /build/.

FROM gcr.io/distroless/static:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6
USER nonroot:nonroot
COPY --from=builder --chown=nonroot:nonroot /build/crtsh-web /app/crtsh-web
COPY --from=builder --chown=nonroot:nonroot /sbin/tini-static /sbin/tini
VOLUME ["/config"]
ENTRYPOINT [ "/sbin/tini", "--", "/app/crtsh-web" ]

LABEL \
    org.opencontainers.image.base.name="gcr.io/distroless/static:nonroot" \
    org.opencontainers.image.title="crtsh-web" \
    org.opencontainers.image.source="https://github.com/crtsh/crtsh-web"

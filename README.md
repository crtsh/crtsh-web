# crtsh-web

New front end application for https://crt.sh/

## Architecture

crtsh-web is a thin HTTP gateway that translates incoming web requests into
PL/pgSQL function calls (`web_apis` / `web_apis_test`) against the `certwatch`
PostgreSQL database and returns the response. It replaces the original Apache
`mod_certwatch` C module.

```
                  ┌──────────────┐
  client ──────▶  │  Web Server  │ :8080
                  │  (fasthttp)  │
                  └──────┬───────┘
                         │ pgx connection pool
                         ▼
                  ┌──────────────┐
                  │  PostgreSQL  │
                  │  certwatch   │
                  └──────────────┘

                  ┌──────────────┐
  ops    ──────▶  │  Monitoring  │ :8081
                  │   Server     │
                  └──────────────┘
```

A reverse proxy (e.g. nginx) is expected in front of the web server to handle
TLS termination and set `X-Forwarded-For` / `X-Real-IP` headers.

## Building

### Prerequisites

- Go 1.26.5+ (see `go.mod`)
- [quicktemplate](https://github.com/valyala/quicktemplate) code generator (`qtc`)

### Build Steps

```sh
# Generate template code (required before go build):
go run github.com/valyala/quicktemplate/qtc@latest -dir=request/templates

# Build the binary:
CGO_ENABLED=0 go build -o crtsh-web \
  -ldflags "-X github.com/crtsh/crtsh-web/config.BuildTimestamp=$(date --utc +%Y-%m-%dT%H:%M:%SZ)"
```

Or use the Makefile:

```sh
make
```

### Docker

```sh
docker build -t crtsh-web .
```

The image uses `gcr.io/distroless/static:nonroot` as the runtime base and runs
as a non-root user. Configuration is read from the `/config` volume.

## Configuration

Configuration is read from (in order of precedence):

1. Environment variables prefixed with `CRTSHWEB_` (dots become underscores,
   e.g. `CRTSHWEB_CERTWATCHDB_HOST`).
2. `config.yaml` found in `/config/`, `./config/`, or the working directory.

### Database (`certwatchdb`)

| Key | Default | Description |
|-----|---------|-------------|
| `certwatchdb.host` | `/var/run/postgresql` | PostgreSQL host or Unix socket directory |
| `certwatchdb.port` | `5432` | PostgreSQL port (ignored for Unix sockets) |
| `certwatchdb.user` | `httpd` | Database user |
| `certwatchdb.password` | _(empty)_ | Database password (omit for peer/socket auth) |

### Connection Pool (`pool`)

| Key | Default | Description |
|-----|---------|-------------|
| `pool.maxConns` | `32` | Maximum connections in the pool |
| `pool.minConns` | `0` | Minimum idle connections to maintain |
| `pool.maxConnLifetime` | `30m` | Maximum lifetime of any connection |
| `pool.maxConnIdleTime` | `5m` | Idle time before a connection is closed |

### Server (`server`)

| Key | Default | Description |
|-----|---------|-------------|
| `server.webserverPort` | `8080` | TCP port for the web server (0 to disable) |
| `server.webserverPath` | _(empty)_ | Unix socket path for the web server |
| `server.monitoringPort` | `8081` | TCP port for the monitoring server (0 to disable) |
| `server.monitoringPath` | _(empty)_ | Unix socket path for the monitoring server |
| `server.socketPermissions` | `0600` | File permissions for Unix sockets |
| `server.readTimeout` | `30s` | HTTP read timeout |
| `server.idleTimeout` | `30s` | HTTP idle timeout |
| `server.disableKeepalive` | `false` | Disable HTTP keep-alive |
| `server.requestTimeout` | `30s` | Maximum time for a web_apis database call |
| `server.livezTimeout` | `500ms` | Liveness probe timeout |
| `server.readyzTimeout` | `500ms` | Readiness probe timeout |
| `server.rememberBusyTimeout` | `5s` | How long to report not-ready after a timeout |
| `server.metricsTimeout` | `8s` | Metrics endpoint timeout |
| `server.enableDebugEndpoints` | `false` | Enable `/debug/build`, `/debug/config`, `/debug/pprof/*` |

### Logging (`logging`)

| Key | Default | Description |
|-----|---------|-------------|
| `logging.isDevelopment` | `false` | Use console-friendly log output |
| `logging.level` | _(auto)_ | Log level override (debug, info, warn, error) |
| `logging.samplingInitial` | `MaxInt` | Zap sampling initial (MaxInt = disabled) |
| `logging.samplingThereafter` | `MaxInt` | Zap sampling thereafter (MaxInt = disabled) |

## Monitoring Endpoints

All monitoring endpoints are served on the monitoring server (default port 8081).

| Path | Description |
|------|-------------|
| `/livez` | Liveness probe. Returns 200 if the most recent request outcome was not an error. |
| `/readyz` | Readiness probe. Returns 200 if the service is not in a busy/timeout state and the database is reachable (ping). |
| `/metrics` | Prometheus metrics (request latency, fasthttp concurrency/connections). |
| `/debug/build` | Build information and dependency versions. **Requires `server.enableDebugEndpoints: true`.** |
| `/debug/config` | Runtime configuration as JSON. **Requires `server.enableDebugEndpoints: true`.** |
| `/debug/pprof/*` | Go pprof profiling endpoints. **Requires `server.enableDebugEndpoints: true`.** |

### Prometheus Metrics

- `crtshweb_request_latency` — Request latency summary (labels: `type=monitoring|web`).
- `crtshweb_fasthttp_concurrency` — Current concurrent connections (labels: `server=monitoring|web`).
- `crtshweb_fasthttp_open` — Open connections.
- `crtshweb_fasthttp_rejected` — Rejected connections.
- `crtshweb_fasthttp_maxconcurrency` — Max concurrent connections.
- `crtshweb_fasthttp_maxconnsperip` — Max connections per IP.

## Deployment

### Recommended production setup

1. Run behind a reverse proxy that terminates TLS and sets `X-Forwarded-For`.
2. Bind the monitoring server to localhost or a private network
   (`server.monitoringPort: 8081` with firewall rules, or use a Unix socket).
3. Keep `server.enableDebugEndpoints: false` (the default) unless actively
   debugging.
4. Configure container orchestration to use `/livez` for liveness and `/readyz`
   for readiness probes.

### Example `config.yaml`

```yaml
certwatchdb:
  host: /var/run/postgresql
  user: httpd

pool:
  maxConns: 64

server:
  webserverPort: 8080
  monitoringPort: 8081
  requestTimeout: 30s

logging:
  level: info
```

## License

[GPLv3](LICENSE)

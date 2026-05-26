package request

// This file is a Go port of the request-processing logic from
// github.com/crtsh/mod_certwatch (an Apache module written in C). It calls the
// `web_apis` / `web_apis_test` PL/pgSQL functions that live in the
// certwatch_db database, and translates the result back into an HTTP
// response, honouring the optional `[BEGIN_HEADERS]...[END_HEADERS]` prefix
// that the PL/pgSQL functions use to customise the response headers.

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/crtsh/crtsh-web/certwatch"
	"github.com/crtsh/crtsh-web/config"
	"github.com/crtsh/crtsh-web/health"
	"github.com/crtsh/crtsh-web/logger"
	"github.com/crtsh/crtsh-web/request/templates"
	"github.com/crtsh/crtsh-web/utils"

	"github.com/valyala/fasthttp"

	"go.uber.org/zap"
)

const (
	// testPathPrefix routes requests through the `web_apis_test` PL/pgSQL
	// function instead of `web_apis`.
	testPathPrefix = "/_ROB_IS_TESTING_/"

	// Markers used by the PL/pgSQL functions to inject custom HTTP
	// response headers ahead of the response body.
	headersBegin = "[BEGIN_HEADERS]\n"
	headersEnd   = "[END_HEADERS]\n"
)

// WebAPIs handles a crt.sh request by invoking the appropriate PL/pgSQL
// `web_apis` function and writing its response back to the client. It
// returns 0 normally, or -1 if the request timed out before the database
// responded (in which case the caller should emit a 503).
func WebAPIs(fhctx *fasthttp.RequestCtx) int {
	ctxWithDeadline, cancel := context.WithDeadline(context.Background(), fhctx.Time().Add(config.Config.Server.RequestTimeout))
	defer cancel()

	doneChan := make(chan int, 1)
	go func() {
		doneChan <- handleWebAPIs(ctxWithDeadline, fhctx)
	}()

	return health.CompleteRequest(ctxWithDeadline, doneChan)
}

func handleWebAPIs(ctx context.Context, fhctx *fasthttp.RequestCtx) int {
	path := utils.B2S(fhctx.Path())
	rawQuery := utils.B2S(fhctx.URI().QueryString())

	// `/test/...` is a legacy redirect to the canonical query-string form.
	if strings.HasPrefix(path, "/test/") {
		location := fmt.Sprintf("https://%s/?%s", utils.B2S(fhctx.Host()), rawQuery)
		fhctx.Response.Header.Set("Location", location)
		fhctx.SetStatusCode(fasthttp.StatusFound)
		logger.SetDetails(fhctx, zap.InfoLevel, "Legacy /test/ redirect", nil, nil)
		return 0
	}

	// Decline anything that looks like a static asset (e.g. images,
	// robots.txt). `.json` paths are valid API endpoints and are NOT
	// declined here.
	if dot := strings.LastIndexByte(path, '.'); dot >= 0 && !strings.EqualFold(path[dot:], ".json") {
		fhctx.NotFound()
		logger.SetDetails(fhctx, zap.InfoLevel, "Not a web_apis endpoint", nil, nil)
		return 0
	}

	// Gather the raw request parameter string.
	var rawParams string
	switch {
	case fhctx.IsGet():
		rawParams = rawQuery
	case fhctx.IsPost():
		rawParams = utils.B2S(fhctx.PostBody())
	default:
		fhctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		logger.SetDetails(fhctx, zap.InfoLevel, "Method not allowed", nil, nil)
		return 0
	}

	// If the client explicitly asked for JSON, prepend `output=json` so
	// the PL/pgSQL function can see it.
	if accept := utils.B2S(fhctx.Request.Header.Peek("Accept")); accept == "application/json" {
		if rawParams == "" {
			rawParams = "output=json"
		} else {
			rawParams = "output=json&" + rawParams
		}
	}

	names, values := parseParams(rawParams)

	// If the URI has a meaningful path component (anything other than
	// `/` or `/?...`), add `output=<path>` so the PL/pgSQL function can
	// dispatch on it. This mirrors the unparsed_uri check in
	// mod_certwatch.
	unparsedURI := path
	if rawQuery != "" {
		unparsedURI = path + "?" + rawQuery
	}
	if len(unparsedURI) > 1 && !strings.Contains(unparsedURI, "/?") {
		output := strings.TrimPrefix(path, "/")
		output = strings.TrimPrefix(output, strings.TrimPrefix(testPathPrefix, "/"))
		if output != "" {
			names = append(names, "output")
			values = append(values, output)
		}
	}

	// The last URI segment is passed to the PL/pgSQL function as $1 and
	// is used by some endpoints (e.g. `/atom`) for additional dispatch.
	lastSegment := ""
	if idx := strings.LastIndexByte(path, '/'); idx >= 0 {
		lastSegment = path[idx+1:]
	}

	// Choose between the production and test PL/pgSQL functions.
	fn := "web_apis"
	if strings.HasPrefix(path, testPathPrefix) {
		fn = "web_apis_test"
	}

	// Annotate the SQL with the client IP and request line so it shows
	// up in PostgreSQL's `pg_stat_activity` output, matching what
	// mod_certwatch did.
	clientIP := utils.B2S(fhctx.Request.Header.Peek("X-Forwarded-For"))
	if clientIP == "" {
		clientIP = fhctx.RemoteAddr().String()
	}
	requestLine := fmt.Sprintf("%s %s %s",
		utils.B2S(fhctx.Method()),
		utils.B2S(fhctx.RequestURI()),
		utils.B2S(fhctx.Request.Header.Protocol()),
	)
	sql := fmt.Sprintf("SELECT %s($1,$2,$3) -- [%s] %s", fn, sanitizeComment(clientIP), sanitizeComment(requestLine))

	// pgx marshals []string as text[] automatically. Pass nil (rather
	// than an empty slice) so the PL/pgSQL function sees NULL when no
	// parameters were supplied, matching mod_certwatch's behaviour.
	var pgNames, pgValues any
	if len(names) > 0 {
		pgNames = names
		pgValues = values
	}

	startTime := time.Now()
	var response []byte
	err := certwatch.Pool.QueryRow(ctx, sql, lastSegment, pgNames, pgValues).Scan(&response)
	queryDuration := time.Since(startTime)

	if err != nil {
		nonError := time.Time{}
		errTime := time.Now()
		health.UpdateLatestTimestamps(&nonError, &errTime, nil)
		logger.SetDetails(fhctx, zap.ErrorLevel, "web_apis query failed", err, []zap.Field{
			zap.Duration("query_duration", queryDuration),
		})
		writeErrorPage(fhctx, queryDuration, err)
		return 0
	}

	nonErrorTime := time.Now()
	health.UpdateLatestTimestamps(&nonErrorTime, nil, nil)

	if len(response) == 0 {
		fhctx.NotFound()
		logger.SetDetails(fhctx, zap.InfoLevel, "Empty web_apis response", nil, []zap.Field{
			zap.Duration("query_duration", queryDuration),
		})
		return 0
	}

	body := response
	customisedContentType := false
	if bytes := utils.B2S(response); strings.HasPrefix(bytes, headersBegin) {
		if end := strings.Index(bytes, headersEnd); end >= 0 {
			headerBlock := bytes[len(headersBegin):end]
			body = response[end+len(headersEnd):]
			for _, line := range strings.Split(headerBlock, "\n") {
				if line == "" {
					continue
				}
				colon := strings.IndexByte(line, ':')
				if colon < 0 {
					continue
				}
				name := strings.TrimSpace(line[:colon])
				value := strings.TrimSpace(line[colon+1:])
				if strings.EqualFold(name, "Content-Type") {
					fhctx.SetContentType(value)
					customisedContentType = true
				} else {
					fhctx.Response.Header.Set(name, value)
				}
			}
		}
	}

	if !customisedContentType {
		fhctx.SetContentType("text/html; charset=UTF-8")
	}
	fhctx.SetStatusCode(fasthttp.StatusOK)
	fhctx.SetBody(body)
	logger.SetDetails(fhctx, zap.InfoLevel, "web_apis", nil, []zap.Field{
		zap.Duration("query_duration", queryDuration),
		zap.String("web_apis_fn", fn),
	})
	return 0
}

// parseParams parses a URL-encoded `name=value&...` string in the same way
// as mod_certwatch: it preserves the order and any duplicate names,
// lowercases the names, and percent-decodes both names and values. Empty
// keys are dropped to match the C implementation.
func parseParams(raw string) ([]string, []string) {
	if raw == "" {
		return nil, nil
	}
	var names, values []string
	for len(raw) > 0 {
		var pair string
		if i := strings.IndexByte(raw, '&'); i >= 0 {
			pair, raw = raw[:i], raw[i+1:]
		} else {
			pair, raw = raw, ""
		}
		if pair == "" {
			continue
		}
		var name, value string
		if eq := strings.IndexByte(pair, '='); eq >= 0 {
			name, value = pair[:eq], pair[eq+1:]
		} else {
			name = pair
		}
		decodedName, err := url.QueryUnescape(name)
		if err != nil {
			decodedName = name
		}
		decodedValue, err := url.QueryUnescape(value)
		if err != nil {
			decodedValue = value
		}
		if decodedName == "" {
			continue
		}
		names = append(names, strings.ToLower(decodedName))
		values = append(values, decodedValue)
	}
	return names, values
}

// sanitizeComment strips any characters that could prematurely terminate
// the trailing SQL line comment that we use to tag queries with their
// originating client/request.
func sanitizeComment(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, s)
}

// writeErrorPage renders the branded crt.sh error page on database failure,
// matching the original mod_certwatch error response.
func writeErrorPage(fhctx *fasthttp.RequestCtx, elapsed time.Duration, err error) {
	fhctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
	fhctx.SetContentType("text/html; charset=UTF-8")
	if fhctx.IsHead() {
		return
	}
	seconds := int64(elapsed.Round(time.Second).Seconds())
	plural := "s"
	if seconds == 1 {
		plural = ""
	}
	templates.WriteErrorPage(fhctx, seconds, plural, err.Error(), time.Now().Year())
}

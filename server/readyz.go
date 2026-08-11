package server

import (
	"context"
	"time"

	"github.com/crtsh/crtsh-web/certwatch"
	"github.com/crtsh/crtsh-web/config"
	"github.com/crtsh/crtsh-web/health"
	"github.com/crtsh/crtsh-web/utils"

	"github.com/valyala/fasthttp"

	"go.uber.org/zap"
)

func readyz(fhctx *fasthttp.RequestCtx) int {
	fhctx.SetUserValue("level", zap.DebugLevel)
	fhctx.SetUserValue("msg", "Readiness check")

	ctx, cancel := context.WithDeadline(context.Background(), fhctx.Time().Add(time.Duration(config.Config.Server.ReadyzTimeout)))
	defer cancel()

	ready, fields := health.IsReady()
	var err error
	if ready {
		err = certwatch.Ping(ctx)
		if err != nil {
			ready = false
		}
	}

	if ctx.Err() != nil {
		return -1
	}

	fhctx.SetUserValue("zap_fields", fields)
	if err != nil {
		fhctx.SetUserValue("error", err)
	}
	statusCode := fasthttp.StatusOK
	if !ready {
		statusCode = fasthttp.StatusServiceUnavailable
	}
	fhctx.SetContentType("text/plain")
	fhctx.SetStatusCode(statusCode)
	if !fhctx.IsHead() {
		if statusCode == fasthttp.StatusOK {
			fhctx.SetBody(utils.S2B("OK"))
		} else {
			fhctx.SetBody(utils.S2B("ERROR"))
		}
	}
	return 0
}

package server

import (
	"github.com/crtsh/crtsh-web/health"
	"github.com/crtsh/crtsh-web/utils"

	"github.com/valyala/fasthttp"

	"go.uber.org/zap"
)

func livez(ctx *fasthttp.RequestCtx) int {
	ctx.SetUserValue("level", zap.DebugLevel)
	ctx.SetUserValue("msg", "Liveness check")

	alive, fields := health.IsAlive()

	ctx.SetUserValue("zap_fields", fields)
	statusCode := fasthttp.StatusOK
	if !alive {
		statusCode = fasthttp.StatusServiceUnavailable
	}
	ctx.SetContentType("text/plain")
	ctx.SetStatusCode(statusCode)
	if !ctx.IsHead() {
		if statusCode == fasthttp.StatusOK {
			ctx.SetBody(utils.S2B("OK"))
		} else {
			ctx.SetBody(utils.S2B("ERROR"))
		}
	}
	return 0
}

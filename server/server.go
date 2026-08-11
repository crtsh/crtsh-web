package server

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/crtsh/crtsh-web/config"
	"github.com/crtsh/crtsh-web/logger"
	"github.com/crtsh/crtsh-web/request"
	"github.com/crtsh/crtsh-web/utils"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/valyala/fasthttp"

	"go.uber.org/zap"
)

var webServer *fasthttp.Server
var webRequestLatency prometheus.Summary

func webHandler(fhctx *fasthttp.RequestCtx) {
	status := 0
	assetPath := strings.ToLower(utils.B2S(fhctx.Path())[1:])
	extension := path.Ext(assetPath)
	if extension != "" && extension != ".json" {
		serveStaticAsset(fhctx, assetPath, extension)
	} else {
		// All other requests are routed to the PL/pgSQL `web_apis` function.
		status = request.WebAPIs(fhctx)
		if status == -1 {
			logger.SetDetails(fhctx, zap.WarnLevel, "web_apis timeout", nil, nil)
		}
	}

	if status == -1 { // Request timed out.
		fhctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		fhctx.SetContentType("text/plain")
		if !fhctx.IsHead() {
			fhctx.SetBody(utils.S2B("ERROR"))
		}
		defer fhctx.TimeoutErrorWithResponse(&fhctx.Response) // The logger needs to run first.
	}

	logger.LogRequest(fhctx)
	webRequestLatency.Observe(float64(time.Since(fhctx.Time())) / float64(time.Second))
}

var monitoringServer *fasthttp.Server
var monitoringRequestLatency prometheus.Summary

func monitoringHandler(fhctx *fasthttp.RequestCtx) {
	status := 0
	switch strings.ToLower(utils.B2S(fhctx.Path())[1:]) {
	case request.ENDPOINTSTRING_FAVICON:
		serveStaticAsset(fhctx, request.ENDPOINTSTRING_FAVICON, ".ico")
	case request.ENDPOINTSTRING_LIVEZ:
		status = livez(fhctx)
	case request.ENDPOINTSTRING_READYZ:
		status = readyz(fhctx)
	case request.ENDPOINTSTRING_METRICS:
		status = metrics(fhctx)
	case request.ENDPOINTSTRING_BUILD:
		if config.Config.Server.EnableDebugEndpoints {
			buildInfo(fhctx)
		} else {
			fhctx.NotFound()
		}
	case request.ENDPOINTSTRING_CONFIG:
		if config.Config.Server.EnableDebugEndpoints {
			configInfo(fhctx)
		} else {
			fhctx.NotFound()
		}
	default:
		if config.Config.Server.EnableDebugEndpoints && profilingHandler(fhctx) {
			// Handled by pprof.
		} else {
			fhctx.NotFound()
		}
	}

	if status == -1 { // Request timed out.
		fhctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		fhctx.SetContentType("text/plain")
		if !fhctx.IsHead() {
			fhctx.SetBody(utils.S2B("ERROR"))
		}
		logger.SetDetails(fhctx, zap.WarnLevel, "Monitoring timeout", nil, nil)
		defer fhctx.TimeoutErrorWithResponse(&fhctx.Response) // The logger needs to run first.
	}

	logger.LogRequest(fhctx)
	monitoringRequestLatency.Observe(float64(time.Since(fhctx.Time())) / float64(time.Second))
}

func Run() {
	webServer = &fasthttp.Server{
		Handler:               webHandler,
		CloseOnShutdown:       true,
		ReadTimeout:           config.Config.Server.ReadTimeout,
		IdleTimeout:           config.Config.Server.IdleTimeout,
		DisableKeepalive:      config.Config.Server.DisableKeepalive,
		MaxRequestBodySize:    config.Config.Server.MaxRequestBodySize,
		NoDefaultServerHeader: true,
	}
	if config.Config.Server.WebserverPort != 0 {
		logger.Logger.Info("Starting WebServer", zap.Int("port", config.Config.Server.WebserverPort))
		go func() {
			if err := webServer.ListenAndServe(fmt.Sprintf(":%d", config.Config.Server.WebserverPort)); err != nil {
				logger.Logger.Fatal("webServer.ListenAndServe failed", zap.Error(err))
			}
		}()
	}
	if config.Config.Server.WebserverPath != "" {
		logger.Logger.Info("Starting WebServer", zap.String("path", config.Config.Server.WebserverPath))
		go func() {
			if err := webServer.ListenAndServeUNIX(config.Config.Server.WebserverPath, config.Config.Server.SocketPermissions); err != nil {
				logger.Logger.Fatal("webServer.ListenAndServeUNIX failed", zap.Error(err))
			}
		}()
	}

	monitoringServer = &fasthttp.Server{
		Handler:               monitoringHandler,
		CloseOnShutdown:       true,
		ReadTimeout:           config.Config.Server.ReadTimeout,
		IdleTimeout:           config.Config.Server.IdleTimeout,
		DisableKeepalive:      config.Config.Server.DisableKeepalive,
		NoDefaultServerHeader: true,
	}
	if config.Config.Server.MonitoringPort != 0 {
		logger.Logger.Info("Starting MonitoringServer", zap.Int("port", config.Config.Server.MonitoringPort))
		go func() {
			if err := monitoringServer.ListenAndServe(fmt.Sprintf(":%d", config.Config.Server.MonitoringPort)); err != nil {
				logger.Logger.Fatal("monitoringServer.ListenAndServe failed", zap.Error(err))
			}
		}()
	}
	if config.Config.Server.MonitoringPath != "" {
		logger.Logger.Info("Starting MonitoringServer", zap.String("path", config.Config.Server.MonitoringPath))
		go func() {
			if err := monitoringServer.ListenAndServeUNIX(config.Config.Server.MonitoringPath, config.Config.Server.SocketPermissions); err != nil {
				logger.Logger.Fatal("monitoringServer.ListenAndServeUNIX failed", zap.Error(err))
			}
		}()
	}
}

func Shutdown() {
	logger.Logger.Info("Stopping WebServer (gracefully)")
	if err := webServer.Shutdown(); err != nil {
		logger.Logger.Error("webServer.Shutdown failed", zap.Error(err))
	}
	logger.Logger.Info("Stopped WebServer")

	logger.Logger.Info("Stopping MonitoringServer (gracefully)")
	if err := monitoringServer.Shutdown(); err != nil {
		logger.Logger.Error("monitoringServer.Shutdown failed", zap.Error(err))
	}
	logger.Logger.Info("Stopped MonitoringServer")
}

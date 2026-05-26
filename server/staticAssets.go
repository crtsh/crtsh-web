package server

import (
	"embed"

	"github.com/crtsh/crtsh-web/logger"

	"github.com/valyala/fasthttp"

	"go.uber.org/zap"
)

//go:embed staticAssets/*
var staticAssets embed.FS

func serveStaticAsset(fhctx *fasthttp.RequestCtx, assetPath, extension string) {
	contentType := "text/plain"
	switch extension {
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".ico":
		contentType = "image/x-icon"
	case ".js":
		contentType = "text/javascript; charset=utf-8"
	case ".png":
		contentType = "image/png"
	case ".svg":
		contentType = "image/svg+xml"
	case ".txt":
		contentType = "text/plain; charset=utf-8"
	default:
		logger.SetDetails(fhctx, zap.WarnLevel, "Unsupported static asset type", nil, []zap.Field{
			zap.String("asset_path", assetPath),
		})
		fhctx.NotFound()
		return
	}

	body, err := staticAssets.ReadFile("staticAssets/" + assetPath)
	if err != nil {
		logger.SetDetails(fhctx, zap.WarnLevel, "Error reading static asset", nil, []zap.Field{
			zap.String("asset_path", assetPath),
			zap.Error(err),
		})
		fhctx.NotFound()
		return
	}

	logger.SetDetails(fhctx, zap.InfoLevel, "Static asset served", nil, []zap.Field{
		zap.String("asset_path", assetPath),
	})
	fhctx.Response.Header.Set(fasthttp.HeaderCacheControl, "max-age=86400")
	fhctx.SetStatusCode(fasthttp.StatusOK)
	fhctx.SetContentType(contentType)
	if !fhctx.IsHead() {
		if fhctx.IsGet() {
			fhctx.SetBody(body)
		} else {
			fhctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			logger.SetDetails(fhctx, zap.InfoLevel, "Method not allowed", nil, nil)
		}
	}
}

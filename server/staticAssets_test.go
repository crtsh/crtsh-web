package server

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func newAssetRequest(method, path string) *fasthttp.RequestCtx {
	var ctx fasthttp.RequestCtx
	ctx.Request.SetRequestURI(path)
	ctx.Request.Header.SetMethod(method)
	return &ctx
}

func TestServeStaticAsset_CSS(t *testing.T) {
	fhctx := newAssetRequest("GET", "/crtsh.css")
	serveStaticAsset(fhctx, "crtsh.css", ".css")
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d", fhctx.Response.StatusCode())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct != "text/css; charset=utf-8" {
		t.Fatalf("content-type: got %q", ct)
	}
	if len(fhctx.Response.Body()) == 0 {
		t.Fatal("expected non-empty body")
	}
	if cc := string(fhctx.Response.Header.Peek("Cache-Control")); cc != "max-age=86400" {
		t.Fatalf("cache-control: got %q", cc)
	}
}

func TestServeStaticAsset_RobotsTxt(t *testing.T) {
	fhctx := newAssetRequest("GET", "/robots.txt")
	serveStaticAsset(fhctx, "robots.txt", ".txt")
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d", fhctx.Response.StatusCode())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct != "text/plain; charset=utf-8" {
		t.Fatalf("content-type: got %q", ct)
	}
}

func TestServeStaticAsset_Favicon(t *testing.T) {
	fhctx := newAssetRequest("GET", "/favicon.ico")
	serveStaticAsset(fhctx, "favicon.ico", ".ico")
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d", fhctx.Response.StatusCode())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct != "image/x-icon" {
		t.Fatalf("content-type: got %q", ct)
	}
}

func TestServeStaticAsset_JS(t *testing.T) {
	fhctx := newAssetRequest("GET", "/asn1js/asn1.js")
	serveStaticAsset(fhctx, "asn1js/asn1.js", ".js")
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d", fhctx.Response.StatusCode())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct != "text/javascript; charset=utf-8" {
		t.Fatalf("content-type: got %q", ct)
	}
}

func TestServeStaticAsset_PNG(t *testing.T) {
	fhctx := newAssetRequest("GET", "/censys.png")
	serveStaticAsset(fhctx, "censys.png", ".png")
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d", fhctx.Response.StatusCode())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct != "image/png" {
		t.Fatalf("content-type: got %q", ct)
	}
}

func TestServeStaticAsset_SVG(t *testing.T) {
	fhctx := newAssetRequest("GET", "/sectigo.svg")
	serveStaticAsset(fhctx, "sectigo.svg", ".svg")
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d", fhctx.Response.StatusCode())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct != "image/svg+xml" {
		t.Fatalf("content-type: got %q", ct)
	}
}

func TestServeStaticAsset_GIF(t *testing.T) {
	fhctx := newAssetRequest("GET", "/spinner.gif")
	serveStaticAsset(fhctx, "spinner.gif", ".gif")
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d", fhctx.Response.StatusCode())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct != "image/gif" {
		t.Fatalf("content-type: got %q", ct)
	}
}

func TestServeStaticAsset_NotFound(t *testing.T) {
	fhctx := newAssetRequest("GET", "/nonexistent.css")
	serveStaticAsset(fhctx, "nonexistent.css", ".css")
	if fhctx.Response.StatusCode() != 404 {
		t.Fatalf("status: got %d, want 404", fhctx.Response.StatusCode())
	}
}

func TestServeStaticAsset_UnsupportedExtension(t *testing.T) {
	fhctx := newAssetRequest("GET", "/file.xyz")
	serveStaticAsset(fhctx, "file.xyz", ".xyz")
	if fhctx.Response.StatusCode() != 404 {
		t.Fatalf("status: got %d, want 404", fhctx.Response.StatusCode())
	}
}

func TestServeStaticAsset_HeadRequest(t *testing.T) {
	fhctx := newAssetRequest("HEAD", "/crtsh.css")
	serveStaticAsset(fhctx, "crtsh.css", ".css")
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d", fhctx.Response.StatusCode())
	}
	if len(fhctx.Response.Body()) != 0 {
		t.Fatal("HEAD response should have no body")
	}
}

func TestServeStaticAsset_PostMethodNotAllowed(t *testing.T) {
	fhctx := newAssetRequest("POST", "/crtsh.css")
	serveStaticAsset(fhctx, "crtsh.css", ".css")
	if fhctx.Response.StatusCode() != 405 {
		t.Fatalf("status: got %d, want 405", fhctx.Response.StatusCode())
	}
}

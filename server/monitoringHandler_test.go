package server

import (
	"testing"

	"github.com/crtsh/crtsh-web/config"

	"github.com/valyala/fasthttp"
)

func newMonitoringRequest(method, path string) *fasthttp.RequestCtx {
	var ctx fasthttp.RequestCtx
	ctx.Request.SetRequestURI(path)
	ctx.Request.Header.SetMethod(method)
	return &ctx
}

func TestMonitoringHandler_Livez(t *testing.T) {
	fhctx := newMonitoringRequest("GET", "/livez")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d, want 200", fhctx.Response.StatusCode())
	}
	if string(fhctx.Response.Body()) != "OK" {
		t.Fatalf("body: got %q", fhctx.Response.Body())
	}
}

func TestMonitoringHandler_Readyz_NilPool(t *testing.T) {
	// With no database pool, readyz should return 503.
	fhctx := newMonitoringRequest("GET", "/readyz")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 503 {
		t.Fatalf("status: got %d, want 503", fhctx.Response.StatusCode())
	}
}

func TestMonitoringHandler_Metrics(t *testing.T) {
	if webServer == nil {
		t.Skip("requires running servers")
	}
	fhctx := newMonitoringRequest("GET", "/metrics")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d, want 200", fhctx.Response.StatusCode())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct == "" {
		t.Fatal("expected content-type header")
	}
	if len(fhctx.Response.Body()) == 0 {
		t.Fatal("expected non-empty metrics body")
	}
}

func TestMonitoringHandler_Favicon(t *testing.T) {
	fhctx := newMonitoringRequest("GET", "/favicon.ico")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d, want 200", fhctx.Response.StatusCode())
	}
}

func TestMonitoringHandler_NotFound(t *testing.T) {
	fhctx := newMonitoringRequest("GET", "/nonexistent")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 404 {
		t.Fatalf("status: got %d, want 404", fhctx.Response.StatusCode())
	}
}

func TestMonitoringHandler_DebugBuild_Disabled(t *testing.T) {
	config.Config.Server.EnableDebugEndpoints = false
	fhctx := newMonitoringRequest("GET", "/debug/build")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 404 {
		t.Fatalf("status: got %d, want 404 when debug disabled", fhctx.Response.StatusCode())
	}
}

func TestMonitoringHandler_DebugBuild_Enabled(t *testing.T) {
	old := config.Config.Server.EnableDebugEndpoints
	config.Config.Server.EnableDebugEndpoints = true
	defer func() { config.Config.Server.EnableDebugEndpoints = old }()

	fhctx := newMonitoringRequest("GET", "/debug/build")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d, want 200 when debug enabled", fhctx.Response.StatusCode())
	}
}

func TestMonitoringHandler_DebugConfig_Disabled(t *testing.T) {
	config.Config.Server.EnableDebugEndpoints = false
	fhctx := newMonitoringRequest("GET", "/debug/config")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 404 {
		t.Fatalf("status: got %d, want 404 when debug disabled", fhctx.Response.StatusCode())
	}
}

func TestMonitoringHandler_DebugConfig_Enabled(t *testing.T) {
	old := config.Config.Server.EnableDebugEndpoints
	config.Config.Server.EnableDebugEndpoints = true
	defer func() { config.Config.Server.EnableDebugEndpoints = old }()

	fhctx := newMonitoringRequest("GET", "/debug/config")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d, want 200 when debug enabled", fhctx.Response.StatusCode())
	}
	// Password should be redacted (json:"-").
	body := string(fhctx.Response.Body())
	if contains(body, "password") {
		t.Fatal("password field should not appear in config output")
	}
}

func TestMonitoringHandler_DebugPprof_Disabled(t *testing.T) {
	config.Config.Server.EnableDebugEndpoints = false
	fhctx := newMonitoringRequest("GET", "/debug/pprof/")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 404 {
		t.Fatalf("status: got %d, want 404 when debug disabled", fhctx.Response.StatusCode())
	}
}

func TestMonitoringHandler_DebugPprof_Enabled(t *testing.T) {
	old := config.Config.Server.EnableDebugEndpoints
	config.Config.Server.EnableDebugEndpoints = true
	defer func() { config.Config.Server.EnableDebugEndpoints = old }()

	fhctx := newMonitoringRequest("GET", "/debug/pprof/")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d, want 200 when debug enabled", fhctx.Response.StatusCode())
	}
}

func TestMonitoringHandler_CaseInsensitive(t *testing.T) {
	fhctx := newMonitoringRequest("GET", "/LIVEZ")
	monitoringHandler(fhctx)
	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("status: got %d, want 200 (case-insensitive)", fhctx.Response.StatusCode())
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package main

import (
	"context"
	"os/signal"
	"syscall"

	_ "go.uber.org/automaxprocs"

	"github.com/crtsh/crtsh-web/certwatch"
	"github.com/crtsh/crtsh-web/logger"
	"github.com/crtsh/crtsh-web/server"

	_ "github.com/valyala/quicktemplate"

	"go.uber.org/zap"
)

func main() {
	// Configure graceful shutdown capabilities.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer logger.Logger.Info("Shutting down")

	// Start the HTTP servers first so that health probes are available while the database pool initialises.
	server.Run()
	defer server.Shutdown()

	// Initialize the PostgreSQL connection pool used by the web_apis gateway (Go port of mod_certwatch).
	if err := certwatch.Init(); err != nil {
		logger.Logger.Fatal("certwatch.Init failed", zap.Error(err))
	}
	defer certwatch.Close()

	// Wait to be interrupted.
	<-ctx.Done()

	// Ensure all log messages are flushed before we exit.
	logger.Logger.Sync()
}

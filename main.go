package main

import (
	"context"
	"os/signal"
	"syscall"

	_ "go.uber.org/automaxprocs"

	"go.uber.org/zap"

	"github.com/crtsh/crtsh-web/certwatch"
	"github.com/crtsh/crtsh-web/logger"
	"github.com/crtsh/crtsh-web/server"
)

func main() {
	// Configure graceful shutdown capabilities.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer logger.Logger.Info("Shutting down")

	// Initialize the PostgreSQL connection pool used by the web_apis
	// gateway (Go port of mod_certwatch).
	if err := certwatch.Init(); err != nil {
		logger.Logger.Fatal("certwatch.Init failed", zap.Error(err))
	}
	defer certwatch.Close()

	// Start the HTTP servers (Web and Monitoring).
	server.Run()
	defer server.Shutdown()

	// Wait to be interrupted.
	<-ctx.Done()

	// Ensure all log messages are flushed before we exit.
	logger.Logger.Sync()
}

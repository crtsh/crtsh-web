package certwatch

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/crtsh/crtsh-web/config"
	"github.com/crtsh/crtsh-web/logger"

	"github.com/jackc/pgx/v5/pgxpool"

	"go.uber.org/zap"
)

// Pool is the application-wide PostgreSQL connection pool.
//
// mod_certwatch deliberately opened a new libpq connection per request and
// relied on PgBouncer for pooling. pgxpool plays nicely with PgBouncer
// (in session or transaction mode) while also avoiding the per-request
// connection-setup cost when no external pooler is present.
var Pool *pgxpool.Pool

func Init() error {
	connectString := "postgresql:///certwatch?host=" + url.QueryEscape(config.Config.CertWatchDB.Host) + "&application_name=crtsh-web@" + url.QueryEscape(config.VcsRevision) + "&user=" + url.QueryEscape(config.Config.CertWatchDB.User)
	if !strings.Contains(config.Config.CertWatchDB.Host, "/") {
		connectString += fmt.Sprintf("&port=%d", config.Config.CertWatchDB.Port)
	}
	connectStringToLog := connectString
	if config.Config.CertWatchDB.Password != "" {
		connectString += "&password=" + url.QueryEscape(config.Config.CertWatchDB.Password)
		connectStringToLog += "&password=<redacted>"
	}

	pCfg, err := pgxpool.ParseConfig(connectString)
	if err != nil {
		return err
	}
	if config.Config.Pool.MaxConns > 0 {
		pCfg.MaxConns = config.Config.Pool.MaxConns
	}
	if config.Config.Pool.MinConns > 0 {
		pCfg.MinConns = config.Config.Pool.MinConns
	}
	if config.Config.Pool.MaxConnLifetime > 0 {
		pCfg.MaxConnLifetime = config.Config.Pool.MaxConnLifetime
	}
	if config.Config.Pool.MaxConnIdleTime > 0 {
		pCfg.MaxConnIdleTime = config.Config.Pool.MaxConnIdleTime
	}

	Pool, err = pgxpool.NewWithConfig(context.Background(), pCfg)
	if err != nil {
		return err
	}

	logger.Logger.Info(
		"Database pool initialized",
		zap.Int32("max_conns", pCfg.MaxConns),
		zap.Int32("min_conns", pCfg.MinConns),
	)
	return nil
}

func Close() {
	if Pool != nil {
		logger.Logger.Info("Closing database pool")
		Pool.Close()
	}
}

func Ping(ctx context.Context) error {
	if Pool == nil {
		return nil
	}
	return Pool.Ping(ctx)
}

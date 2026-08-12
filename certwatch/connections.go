package certwatch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/crtsh/crtsh-web/config"
	"github.com/crtsh/crtsh-web/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go.uber.org/zap"
)

// DB is the interface satisfied by *pgxpool.Pool, allowing test substitution.
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Ping(ctx context.Context) error
	Close()
}

// Pool is the application-wide PostgreSQL connection pool.
var Pool DB

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
		return errors.New("database pool not initialised")
	}
	return Pool.Ping(ctx)
}

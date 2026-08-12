package config

import (
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	// config.init() runs automatically; verify key defaults.
	if Config.CertWatchDB.Host != "/var/run/postgresql" {
		t.Fatalf("certwatchdb.host: got %q", Config.CertWatchDB.Host)
	}
	if Config.CertWatchDB.Port != 5432 {
		t.Fatalf("certwatchdb.port: got %d", Config.CertWatchDB.Port)
	}
	if Config.CertWatchDB.User != "httpd" {
		t.Fatalf("certwatchdb.user: got %q", Config.CertWatchDB.User)
	}
	if Config.CertWatchDB.Password != "" {
		t.Fatal("certwatchdb.password should default to empty")
	}
}

func TestDefaults_Pool(t *testing.T) {
	if Config.Pool.MaxConns != 32 {
		t.Fatalf("pool.maxConns: got %d", Config.Pool.MaxConns)
	}
	if Config.Pool.MinConns != 0 {
		t.Fatalf("pool.minConns: got %d", Config.Pool.MinConns)
	}
	if Config.Pool.MaxConnLifetime != 30*time.Minute {
		t.Fatalf("pool.maxConnLifetime: got %v", Config.Pool.MaxConnLifetime)
	}
	if Config.Pool.MaxConnIdleTime != 5*time.Minute {
		t.Fatalf("pool.maxConnIdleTime: got %v", Config.Pool.MaxConnIdleTime)
	}
}

func TestDefaults_Server(t *testing.T) {
	if Config.Server.WebserverPort != 8080 {
		t.Fatalf("server.webserverPort: got %d", Config.Server.WebserverPort)
	}
	if Config.Server.MonitoringPort != 8081 {
		t.Fatalf("server.monitoringPort: got %d", Config.Server.MonitoringPort)
	}
	if Config.Server.ReadTimeout != 30*time.Second {
		t.Fatalf("server.readTimeout: got %v", Config.Server.ReadTimeout)
	}
	if Config.Server.IdleTimeout != 30*time.Second {
		t.Fatalf("server.idleTimeout: got %v", Config.Server.IdleTimeout)
	}
	if Config.Server.RequestTimeout != 30*time.Second {
		t.Fatalf("server.requestTimeout: got %v", Config.Server.RequestTimeout)
	}
	if Config.Server.DisableKeepalive != false {
		t.Fatal("server.disableKeepalive should default to false")
	}
	if Config.Server.EnableDebugEndpoints != false {
		t.Fatal("server.enableDebugEndpoints should default to false")
	}
	if Config.Server.MaxRequestBodySize != 1024*1024 {
		t.Fatalf("server.maxRequestBodySize: got %d", Config.Server.MaxRequestBodySize)
	}
}

func TestDefaults_Timeouts(t *testing.T) {
	if Config.Server.LivezTimeout != 500*time.Millisecond {
		t.Fatalf("server.livezTimeout: got %v", Config.Server.LivezTimeout)
	}
	if Config.Server.ReadyzTimeout != 500*time.Millisecond {
		t.Fatalf("server.readyzTimeout: got %v", Config.Server.ReadyzTimeout)
	}
	if Config.Server.RememberBusyTimeout != 5*time.Second {
		t.Fatalf("server.rememberBusyTimeout: got %v", Config.Server.RememberBusyTimeout)
	}
	if Config.Server.MetricsTimeout != 8*time.Second {
		t.Fatalf("server.metricsTimeout: got %v", Config.Server.MetricsTimeout)
	}
}

func TestDefaults_Logging(t *testing.T) {
	if Config.Logging.IsDevelopment != false {
		t.Fatal("logging.isDevelopment should default to false")
	}
	if Config.Logging.Level != "" {
		t.Fatalf("logging.level: got %q, want empty", Config.Logging.Level)
	}
}

func TestApplicationName(t *testing.T) {
	if ApplicationName == "" {
		t.Fatal("ApplicationName should not be empty")
	}
	if ApplicationNamespace == "" {
		t.Fatal("ApplicationNamespace should not be empty")
	}
}

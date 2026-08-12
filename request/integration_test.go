//go:build integration

package request

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/crtsh/crtsh-web/certwatch"
	"github.com/crtsh/crtsh-web/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valyala/fasthttp"
)

// Set CRTSHWEB_TEST_DSN to a PostgreSQL connection string to run these tests.
// Example: CRTSHWEB_TEST_DSN="postgres://user:pass@localhost:5432/testdb?sslmode=disable"

func setupTestDB(t *testing.T) func() {
	t.Helper()
	dsn := os.Getenv("CRTSHWEB_TEST_DSN")
	if dsn == "" {
		t.Skip("set CRTSHWEB_TEST_DSN to run integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION web_apis(
			p_last_segment text,
			p_names text[],
			p_values text[]
		) RETURNS bytea AS $$
		DECLARE
			v_q text;
		BEGIN
			IF p_names IS NOT NULL THEN
				FOR i IN 1..array_length(p_names, 1) LOOP
					IF p_names[i] = 'q' THEN
						v_q := p_values[i];
					END IF;
				END LOOP;
			END IF;

			IF v_q = 'empty' THEN
				RETURN NULL;
			ELSIF v_q = 'headers' THEN
				RETURN convert_to(
					E'[BEGIN_HEADERS]\nContent-Type: application/json\nX-Test: integration\n[END_HEADERS]\n{"result":"with-headers"}',
					'UTF8'
				);
			ELSIF v_q = 'error' THEN
				RAISE EXCEPTION 'deliberate test error';
			ELSIF v_q IS NOT NULL THEN
				RETURN convert_to('<html>' || v_q || '</html>', 'UTF8');
			ELSE
				RETURN convert_to('<html>default</html>', 'UTF8');
			END IF;
		END;
		$$ LANGUAGE plpgsql;
	`)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create stub web_apis: %v", err)
	}

	old := certwatch.Pool
	certwatch.Pool = pool
	config.Config.Server.RequestTimeout = 5 * time.Second

	return func() {
		pool.Exec(context.Background(), "DROP FUNCTION IF EXISTS web_apis(text, text[], text[])")
		certwatch.Pool = old
		pool.Close()
	}
}

func TestIntegration_BasicQuery(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	fhctx := newGetRequest("/?q=hello")
	result := handleWebAPIs(context.Background(), fhctx)

	if result.statusCode != fasthttp.StatusOK {
		t.Fatalf("status: got %d", result.statusCode)
	}
	if string(result.body) != "<html>hello</html>" {
		t.Fatalf("body: got %q", result.body)
	}
	if result.contentType != "text/html; charset=UTF-8" {
		t.Fatalf("content-type: got %q", result.contentType)
	}
}

func TestIntegration_EmptyResponse(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	fhctx := newGetRequest("/?q=empty")
	result := handleWebAPIs(context.Background(), fhctx)

	if !result.notFound {
		t.Fatalf("expected notFound for NULL response, got status %d", result.statusCode)
	}
}

func TestIntegration_CustomHeaders(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	fhctx := newGetRequest("/?q=headers")
	result := handleWebAPIs(context.Background(), fhctx)

	if result.statusCode != fasthttp.StatusOK {
		t.Fatalf("status: got %d", result.statusCode)
	}
	if result.contentType != "application/json" {
		t.Fatalf("content-type: got %q", result.contentType)
	}
	if len(result.headers) != 1 || result.headers[0][0] != "X-Test" || result.headers[0][1] != "integration" {
		t.Fatalf("headers: got %v", result.headers)
	}
	if string(result.body) != `{"result":"with-headers"}` {
		t.Fatalf("body: got %q", result.body)
	}
}

func TestIntegration_QueryError(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	fhctx := newGetRequest("/?q=error")
	result := handleWebAPIs(context.Background(), fhctx)

	if result.statusCode != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503", result.statusCode)
	}
	if result.err == nil {
		t.Fatal("expected error to be set")
	}
}

func TestIntegration_PostRequest(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	var fhctx fasthttp.RequestCtx
	fhctx.Request.SetRequestURI("/")
	fhctx.Request.Header.SetMethod("POST")
	fhctx.Request.SetBody([]byte("q=posted"))
	result := handleWebAPIs(context.Background(), &fhctx)

	if result.statusCode != fasthttp.StatusOK {
		t.Fatalf("status: got %d", result.statusCode)
	}
	if string(result.body) != "<html>posted</html>" {
		t.Fatalf("body: got %q", result.body)
	}
}

func TestIntegration_ContextCancellation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fhctx := newGetRequest("/?q=hello")
	result := handleWebAPIs(ctx, fhctx)

	if result.statusCode != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503", result.statusCode)
	}
}

func TestIntegration_FullFlow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	var fhctx fasthttp.RequestCtx
	fhctx.Request.SetRequestURI("/?q=fullflow")
	fhctx.Request.Header.SetMethod("GET")

	result := handleWebAPIs(context.Background(), &fhctx)
	applyWebAPIResult(&fhctx, result)

	if fhctx.Response.StatusCode() != 200 {
		t.Fatalf("response status: got %d", fhctx.Response.StatusCode())
	}
	if string(fhctx.Response.Body()) != "<html>fullflow</html>" {
		t.Fatalf("response body: got %q", fhctx.Response.Body())
	}
	if ct := string(fhctx.Response.Header.ContentType()); ct != "text/html; charset=UTF-8" {
		t.Fatalf("content-type: got %q", ct)
	}
}

func TestIntegration_NullParams(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	fhctx := newGetRequest("/atom")
	result := handleWebAPIs(context.Background(), fhctx)

	if result.statusCode != fasthttp.StatusOK {
		t.Fatalf("status: got %d", result.statusCode)
	}
}

package database

import (
	"net/url"
	"testing"
)

func TestDatabaseDSNSupportsNeonConnectionURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://portfolio-user:test-secret@ep-example-pooler.neon.tech/portfolio?sslmode=disable&channel_binding=require")
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_SSLMODE", "require")

	dsn, err := databaseDSN()
	if err != nil {
		t.Fatalf("expected valid PostgreSQL URL: %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("expected generated URL to parse: %v", err)
	}
	if parsed.Scheme != "postgresql" || parsed.Host != "ep-example-pooler.neon.tech" {
		t.Fatalf("unexpected database scheme or host: %s://%s", parsed.Scheme, parsed.Host)
	}
	if parsed.Query().Get("sslmode") != "require" {
		t.Fatal("expected TLS to remain required")
	}
	if parsed.Query().Has("channel_binding") {
		t.Fatal("unsupported channel_binding option should be removed for the migration driver")
	}
}

func TestDatabaseDSNEncodesDiscreteCredentials(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "portfolio user")
	t.Setenv("DB_PASSWORD", "pass@word/with?symbols")
	t.Setenv("DB_NAME", "portfolio db")
	t.Setenv("DB_SSLMODE", "require")

	dsn, err := databaseDSN()
	if err != nil {
		t.Fatalf("expected valid discrete configuration: %v", err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("expected generated URL to parse: %v", err)
	}
	password, ok := parsed.User.Password()
	if !ok || parsed.User.Username() != "portfolio user" || password != "pass@word/with?symbols" {
		t.Fatal("expected credentials to be encoded and preserved in the URL")
	}
	if parsed.Path != "/portfolio db" || parsed.Query().Get("sslmode") != "require" {
		t.Fatal("expected database name and TLS mode to be preserved")
	}
}

func TestDatabaseDSNRejectsInvalidConnectionURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "https://example.com/not-postgres")
	if _, err := databaseDSN(); err == nil {
		t.Fatal("expected non-PostgreSQL URL to be rejected")
	}
}

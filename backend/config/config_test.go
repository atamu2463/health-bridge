package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:password@db:5432/health_bridge")
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:3000, https://example.com, http://localhost:3000")
	t.Setenv("PORT", "9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.DatabaseURL != "postgresql://user:password@db:5432/health_bridge" {
		t.Fatal("DatabaseURL が環境変数の値と一致しません")
	}

	wantOrigins := []string{"http://localhost:3000", "https://example.com"}
	if !reflect.DeepEqual(cfg.AllowedOrigins, wantOrigins) {
		t.Fatalf("AllowedOrigins = %#v, want %#v", cfg.AllowedOrigins, wantOrigins)
	}
}

func TestLoadUsesDefaultPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:password@db:5432/health_bridge")
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != defaultPort {
		t.Fatalf("Port = %q, want %q", cfg.Port, defaultPort)
	}
}

func TestParseAllowedOriginsAcceptsCanonicalOrigins(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "HTTPとport", value: "http://localhost:3000"},
		{name: "HTTPS", value: "https://example.com"},
		{name: "HTTPSとport", value: "https://example.com:8443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origins, err := parseAllowedOrigins(tt.value)
			if err != nil {
				t.Fatalf("parseAllowedOrigins() error = %v", err)
			}
			if !reflect.DeepEqual(origins, []string{tt.value}) {
				t.Fatalf("parseAllowedOrigins() = %#v, want %#v", origins, []string{tt.value})
			}
		})
	}
}

func TestParseAllowedOriginsRejectsNonCanonicalOrigins(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "HTTP以外のscheme", value: "ftp://example.com"},
		{name: "Hostなし", value: "https://"},
		{name: "ワイルドカード", value: "*"},
		{name: "ユーザー情報", value: "https://user@example.com"},
		{name: "パス", value: "https://example.com/path"},
		{name: "末尾slash", value: "https://example.com/"},
		{name: "クエリ", value: "https://example.com?key=value"},
		{name: "空のクエリ", value: "https://example.com?"},
		{name: "フラグメント", value: "https://example.com#fragment"},
		{name: "不正なURL", value: "https://example.com/%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseAllowedOrigins(tt.value); err == nil {
				t.Fatal("parseAllowedOrigins() error = nil, want error")
			}
		})
	}
}

func TestLoadDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:password@db:5432/health_bridge")

	databaseURL, err := LoadDatabaseURL()
	if err != nil {
		t.Fatalf("LoadDatabaseURL() error = %v", err)
	}
	if databaseURL != "postgresql://user:password@db:5432/health_bridge" {
		t.Fatal("LoadDatabaseURL() が環境変数の値と一致しません")
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		databaseURL    string
		allowedOrigins string
		port           string
		wantError      string
	}{
		{
			name:           "DATABASE_URLなし",
			allowedOrigins: "http://localhost:3000",
			port:           "8080",
			wantError:      "DATABASE_URL",
		},
		{
			name:        "ALLOWED_ORIGINSなし",
			databaseURL: "postgresql://user:password@db:5432/health_bridge",
			port:        "8080",
			wantError:   "ALLOWED_ORIGINS",
		},
		{
			name:           "不正なPORT",
			databaseURL:    "postgresql://user:password@db:5432/health_bridge",
			allowedOrigins: "http://localhost:3000",
			port:           "70000",
			wantError:      "PORT",
		},
		{
			name:           "ワイルドカードOrigin",
			databaseURL:    "postgresql://user:password@db:5432/health_bridge",
			allowedOrigins: "*",
			port:           "8080",
			wantError:      "ALLOWED_ORIGINS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tt.databaseURL)
			t.Setenv("ALLOWED_ORIGINS", tt.allowedOrigins)
			t.Setenv("PORT", tt.port)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Load() error = %q, want containing %q", err, tt.wantError)
			}
		})
	}
}

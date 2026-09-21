package config

import (
	"strings"
	"testing"
)

func TestConnectDBDoesNotExposeDatabaseURLInError(t *testing.T) {
	databaseURL := "invalid-secret-database-url"

	_, err := ConnectDB(databaseURL)
	if err == nil {
		t.Fatal("ConnectDB() error = nil, want error")
	}
	if strings.Contains(err.Error(), databaseURL) {
		t.Fatalf("ConnectDB() error contains database URL: %q", err)
	}
}

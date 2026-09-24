package repository

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestRepositoryErrorDoesNotExposeDatabaseDetails(t *testing.T) {
	if err := repositoryError(gorm.ErrRecordNotFound); !errors.Is(err, ErrNotFound) {
		t.Fatalf("repositoryError() = %v, want ErrNotFound", err)
	}

	secret := "postgresql://user:password@example.invalid/database"
	err := repositoryError(errors.New(secret))
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("repositoryError() = %v, want ErrUnavailable", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("repositoryError() contains database details: %q", err)
	}
}

package repositories

import (
	"database/sql"
	"testing"
)

func TestNewTimelineRepository(t *testing.T) {
	db := &sql.DB{}
	repo := NewTimelineRepository(db)

	if repo == nil {
		t.Error("Expected repository to not be nil")
	}

	if repo.db != db {
		t.Error("Expected repository to hold the database connection")
	}
}

// Timeline repository tests would require database mocking or integration tests
// The core logic is tested through the issues_handler_test.go and timeline_handler_test.go
// which test the full stack including repository layer

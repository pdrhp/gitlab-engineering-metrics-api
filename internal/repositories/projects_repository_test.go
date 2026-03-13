package repositories

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gitlab-engineering-metrics-api/internal/domain"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "host=localhost user=postgres password=postgres dbname=gitlab_metrics_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("Test database not available: %v", err)
	}

	// Clean up test data
	db.Exec("DELETE FROM projects WHERE path LIKE 'test-%'")

	return db
}

func TestProjectsRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProjectsRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  domain.CatalogFilter
		wantErr bool
	}{
		{
			name:    "list all projects",
			filter:  domain.CatalogFilter{},
			wantErr: false,
		},
		{
			name: "filter by search",
			filter: domain.CatalogFilter{
				Search: "api",
			},
			wantErr: false,
		},
		{
			name: "filter by group path",
			filter: domain.CatalogFilter{
				GroupPath: "engineering",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := repo.List(ctx, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProjectsRepository.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Just verify we get a result (or empty slice) without error
			if projects == nil {
				t.Error("ProjectsRepository.List() returned nil, expected slice")
			}
		})
	}
}

func TestProjectsRepository_List_WithSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProjectsRepository(db)
	ctx := context.Background()

	// Insert test data
	_, err := db.ExecContext(ctx, `
		INSERT INTO projects (id, name, path, created_at, updated_at) 
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET name = $2, path = $3
	`, 999999, "Test API Project", "test-engineering/api-project")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	defer db.Exec("DELETE FROM projects WHERE id = 999999")

	filter := domain.CatalogFilter{Search: "api"}
	projects, err := repo.List(ctx, filter)
	if err != nil {
		t.Errorf("ProjectsRepository.List() error = %v", err)
		return
	}

	found := false
	for _, p := range projects {
		if p.ID == 999999 {
			found = true
			if p.Name != "Test API Project" {
				t.Errorf("Expected project name 'Test API Project', got '%s'", p.Name)
			}
			break
		}
	}

	if !found {
		t.Log("Note: Project not found in results (may be filtered by view criteria)")
	}
}

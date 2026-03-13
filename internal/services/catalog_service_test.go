package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/repositories"
)

func setupTestService(t *testing.T) (*CatalogService, *sql.DB) {
	db, err := sql.Open("postgres", "host=localhost user=postgres password=postgres dbname=gitlab_metrics_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("Test database not available: %v", err)
	}

	projectsRepo := repositories.NewProjectsRepository(db)
	groupsRepo := repositories.NewGroupsRepository(db)
	usersRepo := repositories.NewUsersRepository(db)

	service := NewCatalogService(projectsRepo, groupsRepo, usersRepo)
	return service, db
}

func TestCatalogService_ListProjects(t *testing.T) {
	service, db := setupTestService(t)
	defer db.Close()

	ctx := context.Background()
	filter := domain.CatalogFilter{}

	projects, err := service.ListProjects(ctx, filter)
	if err != nil {
		t.Errorf("CatalogService.ListProjects() error = %v", err)
		return
	}

	if projects == nil {
		t.Error("CatalogService.ListProjects() returned nil")
	}
}

func TestCatalogService_ListGroups(t *testing.T) {
	service, db := setupTestService(t)
	defer db.Close()

	ctx := context.Background()
	filter := domain.CatalogFilter{}

	groups, err := service.ListGroups(ctx, filter)
	if err != nil {
		t.Errorf("CatalogService.ListGroups() error = %v", err)
		return
	}

	if groups == nil {
		t.Error("CatalogService.ListGroups() returned nil")
	}
}

func TestCatalogService_ListUsers(t *testing.T) {
	service, db := setupTestService(t)
	defer db.Close()

	ctx := context.Background()
	filter := domain.CatalogFilter{}

	users, err := service.ListUsers(ctx, filter)
	if err != nil {
		t.Errorf("CatalogService.ListUsers() error = %v", err)
		return
	}

	if users == nil {
		t.Error("CatalogService.ListUsers() returned nil")
	}
}

func TestCatalogService_ListProjects_InvalidSearch(t *testing.T) {
	service, db := setupTestService(t)
	defer db.Close()

	ctx := context.Background()
	filter := domain.CatalogFilter{
		Search: "ab", // Too short
	}

	_, err := service.ListProjects(ctx, filter)
	if err == nil {
		t.Error("Expected error for short search term, got nil")
	}
}

package services

import (
	"context"
	"errors"
	"fmt"

	"gitlab-engineering-metrics-api/internal/domain"
)

// ProjectsRepository defines the interface for project data access
type ProjectsRepository interface {
	List(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error)
}

// GroupsRepository defines the interface for group data access
type GroupsRepository interface {
	List(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error)
}

// UsersRepository defines the interface for user data access
type UsersRepository interface {
	List(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error)
}

// CatalogService provides catalog operations
type CatalogService struct {
	projectsRepo ProjectsRepository
	groupsRepo   GroupsRepository
	usersRepo    UsersRepository
}

// NewCatalogService creates a new catalog service
func NewCatalogService(
	projectsRepo ProjectsRepository,
	groupsRepo GroupsRepository,
	usersRepo UsersRepository,
) *CatalogService {
	return &CatalogService{
		projectsRepo: projectsRepo,
		groupsRepo:   groupsRepo,
		usersRepo:    usersRepo,
	}
}

// ListProjects returns a list of projects
func (s *CatalogService) ListProjects(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error) {
	if err := s.validateFilter(filter); err != nil {
		return nil, err
	}

	projects, err := s.projectsRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}

// ListGroups returns a list of groups
func (s *CatalogService) ListGroups(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error) {
	if err := s.validateFilter(filter); err != nil {
		return nil, err
	}

	groups, err := s.groupsRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	return groups, nil
}

// ListUsers returns a list of users
func (s *CatalogService) ListUsers(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error) {
	if err := s.validateFilter(filter); err != nil {
		return nil, err
	}

	users, err := s.usersRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

// validateFilter validates the catalog filter
func (s *CatalogService) validateFilter(filter domain.CatalogFilter) error {
	if filter.Search != "" && len(filter.Search) < 3 {
		return errors.New("search term must be at least 3 characters")
	}
	return nil
}

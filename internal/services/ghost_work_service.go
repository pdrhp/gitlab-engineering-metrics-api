package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab-engineering-metrics-api/internal/domain"
)

// GhostWorkRepository defines the interface for ghost work data access
type GhostWorkRepository interface {
	GetGhostWorkIssues(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error)
}

// GhostWorkService provides ghost work metrics operations
type GhostWorkService struct {
	repo GhostWorkRepository
}

// NewGhostWorkService creates a new ghost work service
func NewGhostWorkService(repo GhostWorkRepository) *GhostWorkService {
	return &GhostWorkService{repo: repo}
}

// GetGhostWorkMetrics returns ghost work metrics
func (s *GhostWorkService) GetGhostWorkMetrics(ctx context.Context, filter domain.GhostWorkFilter) (*domain.GhostWorkMetricsResponse, error) {
	if err := s.validateFilter(filter); err != nil {
		return nil, err
	}

	// Set default pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	metrics, err := s.repo.GetGhostWorkIssues(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get ghost work metrics: %w", err)
	}

	return metrics, nil
}

// validateFilter validates the ghost work filter
func (s *GhostWorkService) validateFilter(filter domain.GhostWorkFilter) error {
	// Validate date format and range if dates are provided
	if filter.StartDate != "" || filter.EndDate != "" {
		if filter.StartDate == "" || filter.EndDate == "" {
			return errors.New("both start_date and end_date are required when filtering by date")
		}

		startDate, err := time.Parse("2006-01-02", filter.StartDate)
		if err != nil {
			return errors.New("invalid start_date format, expected YYYY-MM-DD")
		}

		endDate, err := time.Parse("2006-01-02", filter.EndDate)
		if err != nil {
			return errors.New("invalid end_date format, expected YYYY-MM-DD")
		}

		if endDate.Before(startDate) {
			return errors.New("end_date must be after start_date")
		}

		// Check date range is not too large (max 90 days)
		maxRange := 90 * 24 * time.Hour
		if endDate.Sub(startDate) > maxRange {
			return errors.New("date range cannot exceed 90 days")
		}
	}

	return nil
}

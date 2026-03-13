package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab-engineering-metrics-api/internal/domain"
)

// MetricsRepository defines the interface for metrics data access
type MetricsRepository interface {
	GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error)
	GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error)
	GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error)
}

// MetricsService provides metrics operations
type MetricsService struct {
	repo MetricsRepository
}

// NewMetricsService creates a new metrics service
func NewMetricsService(repo MetricsRepository) *MetricsService {
	return &MetricsService{repo: repo}
}

// GetDeliveryMetrics returns delivery metrics
func (s *MetricsService) GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error) {
	if err := s.validateFilter(filter); err != nil {
		return nil, err
	}

	metrics, err := s.repo.GetDeliveryMetrics(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery metrics: %w", err)
	}

	return metrics, nil
}

// GetQualityMetrics returns quality metrics
func (s *MetricsService) GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error) {
	if err := s.validateFilter(filter); err != nil {
		return nil, err
	}

	metrics, err := s.repo.GetQualityMetrics(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality metrics: %w", err)
	}

	return metrics, nil
}

// GetWipMetrics returns WIP metrics
func (s *MetricsService) GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error) {
	// WIP metrics don't need date validation but may need other validation
	if err := s.validateWIPFilter(filter); err != nil {
		return nil, err
	}

	metrics, err := s.repo.GetWipMetrics(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get WIP metrics: %w", err)
	}

	return metrics, nil
}

// validateFilter validates the metrics filter
func (s *MetricsService) validateFilter(filter domain.MetricsFilter) error {
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

// validateWIPFilter validates WIP-specific filters
func (s *MetricsService) validateWIPFilter(filter domain.MetricsFilter) error {
	// WIP metrics ignore date range, so we only validate other filters
	// Group path and project ID can be validated for proper format if needed
	return nil
}

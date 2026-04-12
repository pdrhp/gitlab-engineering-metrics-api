package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gitlab-engineering-metrics-api/internal/domain"
)

// UserLookupRepository defines the interface for looking up users by username
type UserLookupRepository interface {
	GetByUsername(ctx context.Context, username string, filter domain.CatalogFilter) (*domain.User, error)
}

// UserPerformanceMetricsService defines the interface for getting metrics for user performance
type UserPerformanceMetricsService interface {
	GetDeliveryMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.DeliveryMetricsResponse, error)
	GetQualityMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.QualityMetricsResponse, error)
	GetWipMetrics(ctx context.Context, filter domain.MetricsFilter) (*domain.WipMetricsResponse, error)
}

// IndividualPerformanceRepository defines the contract for fair individual metrics
type IndividualPerformanceRepository interface {
	GetIndividualPerformanceMetrics(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.IndividualPerformanceMetrics, error)
}

// UserPerformanceService provides user performance operations
type UserPerformanceService struct {
	usersRepo          UserLookupRepository
	metricsSvc         UserPerformanceMetricsService
	individualPerfRepo IndividualPerformanceRepository
}

// NewUserPerformanceService creates a new user performance service
func NewUserPerformanceService(
	usersRepo UserLookupRepository,
	metricsSvc UserPerformanceMetricsService,
	individualPerfRepo IndividualPerformanceRepository,
) *UserPerformanceService {
	return &UserPerformanceService{
		usersRepo:          usersRepo,
		metricsSvc:         metricsSvc,
		individualPerfRepo: individualPerfRepo,
	}
}

// Get returns user performance metrics for the given username and filter
func (s *UserPerformanceService) Get(ctx context.Context, username string, filter domain.MetricsFilter) (*domain.UserPerformanceResponse, error) {
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("username is required")
	}

	user, err := s.usersRepo.GetByUsername(ctx, username, domain.CatalogFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	filter.Assignee = username

	delivery, err := s.metricsSvc.GetDeliveryMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	quality, err := s.metricsSvc.GetQualityMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	wip, err := s.metricsSvc.GetWipMetrics(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Get fair individual performance metrics
	individualPerf, err := s.individualPerfRepo.GetIndividualPerformanceMetrics(ctx, username, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to load individual performance metrics: %w", err)
	}

	return &domain.UserPerformanceResponse{
		User: domain.UserPerformanceIdentity{
			Username:                  user.Username,
			DisplayName:               user.DisplayName,
			ActiveIssues:              user.ActiveIssues,
			CompletedIssuesLast30Days: user.CompletedIssuesLast30Days,
		},
		Period: domain.Period{
			StartDate: filter.StartDate,
			EndDate:   filter.EndDate,
		},
		Delivery: domain.UserDeliveryMetrics{
			Throughput:       delivery.Throughput,
			SpeedMetricsDays: delivery.SpeedMetricsDays,
		},
		Quality: domain.UserQualityMetrics{
			Rework:        quality.Rework,
			GhostWork:     domain.GhostWorkMetrics{RatePct: quality.ProcessHealth.BypassRatePct},
			ProcessHealth: quality.ProcessHealth,
			Bottlenecks:   quality.Bottlenecks,
			Defects:       quality.Defects,
		},
		WIP:                   *wip,
		IndividualPerformance: individualPerf,
	}, nil
}

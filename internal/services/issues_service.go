package services

import (
	"context"
	"errors"
	"fmt"

	"gitlab-engineering-metrics-api/internal/domain"
)

// IssuesRepository defines the interface for issue data access
type IssuesRepository interface {
	List(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error)
}

// TimelineRepository defines the interface for timeline data access
type TimelineRepository interface {
	GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error)
}

// IssuesService provides issue operations
type IssuesService struct {
	issuesRepo   IssuesRepository
	timelineRepo TimelineRepository
}

// NewIssuesService creates a new issues service
func NewIssuesService(issuesRepo IssuesRepository, timelineRepo TimelineRepository) *IssuesService {
	return &IssuesService{
		issuesRepo:   issuesRepo,
		timelineRepo: timelineRepo,
	}
}

// ListIssues returns a paginated list of issues
func (s *IssuesService) ListIssues(ctx context.Context, filter domain.IssuesFilter) (*domain.IssuesListResponse, error) {
	// Validate pagination parameters
	if err := s.validateFilter(filter); err != nil {
		return nil, err
	}

	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	// Call repository
	result, err := s.issuesRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	return result, nil
}

// GetTimeline returns the timeline for an issue
func (s *IssuesService) GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error) {
	// Validate issue ID
	if issueID <= 0 {
		return nil, errors.New("invalid issue ID")
	}

	// Call repository
	timeline, err := s.timelineRepo.GetTimeline(ctx, issueID)
	if err != nil {
		if err.Error() == "issue not found" {
			return nil, errors.New("issue not found")
		}
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}

	return timeline, nil
}

// validateFilter validates the issues filter
func (s *IssuesService) validateFilter(filter domain.IssuesFilter) error {
	// Validate page number
	if filter.Page < 0 {
		return errors.New("page must be greater than or equal to 0")
	}

	// Validate page size
	if filter.PageSize < 0 {
		return errors.New("page_size must be greater than or equal to 0")
	}
	if filter.PageSize > 100 {
		return errors.New("page_size cannot exceed 100")
	}

	// Validate project_id if provided
	if filter.ProjectID < 0 {
		return errors.New("project_id must be a positive integer")
	}

	// Validate issue_id if provided
	if filter.IssueID < 0 {
		return errors.New("issue_id must be a positive integer")
	}

	// Validate gitlab_issue_id if provided
	if filter.GitlabIssueID < 0 {
		return errors.New("gitlab_issue_id must be a positive integer")
	}

	// Validate issue_iid if provided
	if filter.IssueIID < 0 {
		return errors.New("issue_iid must be a positive integer")
	}

	return nil
}

package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gitlab-engineering-metrics-api/internal/domain"
)

// TimelineRepository handles database operations for issue timeline
type TimelineRepository struct {
	db *sql.DB
}

// NewTimelineRepository creates a new timeline repository
func NewTimelineRepository(db *sql.DB) *TimelineRepository {
	return &TimelineRepository{db: db}
}

// GetTimeline returns timeline events for an issue
func (r *TimelineRepository) GetTimeline(ctx context.Context, issueID int) (*domain.IssueTimelineResponse, error) {
	// Get issue summary
	summary, err := r.getIssueSummary(ctx, issueID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("issue not found")
		}
		return nil, fmt.Errorf("failed to get issue summary: %w", err)
	}

	// Get timeline events
	events, err := r.getTimelineEvents(ctx, issueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline events: %w", err)
	}

	return &domain.IssueTimelineResponse{
		Issue:    *summary,
		Timeline: events,
	}, nil
}

// getIssueSummary returns basic information about an issue
func (r *TimelineRepository) getIssueSummary(ctx context.Context, issueID int) (*domain.IssueSummary, error) {
	query := `
		SELECT 
			i.id as issue_id,
			i.gitlab_issue_id,
			i.iid as issue_iid,
			i.project_id,
			p.path as project_path,
			i.title,
			i.metadata_labels,
			i.assignees,
			COALESCE(i.current_canonical_state, 'UNKNOWN') as current_canonical_state,
			i.gitlab_created_at
		FROM issues i
		JOIN projects p ON p.id = i.project_id
		WHERE i.id = $1
	`

	var summary domain.IssueSummary
	var labelsRaw, assigneesRaw []byte

	err := r.db.QueryRowContext(ctx, query, issueID).Scan(
		&summary.IssueID,
		&summary.GitlabIssueID,
		&summary.IssueIID,
		&summary.ProjectID,
		&summary.ProjectPath,
		&summary.Title,
		&labelsRaw,
		&assigneesRaw,
		&summary.CurrentCanonicalState,
		&summary.GitlabCreatedAt,
	)
	if err != nil {
		return nil, err
	}

	labels, err := decodeJSONBStringSlice(labelsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata labels: %w", err)
	}
	assignees, err := decodeJSONBStringSlice(assigneesRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode assignees: %w", err)
	}

	summary.MetadataLabels = labels
	summary.Assignees = assignees

	return &summary, nil
}

// getTimelineEvents returns all timeline events for an issue
func (r *TimelineRepository) getTimelineEvents(ctx context.Context, issueID int) ([]domain.TimelineEvent, error) {
	query := `
		SELECT 
			type,
			timestamp,
			actor,
			from_state,
			to_state,
			duration_mins,
			is_rework,
			body
		FROM (
			-- State transitions
			SELECT 
				'state_transition' as type,
				entered_at as timestamp,
				author_name as actor,
				previous_canonical_state as from_state,
				canonical_state as to_state,
				COALESCE(duration_hours_to_next_state * 60, 0)::int as duration_mins,
				CASE 
					WHEN previous_canonical_state = 'QA_REVIEW' AND canonical_state = 'IN_PROGRESS' 
					THEN true 
					ELSE false 
				END as is_rework,
				NULL as body
			FROM vw_issue_state_transitions
			WHERE issue_id = $1

			UNION ALL

			-- Comments
			SELECT 
				'comment' as type,
				comment_timestamp as timestamp,
				author_name as actor,
				NULL as from_state,
				NULL as to_state,
				0 as duration_mins,
				false as is_rework,
				body
			FROM issue_comments
			WHERE issue_id = $1

			UNION ALL

			-- Issue creation (from issues table)
			SELECT 
				'issue_created' as type,
				gitlab_created_at as timestamp,
				NULL as actor,
				NULL as from_state,
				'OPEN' as to_state,
				0 as duration_mins,
				false as is_rework,
				NULL as body
			FROM issues
			WHERE id = $1
		) events
		ORDER BY timestamp ASC, type ASC
	`

	rows, err := r.db.QueryContext(ctx, query, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.TimelineEvent
	for rows.Next() {
		var event domain.TimelineEvent
		var timestamp time.Time
		var fromState, toState, body sql.NullString
		var actor sql.NullString

		err := rows.Scan(
			&event.Type,
			&timestamp,
			&actor,
			&fromState,
			&toState,
			&event.DurationInPreviousStateMins,
			&event.IsRework,
			&body,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeline event: %w", err)
		}

		event.Timestamp = timestamp
		if actor.Valid {
			event.Actor = actor.String
		}
		if fromState.Valid {
			event.FromState = fromState.String
		}
		if toState.Valid {
			event.ToState = toState.String
		}
		if body.Valid {
			event.Body = body.String
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

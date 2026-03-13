package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"gitlab-engineering-metrics-api/internal/domain"
	"gitlab-engineering-metrics-api/internal/observability"
)

var usersRepoLogger = observability.GetLogger().With(slog.String("repository", "users"))

// UsersRepository handles database operations for users
type UsersRepository struct {
	db *sql.DB
}

// NewUsersRepository creates a new users repository
func NewUsersRepository(db *sql.DB) *UsersRepository {
	return &UsersRepository{db: db}
}

// List returns a list of users with their issue statistics
func (r *UsersRepository) List(ctx context.Context, filter domain.CatalogFilter) ([]domain.User, error) {
	usersRepoLogger.Debug("listing users",
		slog.String("group_path", filter.GroupPath),
		slog.String("search", filter.Search),
	)

	query := `
		WITH normalized_assignees AS (
			SELECT 
				i.id as issue_id,
				p.path as project_path,
				i.current_canonical_state,
				a.username
			FROM issues i
			JOIN projects p ON p.id = i.project_id
			CROSS JOIN LATERAL jsonb_array_elements_text(` + normalizedAssigneesJSONBExpr("i.assignees") + `) as a(username)
			WHERE 1=1
	`
	var args []interface{}
	argCount := 1

	if filter.GroupPath != "" {
		query += fmt.Sprintf(" AND p.path LIKE $%d", argCount)
		args = append(args, filter.GroupPath+"/%")
		argCount++
	}

	query += `
		),
		user_stats AS (
			SELECT
				username,
				COUNT(*) FILTER (WHERE current_canonical_state NOT IN ('DONE', 'CANCELED')) as active_issues,
				COUNT(*) FILTER (
					WHERE current_canonical_state = 'DONE' 
					AND EXISTS (
						SELECT 1 FROM vw_issue_state_transitions t 
						WHERE t.issue_id = na.issue_id 
						AND t.canonical_state = 'DONE'
						AND t.entered_at >= NOW() - INTERVAL '30 days'
					)
				) as completed_last_30_days
			FROM normalized_assignees na
			GROUP BY username
		)
		SELECT 
			username,
			username as display_name,
			COALESCE(active_issues, 0) as active_issues,
			COALESCE(completed_last_30_days, 0) as completed_last_30_days
		FROM user_stats
		WHERE 1=1
	`

	if filter.Search != "" {
		query += fmt.Sprintf(" AND username ILIKE $%d", argCount)
		args = append(args, "%"+filter.Search+"%")
		argCount++
	}

	query += ` ORDER BY username`

	usersRepoLogger.Debug("executing query",
		slog.String("query", query),
		slog.Int("arg_count", len(args)),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		usersRepoLogger.Error("failed to query users",
			slog.String("error", err.Error()),
			slog.String("query", query),
		)
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.Username, &u.DisplayName, &u.ActiveIssues, &u.CompletedIssuesLast30Days); err != nil {
			usersRepoLogger.Error("failed to scan user",
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		usersRepoLogger.Error("error iterating users",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	usersRepoLogger.Info("users listed successfully",
		slog.Int("count", len(users)),
	)

	return users, nil
}

// GetByUsername returns a single user by username with their issue statistics
func (r *UsersRepository) GetByUsername(ctx context.Context, username string, filter domain.CatalogFilter) (*domain.User, error) {
	usersRepoLogger.Debug("getting user by username",
		slog.String("username", username),
		slog.String("group_path", filter.GroupPath),
	)

	query := `
		WITH normalized_assignees AS (
			SELECT 
				i.id as issue_id,
				p.path as project_path,
				i.current_canonical_state,
				a.username as assignee_username
			FROM issues i
			JOIN projects p ON p.id = i.project_id
			CROSS JOIN LATERAL jsonb_array_elements_text(` + normalizedAssigneesJSONBExpr("i.assignees") + `) as a(username)
			WHERE 1=1
	`
	var args []interface{}
	argCount := 1

	if filter.GroupPath != "" {
		query += fmt.Sprintf(" AND p.path LIKE $%d", argCount)
		args = append(args, filter.GroupPath+"/%")
		argCount++
	}

	query += `
		),
		user_stats AS (
			SELECT
				assignee_username as username,
				COUNT(*) FILTER (WHERE current_canonical_state NOT IN ('DONE', 'CANCELED')) as active_issues,
				COUNT(*) FILTER (
					WHERE current_canonical_state = 'DONE' 
					AND EXISTS (
						SELECT 1 FROM vw_issue_state_transitions t 
						WHERE t.issue_id = na.issue_id 
						AND t.canonical_state = 'DONE'
						AND t.entered_at >= NOW() - INTERVAL '30 days'
					)
				) as completed_last_30_days
			FROM normalized_assignees na
			GROUP BY assignee_username
		)
		SELECT 
			username,
			username as display_name,
			COALESCE(active_issues, 0) as active_issues,
			COALESCE(completed_last_30_days, 0) as completed_last_30_days
		FROM user_stats
		WHERE username = $` + strconv.Itoa(argCount) + `
		LIMIT 1
	`
	args = append(args, username)
	argCount++

	usersRepoLogger.Debug("executing query",
		slog.String("query", query),
		slog.Int("arg_count", len(args)),
	)

	var u domain.User
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&u.Username,
		&u.DisplayName,
		&u.ActiveIssues,
		&u.CompletedIssuesLast30Days,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			usersRepoLogger.Debug("user not found",
				slog.String("username", username),
			)
			return nil, nil
		}
		usersRepoLogger.Error("failed to query user",
			slog.String("error", err.Error()),
			slog.String("query", query),
		)
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	usersRepoLogger.Info("user retrieved successfully",
		slog.String("username", u.Username),
		slog.Int("active_issues", u.ActiveIssues),
		slog.Int("completed_issues_last_30_days", u.CompletedIssuesLast30Days),
	)

	return &u, nil
}

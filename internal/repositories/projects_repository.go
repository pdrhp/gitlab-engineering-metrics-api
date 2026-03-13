package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gitlab-engineering-metrics-api/internal/domain"
)

// ProjectsRepository handles database operations for projects
type ProjectsRepository struct {
	db *sql.DB
}

// NewProjectsRepository creates a new projects repository
func NewProjectsRepository(db *sql.DB) *ProjectsRepository {
	return &ProjectsRepository{db: db}
}

// List returns a list of projects matching the filter
func (r *ProjectsRepository) List(ctx context.Context, filter domain.CatalogFilter) ([]domain.Project, error) {
	query := `
		SELECT id, name, path, total_issues, last_synced_at 
		FROM vw_projects_catalog 
		WHERE 1=1
	`
	var args []interface{}
	var conditions []string

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR path ILIKE $%d)", len(args)+1, len(args)+1))
		args = append(args, "%"+filter.Search+"%")
	}

	if filter.GroupPath != "" {
		conditions = append(conditions, fmt.Sprintf("group_path = $%d", len(args)+1))
		args = append(args, filter.GroupPath)
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY path"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.TotalIssues, &p.LastSyncedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating projects: %w", err)
	}

	return projects, nil
}

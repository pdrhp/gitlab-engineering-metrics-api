package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab-engineering-metrics-api/internal/domain"
)

// GroupsRepository handles database operations for groups
type GroupsRepository struct {
	db *sql.DB
}

// NewGroupsRepository creates a new groups repository
func NewGroupsRepository(db *sql.DB) *GroupsRepository {
	return &GroupsRepository{db: db}
}

// List returns a list of groups derived from project paths
func (r *GroupsRepository) List(ctx context.Context, filter domain.CatalogFilter) ([]domain.Group, error) {
	query := `
		SELECT 
			split_part(path, '/', 1) as group_path,
			COUNT(*) as project_count,
			COALESCE(SUM(total_issues), 0) as total_issues,
			MAX(last_synced_at) as last_synced_at
		FROM vw_projects_catalog
		WHERE 1=1
	`
	var args []interface{}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND split_part(path, '/', 1) ILIKE $%d", len(args)+1)
		args = append(args, "%"+filter.Search+"%")
	}

	query += `
		GROUP BY split_part(path, '/', 1)
		ORDER BY group_path
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.GroupPath, &g.ProjectCount, &g.TotalIssues, &g.LastSyncedAt); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}

	return groups, nil
}

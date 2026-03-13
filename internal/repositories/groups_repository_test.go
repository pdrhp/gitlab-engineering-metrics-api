package repositories

import (
	"context"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestGroupsRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewGroupsRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  domain.CatalogFilter
		wantErr bool
	}{
		{
			name:    "list all groups",
			filter:  domain.CatalogFilter{},
			wantErr: false,
		},
		{
			name: "filter by search",
			filter: domain.CatalogFilter{
				Search: "engineering",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, err := repo.List(ctx, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GroupsRepository.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if groups == nil {
				t.Error("GroupsRepository.List() returned nil, expected slice")
			}
		})
	}
}

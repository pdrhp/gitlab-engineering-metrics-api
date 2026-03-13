package repositories

import (
	"context"
	"testing"

	"gitlab-engineering-metrics-api/internal/domain"
)

func TestUsersRepository_GetByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUsersRepository(db)
	ctx := context.Background()

	tests := []struct {
		name         string
		username     string
		filter       domain.CatalogFilter
		wantErr      bool
		wantNil      bool
		validateFunc func(t *testing.T, user *domain.User)
	}{
		{
			name:     "get user by username - happy path",
			username: "root",
			filter:   domain.CatalogFilter{},
			wantErr:  false,
			wantNil:  false,
			validateFunc: func(t *testing.T, user *domain.User) {
				if user == nil {
					t.Error("expected user, got nil")
					return
				}
				if user.Username != "root" {
					t.Errorf("expected username 'root', got '%s'", user.Username)
				}
				// Verify DisplayName is populated (same as username)
				if user.DisplayName == "" {
					t.Error("expected DisplayName to be non-empty")
				}
				// ActiveIssues and CompletedIssuesLast30Days should be populated (>= 0)
				if user.ActiveIssues < 0 {
					t.Errorf("expected ActiveIssues >= 0, got %d", user.ActiveIssues)
				}
				if user.CompletedIssuesLast30Days < 0 {
					t.Errorf("expected CompletedIssuesLast30Days >= 0, got %d", user.CompletedIssuesLast30Days)
				}
			},
		},
		{
			name:     "get user by username - not found",
			username: "nonexistent-user-xyz123",
			filter:   domain.CatalogFilter{},
			wantErr:  false,
			wantNil:  true,
			validateFunc: func(t *testing.T, user *domain.User) {
				if user != nil {
					t.Errorf("expected nil user, got %+v", user)
				}
			},
		},
		{
			name:     "get user by username with group path filter - user not in group",
			username: "root",
			filter: domain.CatalogFilter{
				GroupPath: "nonexistent-group-xyz123",
			},
			wantErr: false,
			wantNil: true,
			validateFunc: func(t *testing.T, user *domain.User) {
				if user != nil {
					t.Errorf("expected nil user when user not in group, got %+v", user)
				}
			},
		},
		{
			name:     "get user by username with group path filter - user in group",
			username: "root",
			filter: domain.CatalogFilter{
				GroupPath: "engineering",
			},
			wantErr: false,
			wantNil: false,
			validateFunc: func(t *testing.T, user *domain.User) {
				if user == nil {
					t.Error("expected user, got nil")
					return
				}
				if user.Username != "root" {
					t.Errorf("expected username 'root', got '%s'", user.Username)
				}
				// Verify DisplayName is populated
				if user.DisplayName == "" {
					t.Error("expected DisplayName to be non-empty")
				}
			},
		},
		{
			name:     "get user by username - empty username",
			username: "",
			filter:   domain.CatalogFilter{},
			wantErr:  false,
			wantNil:  true,
			validateFunc: func(t *testing.T, user *domain.User) {
				if user != nil {
					t.Errorf("expected nil user for empty username, got %+v", user)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByUsername(ctx, tt.username, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("UsersRepository.GetByUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil && user != nil {
				t.Errorf("UsersRepository.GetByUsername() expected nil, got %+v", user)
				return
			}
			if !tt.wantNil && user == nil && !tt.wantErr {
				t.Errorf("UsersRepository.GetByUsername() expected non-nil user, got nil")
				return
			}
			if tt.validateFunc != nil {
				tt.validateFunc(t, user)
			}
		})
	}
}

func TestUsersRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUsersRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		filter  domain.CatalogFilter
		wantErr bool
	}{
		{
			name:    "list all users",
			filter:  domain.CatalogFilter{},
			wantErr: false,
		},
		{
			name: "filter by search",
			filter: domain.CatalogFilter{
				Search: "john",
			},
			wantErr: false,
		},
		{
			name: "filter by group path",
			filter: domain.CatalogFilter{
				GroupPath: "engineering",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, err := repo.List(ctx, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("UsersRepository.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if users == nil {
				t.Error("UsersRepository.List() returned nil, expected slice")
			}
		})
	}
}

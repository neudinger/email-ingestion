package storage

import (
	"context"
	"testing"
	"time"

	"main/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) Repository {
	repo, err := NewDuckDBRepository(":memory:")
	require.NoError(t, err)
	return repo
}

func TestDuckDBRepository_InitSchema(t *testing.T) {
	repo := setupTestDB(t)
	assert.NotNil(t, repo)
}

func TestDuckDBRepository_SaveAndGetUsers(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	users := []domain.User{
		{ID: uuid.New(), TenantID: tenantID, ExternalUserID: "ext-1", Email: "test1@example.com", Provider: domain.ProviderGoogle},
		{ID: uuid.New(), TenantID: tenantID, ExternalUserID: "ext-2", Email: "test2@example.com", Provider: domain.ProviderGoogle},
	}

	err := repo.SaveUsers(ctx, users)
	require.NoError(t, err)

	dbUsers, err := repo.GetUsersByTenant(ctx, tenantID)
	require.NoError(t, err)
	assert.Len(t, dbUsers, 2)
	assert.Equal(t, "test1@example.com", dbUsers[0].Email)
}

func TestDuckDBRepository_GetLastSyncTime(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// 1. Test empty state
	lastSync, err := repo.GetLastSyncTime(ctx, tenantID, domain.ProviderMicrosoft)
	require.NoError(t, err)
	assert.True(t, lastSync.IsZero())

	// 2. Test after saving an email
	t1 := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	email := domain.Email{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ReceivedAt: t1,
		Provider:   domain.ProviderMicrosoft,
	}
	err = repo.SaveEmails(ctx, []domain.Email{email})
	require.NoError(t, err)

	lastSync, err = repo.GetLastSyncTime(ctx, tenantID, domain.ProviderMicrosoft)
	require.NoError(t, err)
	assert.Equal(t, t1, lastSync.Local())
}

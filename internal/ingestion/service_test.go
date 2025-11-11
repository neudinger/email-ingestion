package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"

	"main/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) InitSchema(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockRepository) GetLastSyncTime(ctx context.Context, tenantID uuid.UUID, provider domain.Provider) (time.Time, error) {
	args := m.Called(ctx, tenantID, provider)
	return args.Get(0).(time.Time), args.Error(1)
}
func (m *MockRepository) SaveUsers(ctx context.Context, users []domain.User) error {
	args := m.Called(ctx, users)
	return args.Error(0)
}
func (m *MockRepository) SaveEmails(ctx context.Context, emails []domain.Email) error {
	args := m.Called(ctx, emails)
	return args.Error(0)
}
func (m *MockRepository) GetUsersByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]domain.User), args.Error(1)
}

type MockProviderClient struct {
	mock.Mock
}

func (m *MockProviderClient) GetUsers(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]domain.User), args.Error(1)
}
func (m *MockProviderClient) GetEmails(ctx context.Context, tenantID uuid.UUID, externalUserID string, receivedAfter time.Time) ([]domain.Email, error) {
	args := m.Called(ctx, tenantID, externalUserID, receivedAfter)
	return args.Get(0).([]domain.Email), args.Error(1)
}

func TestService_SyncTenant_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockProvider := new(MockProviderClient)

	// Create service and inject mock provider
	service := NewService(mockRepo)
	service.providers[domain.ProviderGoogle] = mockProvider

	tenantID := uuid.New()
	provider := domain.ProviderGoogle
	zeroTime := time.Time{}

	mockUser := domain.User{
		ID: uuid.New(), TenantID: tenantID, ExternalUserID: "ext-123", Provider: provider,
	}
	mockEmail := domain.Email{
		ID: uuid.New(), TenantID: tenantID, UserID: mockUser.ID, Provider: provider,
	}

	mockRepo.On("GetLastSyncTime", ctx, tenantID, provider).Return(zeroTime, nil)
	mockProvider.On("GetUsers", ctx, tenantID).Return([]domain.User{mockUser}, nil)
	mockRepo.On("SaveUsers", ctx, []domain.User{mockUser}).Return(nil)
	mockRepo.On("GetUsersByTenant", ctx, tenantID).Return([]domain.User{mockUser}, nil) // Return user with internal ID
	mockProvider.On("GetEmails", ctx, tenantID, mockUser.ExternalUserID, zeroTime).Return([]domain.Email{mockEmail}, nil)
	mockRepo.On("SaveEmails", ctx, mock.AnythingOfType("[]domain.Email")).Return(nil)

	err := service.SyncTenant(ctx, tenantID, provider)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

func TestService_SyncTenant_ProviderError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	tenantID := uuid.New()

	mockRepo.On("GetLastSyncTime", ctx, tenantID, domain.ProviderMicrosoft).Return(time.Time{}, nil)
	mockRepo.On("GetUsersByTenant", ctx, tenantID).Return([]domain.User{}, nil)

	mockProvider := new(MockProviderClient)
	service.providers[domain.ProviderMicrosoft] = mockProvider
	mockProvider.On("GetUsers", ctx, tenantID).Return([]domain.User{}, errors.New("API failed"))

	err := service.SyncTenant(ctx, tenantID, domain.ProviderMicrosoft)

	assert.Error(t, err)
	assert.Equal(t, "API failed", err.Error())
}

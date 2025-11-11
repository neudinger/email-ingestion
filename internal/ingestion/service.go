package ingestion

import (
	"context"
	"errors"
	"log/slog"

	"main/internal/domain"
	"main/internal/storage"

	"github.com/google/uuid"
)

type Service struct {
	repo      storage.Repository
	providers map[domain.Provider]ProviderClient
}

func NewService(repo storage.Repository) *Service {
	providers := map[domain.Provider]ProviderClient{
		domain.ProviderMicrosoft: NewMicrosoftProvider(),
		domain.ProviderGoogle:    NewGoogleProvider(),
	}
	return &Service{
		repo:      repo,
		providers: providers,
	}
}

func (s *Service) SyncTenant(ctx context.Context, tenantID uuid.UUID, provider domain.Provider) error {
	client, ok := s.providers[provider]
	if !ok {
		return errors.New("provider not supported")
	}

	log := slog.With("tenant_id", tenantID.String(), "provider", provider)
	log.Info("Starting tenant sync")

	lastSyncTime, err := s.repo.GetLastSyncTime(ctx, tenantID, provider)
	if err != nil {
		log.Error("Failed to get last sync time", "err", err)
		return err
	}
	log.Info("Last sync time", "time", lastSyncTime.String())

	users, err := client.GetUsers(ctx, tenantID)
	if err != nil {
		log.Error("Failed to get users", "err", err)
		return err
	}
	if err := s.repo.SaveUsers(ctx, users); err != nil {
		log.Error("Failed to save users", "err", err)
		return err
	}
	log.Info("Synced users", "count", len(users))

	dbUsers, err := s.repo.GetUsersByTenant(ctx, tenantID)
	if err != nil {
		log.Error("Failed to get users from DB", "err", err)
		return err
	}

	userMap := make(map[string]uuid.UUID)
	for _, u := range dbUsers {
		userMap[u.ExternalUserID] = u.ID
	}

	var totalEmails int
	for _, u := range users {
		internalUserID, ok := userMap[u.ExternalUserID]
		if !ok {
			log.Warn("User not found in DB after save, skipping emails", "external_user_id", u.ExternalUserID)
			continue
		}

		emails, err := client.GetEmails(ctx, tenantID, u.ExternalUserID, lastSyncTime)
		if err != nil {
			log.Warn("Failed to get emails for user", "user_email", u.Email, "err", err)
			continue
		}

		if len(emails) > 0 {
			for i := range emails {
				emails[i].UserID = internalUserID
			}

			if err := s.repo.SaveEmails(ctx, emails); err != nil {
				log.Error("Failed to save emails batch", "err", err)
				return err
			}
			totalEmails += len(emails)
		}
	}

	log.Info("Tenant sync complete", "total_new_emails", totalEmails)
	return nil
}

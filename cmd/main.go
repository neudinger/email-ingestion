package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"main/internal/domain"
	"main/internal/ingestion"
	"main/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dbPath := "/data/security.db" 
	
	repo, err := storage.NewDuckDBRepository(dbPath)
	if err != nil {
		slog.Error("Failed to init storage", "err", err)
		os.Exit(1)
	}

	service := ingestion.NewService(repo)
	
	mockTenantID := uuid.New()
	mockProvider := domain.ProviderMicrosoft

	if err := service.SyncTenant(context.Background(), mockTenantID, mockProvider); err != nil {
		slog.Error("Sync failed", "err", err)
		os.Exit(1)
	}

	slog.Info("Sync successful")
}
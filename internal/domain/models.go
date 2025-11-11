package domain

import (
	"time"

	"github.com/google/uuid"
)

type Provider string

const (
	ProviderGoogle    Provider = "google"
	ProviderMicrosoft Provider = "microsoft"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ExternalUserID string    `json:"external_user_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	Provider       Provider  `json:"provider"`
}

type Email struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	UserID            uuid.UUID `json:"user_id"`
	ExternalMessageID string    `json:"external_message_id"`
	From              string    `json:"from"`
	To                []string  `json:"to"`
	Cc                []string  `json:"cc"`
	Bcc               []string  `json:"bcc"`
	Subject           string    `json:"subject"`
	Body              string    `json:"body"`
	ReceivedAt        time.Time `json:"received_at"`
	Provider          Provider  `json:"provider"`
}

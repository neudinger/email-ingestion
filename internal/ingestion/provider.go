package ingestion

import (
	"context"
	"time"

	"main/internal/domain"

	"github.com/google/uuid"
)

type MicrosoftUser struct {
	ID   uuid.UUID
	Name string
	Mail string
}
type MicrosoftEmail struct {
	MessageID string
	From      struct{ EmailAddress struct{ Address string } }
	To        []struct{ EmailAddress struct{ Address string } }
	Received  time.Time
	Subject   string
	Body      string
}
type GoogleUser struct {
	ID       uuid.UUID
	FullName string
	Email    string
}
type GoogleEmail struct {
	MessageID string
	From      string
	To        []string
	Received  time.Time
	Subject   string
	Body      string
}

func GetMicrosoftUsers(tenantID uuid.UUID) ([]MicrosoftUser, error) {
	return []MicrosoftUser{
		{ID: uuid.New(), Name: "Satya Nadella", Mail: "satya@msft.example.com"},
		{ID: uuid.New(), Name: "Phil Spencer", Mail: "phil@msft.example.com"},
	}, nil
}
func GetMicrosoftEmails(userID uuid.UUID, receivedAfter time.Time) ([]MicrosoftEmail, error) {
	return []MicrosoftEmail{
		{
			MessageID: "msft-123",
			From:      struct{ EmailAddress struct{ Address string } }{struct{ Address string }{"bill@external.com"}},
			To: []struct{ EmailAddress struct{ Address string } }{
				{EmailAddress: struct{ Address string }{Address: "satya@msft.example.com"}},
			},
			Received: time.Now().Add(-1 * time.Hour),
			Subject:  "Urgent Invoice",
			Body:     "Please pay this immediately.",
		},
	}, nil
}
func GetGoogleUsers(tenantID uuid.UUID) ([]GoogleUser, error) {
	return []GoogleUser{
		{ID: uuid.New(), FullName: "Sundar Pichai", Email: "sundar@google.example.com"},
	}, nil
}
func GetGoogleEmails(userID uuid.UUID, receivedAfter time.Time) ([]GoogleEmail, error) {
	return []GoogleEmail{
		{
			MessageID: "goog-456",
			From:      "larry@external.com",
			To:        []string{"sundar@google.example.com"},
			Received:  time.Now().Add(-2 * time.Hour),
			Subject:   "Confidential Request",
			Body:      "Buy gift cards.",
		},
	}, nil
}

type ProviderClient interface {
	GetUsers(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error)
	GetEmails(ctx context.Context, tenantID uuid.UUID, externalUserID string, receivedAfter time.Time) ([]domain.Email, error)
}

type microsoftProvider struct{}

func NewMicrosoftProvider() ProviderClient {
	return &microsoftProvider{}
}

func (p *microsoftProvider) GetUsers(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error) {
	msUsers, err := GetMicrosoftUsers(tenantID)
	if err != nil {
		return nil, err
	}

	users := make([]domain.User, len(msUsers))
	for i, u := range msUsers {
		users[i] = domain.User{
			ID:             uuid.New(),
			TenantID:       tenantID,
			ExternalUserID: u.ID.String(),
			Email:          u.Mail,
			Name:           u.Name,
			Provider:       domain.ProviderMicrosoft,
		}
	}
	return users, nil
}

func (p *microsoftProvider) GetEmails(ctx context.Context, tenantID uuid.UUID, externalUserID string, receivedAfter time.Time) ([]domain.Email, error) {
	uid, _ := uuid.Parse(externalUserID)
	msEmails, err := GetMicrosoftEmails(uid, receivedAfter)
	if err != nil {
		return nil, err
	}

	emails := make([]domain.Email, len(msEmails))
	for i, e := range msEmails {
		tos := make([]string, len(e.To))
		for j, to := range e.To {
			tos[j] = to.EmailAddress.Address
		}

		emails[i] = domain.Email{
			ID:                uuid.New(),
			TenantID:          tenantID,
			ExternalMessageID: e.MessageID,
			From:              e.From.EmailAddress.Address,
			To:                tos,
			ReceivedAt:        e.Received,
			Subject:           e.Subject,
			Body:              e.Body,
			Provider:          domain.ProviderMicrosoft,
		}
	}
	return emails, nil
}

type googleProvider struct{}

func NewGoogleProvider() ProviderClient {
	return &googleProvider{}
}

func (p *googleProvider) GetUsers(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error) {
	gUsers, err := GetGoogleUsers(tenantID)
	if err != nil {
		return nil, err
	}

	users := make([]domain.User, len(gUsers))
	for i, u := range gUsers {
		users[i] = domain.User{
			ID:             uuid.New(),
			TenantID:       tenantID,
			ExternalUserID: u.ID.String(),
			Email:          u.Email,
			Name:           u.FullName,
			Provider:       domain.ProviderGoogle,
		}
	}
	return users, nil
}

func (p *googleProvider) GetEmails(ctx context.Context, tenantID uuid.UUID, externalUserID string, receivedAfter time.Time) ([]domain.Email, error) {
	return []domain.Email{}, nil
}

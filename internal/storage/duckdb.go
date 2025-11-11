package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"main/internal/domain"

	"github.com/duckdb/duckdb-go/v2"
	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/google/uuid"
)

type Repository interface {
	InitSchema(ctx context.Context) error
	GetLastSyncTime(ctx context.Context, tenantID uuid.UUID, provider domain.Provider) (time.Time, error)
	SaveUsers(ctx context.Context, users []domain.User) error
	SaveEmails(ctx context.Context, emails []domain.Email) error
	GetUsersByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error)
}

type duckDBRepo struct {
	db *sql.DB
}

func NewDuckDBRepository(dsn string) (Repository, error) {
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	repo := &duckDBRepo{db: db}
	if err := repo.InitSchema(context.Background()); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *duckDBRepo) InitSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR PRIMARY KEY,
			tenant_id VARCHAR,
			external_user_id VARCHAR,
			email VARCHAR,
			name VARCHAR,
			provider VARCHAR,
			UNIQUE(tenant_id, external_user_id, provider)
		)`,
		`CREATE TABLE IF NOT EXISTS emails (
			id VARCHAR PRIMARY KEY,
			tenant_id VARCHAR,
			user_id VARCHAR,
			external_message_id VARCHAR,
			from_email VARCHAR,
			to_emails VARCHAR[], -- Use array type
			cc_emails VARCHAR[],
			bcc_emails VARCHAR[],
			subject VARCHAR,
			body TEXT,
			received_at TIMESTAMP,
			provider VARCHAR,
			UNIQUE(tenant_id, external_message_id, provider)
		)`,
	}

	for _, query := range queries {
		if _, err := r.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func (r *duckDBRepo) GetLastSyncTime(ctx context.Context, tenantID uuid.UUID, provider domain.Provider) (time.Time, error) {
	var lastTime sql.NullTime
	query := `SELECT MAX(received_at) FROM emails WHERE tenant_id = ? AND provider = ?`

	err := r.db.QueryRowContext(ctx, query, tenantID.String(), string(provider)).Scan(&lastTime)
	if err != nil && err != sql.ErrNoRows {
		return time.Time{}, err
	}
	if !lastTime.Valid {
		return time.Time{}, nil
	}
	return lastTime.Time, nil
}


func (r *duckDBRepo) SaveUsers(ctx context.Context, users []domain.User) error {
	if len(users) == 0 {
		return nil
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var app *duckdb.Appender

	if err := conn.Raw(func(dc any) error {
		duckConn, ok := dc.(*duckdb.Conn)
		if !ok {
			return fmt.Errorf("not a duckdb Conn: %T", dc)
		}

		a, err := duckdb.NewAppenderFromConn(duckConn, "", "users")
		if err != nil {
			return err
		}
		app = a
		return nil
	}); err != nil {
		return err
	}
	defer app.Close()

	for _, u := range users {
		if err := app.AppendRow(
			u.ID.String(),
			u.TenantID.String(),
			u.ExternalUserID,
			u.Email,
			u.Name,
			string(u.Provider),
		); err != nil {
			slog.Warn("Failed to append user, skipping", "err", err, "user_id", u.ID)
		}
	}
	return app.Flush()
}

func (r *duckDBRepo) SaveEmails(ctx context.Context, emails []domain.Email) error {
	if len(emails) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO emails (id, tenant_id, user_id, external_message_id, from_email, to_emails, received_at, subject, body, provider)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (tenant_id, external_message_id, provider) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range emails {
		_, err := stmt.ExecContext(ctx,
			e.ID.String(),
			e.TenantID.String(),
			e.UserID.String(),
			e.ExternalMessageID,
			e.From,
			e.To,
			e.ReceivedAt,
			e.Subject,
			e.Body,
			string(e.Provider),
		)
		if err != nil {
			slog.Warn("Failed to insert email, skipping", "err", err, "msg_id", e.ExternalMessageID)
		}
	}
	return tx.Commit()
}

func (r *duckDBRepo) GetUsersByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, tenant_id, external_user_id, email, name, provider FROM users WHERE tenant_id = ?", tenantID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var provider string
		if err := rows.Scan(&u.ID, &u.TenantID, &u.ExternalUserID, &u.Email, &u.Name, &provider); err != nil {
			return nil, err
		}
		u.Provider = domain.Provider(provider)
		users = append(users, u)
	}
	return users, nil
}

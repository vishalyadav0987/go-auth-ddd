package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

type UserRepository struct {
	db *sql.DB
}

// constructor
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Save implements domain.UserRepository
func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (
			id, client_id, email, mobile, password_hash, mpin_hash,
			has_mpin, is_verified, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID(),
		user.ClientID(),
		string(user.Email()),
		string(user.Mobile()),
		user.PasswordHash(),
		user.MPINHash(),
		user.IsHashMpin(),
		user.IsVerified(),
		user.CreatedAt(),
		user.UpdatedAt(),
	)

	return err
}

func (r *UserRepository) scanUser(row *sql.Row) (*domain.User, error) {
	var (
		id           string
		clientID     string
		emailStr     sql.NullString
		mobileStr    sql.NullString
		passwordHash sql.NullString
		mpinHash     sql.NullString
		hasMpin      bool
		isVerified   bool
		role         string
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := row.Scan(
		&id,
		&clientID,
		&emailStr,
		&mobileStr,
		&passwordHash,
		&mpinHash,
		&hasMpin,
		&isVerified,
		&role,
		&createdAt,
		&updatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	emailValidate, _ := domain.NewEmail(emailStr.String)
	mobileValidate, _ := domain.NewMobile(mobileStr.String)

	user := domain.NewUser(
		id,
		clientID,
		emailValidate,
		mobileValidate,
		passwordHash.String,
		mpinHash.String,
		hasMpin,
	)

	if isVerified {
		user.MarkVerified()
	}

	if err := user.AssignRole(domain.Role(role)); err != nil {
		return nil, fmt.Errorf("load user %s: %w", id, err)
	}

	return user, nil
}

// FindByEmail implements domain.UserRepository
func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	query := `
		SELECT id, client_id, email, mobile, password_hash, mpin_hash,
		       has_mpin, is_verified, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	row := r.db.QueryRowContext(ctx, query, string(email))
	return r.scanUser(row)
}

// FindByMobile implements domain.UserRepository
func (r *UserRepository) FindByMobile(ctx context.Context, mobile domain.Mobile) (*domain.User, error) {
	query := `
		SELECT id, client_id, email, mobile, password_hash, mpin_hash,
		       has_mpin, is_verified, role, created_at, updated_at
		FROM users
		WHERE mobile = $1
	`

	row := r.db.QueryRowContext(ctx, query, string(mobile))
	return r.scanUser(row)
}

// FindByClientId implements domain.UserRepository
func (r *UserRepository) FindByClientId(ctx context.Context, clientId string) (*domain.User, error) {
	query := `
		SELECT id, client_id, email, mobile, password_hash, mpin_hash,
		       has_mpin, is_verified, role, created_at, updated_at
		FROM users
		WHERE client_id = $1
	`

	row := r.db.QueryRowContext(ctx, query, clientId)
	return r.scanUser(row)
}

// FindById implements domain.UserRepository
func (r *UserRepository) FindById(ctx context.Context, userId string) (*domain.User, error) {
	query := `
		SELECT id, client_id, email, mobile, password_hash, mpin_hash,
		       has_mpin, is_verified, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, userId)
	return r.scanUser(row)
}

// Update implements domain.UserRepository
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, 
			mobile = $2, 
			password_hash = $3, 
			mpin_hash = $4, 
			is_verified = $5, 
			updated_at = $6
		WHERE id = $7
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		string(user.Email()),
		string(user.Mobile()),
		user.PasswordHash(),
		user.MPINHash(),
		user.IsVerified(),
		user.UpdatedAt(),
		user.ID(),
	)

	return err
}

// UpdateMPIN implements domain.UserRepository
func (r *UserRepository) UpdateMPIN(ctx context.Context, userId string, mpinHash string) error {
	query := `
		UPDATE users
		SET mpin_hash = $1, updated_at = $2, has_mpin = true
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, mpinHash, time.Now().UTC(), userId)
	if err != nil {
		return fmt.Errorf("update mpin: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("user not found")
	}

	return nil
}

// UpdatePassword implements domain.UserRepository
func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, hashedPassword string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, hashedPassword, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("user not found")
	}

	return nil
}

// GetEmailsByUserIDs returns email addresses for multiple users.
//
// Users with NULL or empty email addresses are excluded.
// The returned map uses user ID as the key and email as the value.
func (r *UserRepository) GetEmailsByUserIDs(
	ctx context.Context,
	userIDs []string,
) (map[string]string, error) {
	emails := make(map[string]string)

	if r == nil || r.db == nil {
		return nil, errors.New("user database is not configured")
	}

	if len(userIDs) == 0 {
		return emails, nil
	}

	// Remove empty IDs and duplicates before building the query.
	uniqueIDs := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))

	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)

		if userID == "" {
			continue
		}

		if _, exists := seen[userID]; exists {
			continue
		}

		seen[userID] = struct{}{}
		uniqueIDs = append(uniqueIDs, userID)
	}

	if len(uniqueIDs) == 0 {
		return emails, nil
	}

	// One array parameter instead of one placeholder per user: an
	// all-users broadcast would otherwise exceed PostgreSQL's 65,535
	// parameter limit.
	const query = `
		SELECT id, email
		FROM users
		WHERE id = ANY($1)
		  AND email IS NOT NULL
		  AND BTRIM(email) <> ''
	`

	rows, err := r.db.QueryContext(ctx, query, uniqueIDs)
	if err != nil {
		return nil, fmt.Errorf("get user emails: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			userID string
			email  string
		)

		if err := rows.Scan(&userID, &email); err != nil {
			return nil, fmt.Errorf("scan user email: %w", err)
		}

		emails[userID] = strings.TrimSpace(email)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user emails: %w", err)
	}

	return emails, nil
}

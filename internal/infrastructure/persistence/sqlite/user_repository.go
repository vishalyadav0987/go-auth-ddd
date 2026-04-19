package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// ✅ Implement Save
// implements to UserRepo interface
func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {

	query := "INSERT INTO users (id, client_id, email ,mobile, password_hash, mpin_hash, has_mpin, is_verified, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID(),
		user.ClientID(),
		user.Email(),
		user.Mobile(),
		user.PasswordHash(),
		user.MPINHash(),
		user.IsHashMpin(),
		user.IsVerified(),
		user.CreatedAt(),
		user.UpdatedAt(),
	)

	return err
}

// ✅ Implement FindByEmail
// implements to UserRepo interface
func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	query := "SELECT id, client_id, email, mobile, password_hash, mpin_hash, has_mpin, is_verified, created_at, updated_at FROM users WHERE email = $1"

	row := r.db.QueryRowContext(ctx, query, string(email))

	var (
		id           string
		clientID     string
		emailStr     string
		mobileStr    string
		passwordHash string
		mpinHash     string
		hasMpin      bool
		isVerified   bool
		createdAt    string
		updatedAt    string
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
		&createdAt,
		&updatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	emailValidate, _ := domain.NewEmail(emailStr)
	mobileValidate, _ := domain.NewMobile(mobileStr)

	user := domain.NewUser(
		id, clientID,
		emailValidate,
		mobileValidate,
		passwordHash,
		mpinHash,
		hasMpin,
	)

	if isVerified {
		user.MarkVerified()
	}

	return user, nil
}

func (r *UserRepository) FindByMobile(ctx context.Context, mobile domain.Mobile) (*domain.User, error) {
	query := "SELECT id, client_id, email, mobile, password_hash, mpin_hash, has_mpin, is_verified, created_at, updated_at FROM users WHERE mobile = $1"

	row := r.db.QueryRowContext(ctx, query, string(mobile))

	var (
		id           string
		clientID     string
		emailStr     string
		mobileStr    string
		passwordHash string
		mpinHash     string
		hasMpin      bool
		isVerified   bool
		createdAt    string
		updatedAt    string
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
		&createdAt,
		&updatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	emailValidate, _ := domain.NewEmail(emailStr)
	mobileValidate, _ := domain.NewMobile(mobileStr)

	user := domain.NewUser(
		id, clientID,
		emailValidate,
		mobileValidate,
		passwordHash,
		mpinHash,
		hasMpin,
	)

	if isVerified {
		user.MarkVerified()
	}

	return user, nil
}

func (r *UserRepository) FindByClientId(ctx context.Context, clientId string) (*domain.User, error) {
	query := "SELECT id, client_id, email, mobile, password_hash, mpin_hash, has_mpin, is_verified, created_at, updated_at FROM users WHERE client_id = $1"

	row := r.db.QueryRowContext(ctx, query, clientId)

	var (
		id           string
		clientID     string
		emailStr     string
		mobileStr    string
		passwordHash string
		mpinHash     string
		hasMpin      bool
		isVerified   bool
		createdAt    string
		updatedAt    string
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
		&createdAt,
		&updatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	emailValidate, _ := domain.NewEmail(emailStr)
	mobileValidate, _ := domain.NewMobile(mobileStr)

	user := domain.NewUser(
		id, clientID,
		emailValidate,
		mobileValidate,
		passwordHash,
		mpinHash,
		hasMpin,
	)

	if isVerified {
		user.MarkVerified()
	}

	return user, nil
}

func (r *UserRepository) FindById(ctx context.Context, userId string) (*domain.User, error) {
	query := "SELECT id, client_id, email, mobile, password_hash, mpin_hash, has_mpin, is_verified, created_at, updated_at FROM users WHERE id = $1"

	row := r.db.QueryRowContext(ctx, query, userId)

	var (
		id           string
		clientID     string
		emailStr     string
		mobileStr    string
		passwordHash string
		mpinHash     string
		hasMpin      bool
		isVerified   bool
		createdAt    string
		updatedAt    string
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
		&createdAt,
		&updatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	emailValidate, _ := domain.NewEmail(emailStr)
	mobileValidate, _ := domain.NewMobile(mobileStr)

	user := domain.NewUser(
		id, clientID,
		emailValidate,
		mobileValidate,
		passwordHash,
		mpinHash,
		hasMpin,
	)

	if isVerified {
		user.MarkVerified()
	}

	return user, nil
}

func (r *UserRepository) Update(
	ctx context.Context,
	user *domain.User,
) error {

	query := `
	UPDATE users
	SET email = ?, 
		mobile = ?, 
		password_hash = ?, 
		mpin_hash = ?, 
		is_verified = ?, 
		updated_at = ?
	WHERE id = ?
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.Email(),
		user.Mobile(),
		user.PasswordHash(),
		user.MPINHash(),
		user.IsVerified(),
		user.UpdatedAt(),
		user.ID(),
	)

	return err
}

func (r *UserRepository) UpdateMPIN(
	ctx context.Context,
	userId string,
	mpinHash string,
) error {
	query := `
        UPDATE users
        SET mpin_hash = ?, updated_at = ?, has_mpin = true
        WHERE id = ?
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

func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, hashedPassword string) error {

	query := `
	UPDATE users
	SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
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

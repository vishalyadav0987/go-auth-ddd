package authapp

import (
	"context"

	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

type ResetPasswordUsecase struct {
	repo   domain.UserRepository
	hasher PasswordHasher
	token  TokenManager
}

func NewResetPasswordUsecase(
	repo domain.UserRepository,
	hasher PasswordHasher,
	token TokenManager,
) *ResetPasswordUsecase {
	return &ResetPasswordUsecase{
		repo:   repo,
		hasher: hasher,
		token:  token,
	}
}

type CreatePasswordRequest struct {
	ResetPasswordToken string
	Password           string
}

func (uc *ResetPasswordUsecase) Execute(
	ctx context.Context,
	req CreatePasswordRequest,
) error {

	claims, err := uc.token.VerifyToken(req.ResetPasswordToken)
	if err != nil {
		return err
	}

	if claims.TokenType != TokenTypeResetPasswordToken {
		return domain.ErrInvalidToken
	}

	newPassword, err := domain.NewPassword(req.Password)
	if err != nil {
		return err
	}

	hashedPassword, err := uc.hasher.Hash(string(newPassword))
	if err != nil {
		return err
	}

	if err := uc.repo.UpdatePassword(ctx, claims.UserID, hashedPassword); err != nil {
		return err
	}

	return nil
}

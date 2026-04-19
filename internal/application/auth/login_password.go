package authapp

import (
	"context"
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

type LoginUsecase struct {
	repo   domain.UserRepository
	hasher PasswordHasher
	token  TokenManager
}

func NewLoginPasswordUsecase(
	repo domain.UserRepository,
	hasher PasswordHasher,
	token TokenManager,
) *LoginUsecase {
	return &LoginUsecase{
		repo:   repo,
		hasher: hasher,
		token:  token,
	}
}

// Application layer input structure:
type LoginRequest struct {
	Email    string
	Password string
}

type LoginResponse struct {
	User           *domain.User
	OtpAccessToken string
}

func (uc *LoginUsecase) Execute(
	ctx context.Context,
	req LoginRequest,
) (*LoginResponse, error) {
	// 1. value Object
	email, err := domain.NewEmail(req.Email)
	if err != nil {
		return nil, err
	}

	password, err := domain.NewPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 2. Check existing user
	user, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// 3. compare password
	err = uc.hasher.ComparePassword(user.PasswordHash(), string(password))
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Generating Mpin for User
	if user.MPINHash() == "" {
		mpinHash, err := uc.hasher.Hash("8081")
		if err != nil {
			return nil, err
		}

		err = uc.repo.UpdateMPIN(ctx, user.ID(), mpinHash)
		if err != nil {
			return nil, err
		}
		user.MarkHashMpin()
	}

	// Generate OTP Access Token (5 day)
	otpAccessToken, err := uc.token.GenerateToken(
		user.ID(),
		string(user.Mobile()),
		TokenTypeOtpAccessToken,
		120*time.Hour,
	)

	if err != nil {
		return nil, err
	}

	// // 5️⃣ Optional: check verified
	// if !user.IsVerified() {
	// 	return nil, domain.ErrUserNotVerified
	// }

	return &LoginResponse{
		User:           user,
		OtpAccessToken: otpAccessToken,
	}, nil

}

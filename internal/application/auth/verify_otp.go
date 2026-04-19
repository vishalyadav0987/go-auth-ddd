package authapp

import (
	"context"
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

type VerifyOtpUsecase struct {
	otp    OTPService
	token  TokenManager
	repo   domain.UserRepository
	hasher PasswordHasher
}

func NewVerifyOtpUsecase(
	otp OTPService,
	token TokenManager,
	repo domain.UserRepository,
	hasher PasswordHasher,
) *VerifyOtpUsecase {
	return &VerifyOtpUsecase{
		otp:    otp,
		token:  token,
		repo:   repo,
		hasher: hasher,
	}
}

type VerifyOtpRequest struct {
	OTPVerificationToken string
	OTP                  string
}

type VerifyOtpResponse struct {
	Token string
	User  *domain.User
}

func (uc *VerifyOtpUsecase) Execute(
	ctx context.Context,
	req VerifyOtpRequest,
) (*VerifyOtpResponse, error) {
	claims, err := uc.token.VerifyToken(req.OTPVerificationToken)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != TokenTypeOtpVerificationToken &&
		claims.TokenType != TokenTypeResetOtpVerificationToken {
		return nil, domain.ErrInvalidToken
	}

	if err := uc.otp.VerifyOtp(ctx, claims.UserID, req.OTP); err != nil {
		return nil, err
	}

	user, err := uc.repo.FindById(ctx, claims.UserID)

	if err != nil {
		return nil, err
	}

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

	switch claims.TokenType {
	case TokenTypeOtpVerificationToken:
		otpAccessToken, err := uc.token.GenerateToken(
			claims.UserID,
			claims.Phone,
			TokenTypeAccessToken,
			120*time.Hour,
		)
		if err != nil {
			return nil, err
		}

		return &VerifyOtpResponse{
			Token: otpAccessToken,
			User:  user,
		}, nil

	case TokenTypeResetOtpVerificationToken:
		resetPasswordToken, err := uc.token.GenerateToken(
			claims.UserID,
			claims.Phone,
			TokenTypeResetPasswordToken,
			5*time.Minute,
		)
		if err != nil {
			return nil, err
		}

		return &VerifyOtpResponse{
			Token: resetPasswordToken,
			User:  user,
		}, nil
	}

	return nil, domain.ErrInvalidToken

}

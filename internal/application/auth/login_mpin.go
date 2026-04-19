package authapp

import (
	"context"
	"fmt"
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
)

type LoginMpinUsecase struct {
	repo            domain.UserRepository
	hasher          PasswordHasher
	token           TokenManager
	mpinRateLimiter MPINRateLimiter
}

func NewLoginMPINUsecase(
	repo domain.UserRepository,
	hasher PasswordHasher,
	token TokenManager,
	mpinRateLimiter MPINRateLimiter,
) *LoginMpinUsecase {
	return &LoginMpinUsecase{
		repo:            repo,
		hasher:          hasher,
		token:           token,
		mpinRateLimiter: mpinRateLimiter,
	}
}

type LoginMPINRequest struct {
	OTPAccessToken string
	MPIN           string
}

type LoginMPINResponse struct {
	AccessToken string
	User        *domain.User
}

func (uc *LoginMpinUsecase) Execute(
	ctx context.Context,
	req LoginMPINRequest,
) (*LoginMPINResponse, error) {
	// 1️⃣ Extract & verify OTP Access Token

	claims, err := uc.token.VerifyToken(req.OTPAccessToken)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// 2️⃣ Check token_type == otp_access
	if claims.TokenType != TokenTypeOtpAccessToken {
		return nil, domain.ErrInvalidCredentials
	}

	// 3️⃣ Extract Phone from token
	userWithPhone, err := uc.repo.FindByMobile(ctx, domain.Mobile(claims.Phone))
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// 3️⃣ Extract user_id from token to compare with phone
	userWithId, err := uc.repo.FindById(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// 4️⃣ Load user from DB
	if userWithId.ClientID() != userWithPhone.ClientID() {
		return nil, domain.ErrInvalidCredentials
	}

	fmt.Println("Load user and compare user done.")

	// 1️⃣ Rate limit check FIRST
	if err := uc.mpinRateLimiter.Allow(ctx, userWithPhone.ID(), req.OTPAccessToken); err != nil {
		return nil, err // too many attempts
	}

	// 5️⃣ Validate MPIN format
	mpin, err := domain.NewMPIN(req.MPIN)
	if err != nil {
		return nil, err
	}

	if userWithPhone.MPINHash() == "" {
		fmt.Println("Rate limit check FIRST")
		return nil, domain.ErrInvalidCredentials
	}

	// 6️⃣ Compare hashed MPIN
	if err := uc.hasher.ComparePassword(userWithPhone.MPINHash(), string(mpin)); err != nil {
		// ❌ wrong MPIN → increase fail count
		_ = uc.mpinRateLimiter.RegisterFailure(ctx, userWithPhone.ID(), req.OTPAccessToken)
		return nil, domain.ErrInvalidCredentials
	}

	// 3️⃣ Success → reset failures
	_ = uc.mpinRateLimiter.Reset(ctx, userWithPhone.ID(), req.OTPAccessToken)

	// 8️⃣ Generate final Access Token
	accessToken, err := uc.token.GenerateToken(
		userWithId.ID(),
		string(userWithId.Mobile()),
		TokenTypeAccessToken,
		15*time.Minute,
	)

	if err != nil {
		return nil, err
	}

	// 9️⃣ Return Access Token
	return &LoginMPINResponse{
		AccessToken: accessToken,
		User:        userWithPhone,
	}, nil

}

package otp

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/vishalyadav0987/authentication/internal/domain/auth"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/hash"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/persistence/sqlite"
)

type OtpService struct {
	repo   *sqlite.OtpRepository
	hasher hash.BcryptHasher
}

func NewOTPService(repo *sqlite.OtpRepository, hasher hash.BcryptHasher) *OtpService {
	return &OtpService{repo: repo, hasher: hasher}
}

func generateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n)
}

func (s *OtpService) SendOtp(ctx context.Context, userID string) (string, error) {
	otp := generateOTP()

	hash, err := s.hasher.Hash(otp)
	if err != nil {
		return "", err
	}

	err = s.repo.Save(ctx, userID, string(hash), time.Now().Add(5*time.Minute))
	if err != nil {
		return "", err
	}

	return otp, nil
}

func (s *OtpService) VerifyOtp(ctx context.Context, userID, code string) error {

	hash, expiry, err := s.repo.Find(ctx, userID)
	if err != nil {
		return err
	}

	if time.Now().After(expiry) {
		_ = s.repo.Delete(ctx, userID) // cleanup expired OTP
		return auth.ErrInvalidOTP
	}

	if err := s.hasher.ComparePassword(hash, code); err != nil {
		return auth.ErrInvalidOTP
	}

	return s.repo.Delete(ctx, userID)
}

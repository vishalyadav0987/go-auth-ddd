package authapp

import (
	"context"
	"fmt"
	"time"
)

type RequestOtpUsecase struct {
	otp          OTPService
	token        TokenManager
	emailService EmailService
}

func NewRequestOtpUsecase(
	otp OTPService,
	token TokenManager,
	emailService EmailService,
) *RequestOtpUsecase {
	return &RequestOtpUsecase{
		otp:          otp,
		token:        token,
		emailService: emailService,
	}
}

type RequestOTPRequest struct {
	UserID    string
	Mobile    string
	Email     string
	TokenType TokenType
}

type RequestOTPResponse struct {
	OTPVerificationToken string
}

func (uc *RequestOtpUsecase) Execute(
	ctx context.Context,
	req RequestOTPRequest,
) (*RequestOTPResponse, error) {
	otp, err := uc.otp.SendOtp(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	fmt.Println(otp)

	// if req.Mobile != "" {
	// 	uc.sms.Send(req.Mobile, "Your OTP is: "+otp)
	// }

	if req.Email != "" {
		uc.emailService.Send(req.Email, "Your OTP is: "+otp)
	}

	otpVerificationToken, err := uc.token.GenerateToken(
		req.UserID,
		req.Mobile,
		req.TokenType,
		5*time.Minute,
	)

	if err != nil {
		return nil, err
	}

	return &RequestOTPResponse{
		OTPVerificationToken: otpVerificationToken,
	}, nil
}

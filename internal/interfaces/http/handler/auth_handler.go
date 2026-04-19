package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authapp "github.com/vishalyadav0987/authentication/internal/application/auth"
	domain "github.com/vishalyadav0987/authentication/internal/domain/auth"
	"github.com/vishalyadav0987/authentication/pkg/response"
)

type AuthHandler struct {
	registerUC       *authapp.RegisterUsecase
	loginUc          *authapp.LoginUsecase
	loginWithMpinUc  *authapp.LoginMpinUsecase
	verifyOtpUC      *authapp.VerifyOtpUsecase
	requestOtpUC     *authapp.RequestOtpUsecase
	createPasswordUC *authapp.ResetPasswordUsecase
	repo             domain.UserRepository
}

// Construcutor
func NewAuthHandler(
	registerUC *authapp.RegisterUsecase,
	loginUc *authapp.LoginUsecase,
	loginWithMpinUc *authapp.LoginMpinUsecase,
	verifyOtpUC *authapp.VerifyOtpUsecase,
	requestOtpUC *authapp.RequestOtpUsecase,
	createPasswordUC *authapp.ResetPasswordUsecase,
	repo domain.UserRepository,
) *AuthHandler {
	return &AuthHandler{
		registerUC:       registerUC,
		loginUc:          loginUc,
		loginWithMpinUc:  loginWithMpinUc,
		verifyOtpUC:      verifyOtpUC,
		requestOtpUC:     requestOtpUC,
		createPasswordUC: createPasswordUC,
		repo:             repo,
	}
}

func (h *AuthHandler) RegisterUser(c *gin.Context) {
	var req authapp.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
	}

	user, err := h.registerUC.Execute(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{
		"id":        user.ID(),
		"client_id": user.ClientID(),
	})
}

func (h *AuthHandler) LoginUser(c *gin.Context) {
	var req authapp.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
	}

	result, err := h.loginUc.Execute(c, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	user := result.User

	response.Success(c, http.StatusCreated, gin.H{
		"isActive":       user.IsVerified(),
		"hasMpin":        user.IsHashMpin(),
		"otpAccessToken": result.OtpAccessToken,
		"userData": gin.H{
			"client_id": user.ClientID(),
			"id":        user.ID(),
			"email":     user.Email(),
			"mobile":    user.Mobile(),
		},
		"createdAt": user.CreatedAt(),
	})
}

func (h *AuthHandler) LoginWithMpin(c *gin.Context) {

	// Get otp Access Token from Header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Error(c, http.StatusUnauthorized, "missing token")
		return
	}

	otpAcessTokenString := strings.TrimPrefix(authHeader, "Bearer ")

	var body struct {
		MPIN string `json:"mpin"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.loginWithMpinUc.Execute(c, authapp.LoginMPINRequest{
		OTPAccessToken: otpAcessTokenString,
		MPIN:           body.MPIN,
	})
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	user := result.User

	response.Success(c, http.StatusCreated, gin.H{
		"isActive":    user.IsVerified(),
		"hasMpin":     user.IsHashMpin(),
		"accessToken": result.AccessToken,
		"userData": gin.H{
			"client_id": user.ClientID(),
			"id":        user.ID(),
			"email":     user.Email(),
			"mobile":    user.Mobile(),
		},
		"createdAt": user.CreatedAt(),
	})
}

func (h *AuthHandler) SendOtp(c *gin.Context) {
	var body struct {
		Mobile string `json:"mobile"`
		Email  string `json:"email"`
	}

	// 1️⃣ Bind request
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	var user *domain.User
	var err error

	ctx := c.Request.Context()

	// 2️⃣ Find user
	if body.Mobile != "" {
		user, err = h.repo.FindByMobile(ctx, domain.Mobile(body.Mobile))
	} else if body.Email != "" {
		email, _ := domain.NewEmail(body.Email)
		user, err = h.repo.FindByEmail(ctx, email)
	} else {
		response.Error(c, http.StatusNotFound, "mobile or email required")
		return
	}

	if err != nil {
		response.Error(c, http.StatusNotFound, "user not found")
		return
	}

	// 3️⃣ Call usecase
	resp, err := h.requestOtpUC.Execute(ctx, authapp.RequestOTPRequest{
		UserID:    user.ID(),
		Mobile:    string(user.Mobile()),
		Email:     string(user.Email()),
		TokenType: authapp.TokenTypeOtpVerificationToken,
	})

	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	// 4️⃣ Response
	response.Success(c, http.StatusAccepted, gin.H{
		"otpVerificationToken": resp.OTPVerificationToken,
		"message":              "OTP sent successfully",
	})
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Error(c, 401, "authorization header missing")
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		response.Error(c, 401, "invalid authorization format")
		return
	}

	token := parts[1]

	var body struct {
		OTP string `json:"otp"`
	}

	c.ShouldBindJSON(&body)

	resp, err := h.verifyOtpUC.Execute(c, authapp.VerifyOtpRequest{
		OTPVerificationToken: token,
		OTP:                  body.OTP,
	})

	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, 200, gin.H{
		"otpAccessToken": resp.Token,
		"isActive":       resp.User.IsVerified(),
		"hasMpin":        resp.User.IsHashMpin(),
		"userData": gin.H{
			"client_id": resp.User.ClientID(),
			"id":        resp.User.ID(),
			"email":     resp.User.Email(),
			"mobile":    resp.User.Mobile(),
		},
	})
}

func (h *AuthHandler) SendResetOtp(c *gin.Context) {
	var body struct {
		Mobile string `json:"mobile"`
		Email  string `json:"email"`
	}

	// 1️⃣ Bind request
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	var user *domain.User
	var err error

	ctx := c.Request.Context()

	// 2️⃣ Find user
	if body.Mobile != "" {
		user, err = h.repo.FindByMobile(ctx, domain.Mobile(body.Mobile))
	} else if body.Email != "" {
		email, _ := domain.NewEmail(body.Email)
		user, err = h.repo.FindByEmail(ctx, email)
	} else {
		response.Error(c, http.StatusNotFound, "mobile or email required")
		return
	}

	if err != nil {
		response.Error(c, http.StatusNotFound, "user not found")
		return
	}

	// 3️⃣ Call usecase
	resp, err := h.requestOtpUC.Execute(ctx, authapp.RequestOTPRequest{
		UserID:    user.ID(),
		Mobile:    string(user.Mobile()),
		Email:     string(user.Email()),
		TokenType: authapp.TokenTypeResetOtpVerificationToken,
	})

	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	// 4️⃣ Response
	response.Success(c, http.StatusAccepted, gin.H{
		"otpVerificationToken": resp.OTPVerificationToken,
		"message":              "OTP sent successfully",
	})
}

func (h *AuthHandler) VerifyResetOtp(c *gin.Context) {

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Error(c, 401, "authorization header missing")
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		response.Error(c, 401, "invalid authorization format")
		return
	}

	token := parts[1]

	var body struct {
		OTP string `json:"otp"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, 400, "invalid request body")
		return
	}

	if body.OTP == "" {
		response.Error(c, 400, "otp is required")
		return
	}

	resp, err := h.verifyOtpUC.Execute(c, authapp.VerifyOtpRequest{
		OTPVerificationToken: token,
		OTP:                  body.OTP,
	})

	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, 200, gin.H{
		"resetPasswordToken": resp.Token,
	})
}

func (h *AuthHandler) CreateNewPassword(c *gin.Context) {

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Error(c, 401, "authorization header missing")
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		response.Error(c, 401, "invalid authorization format")
		return
	}

	token := parts[1]

	var body struct {
		Password string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, 400, "invalid request body")
		return
	}

	if body.Password == "" {
		response.Error(c, 400, "new password is required")
		return
	}

	if len(body.Password) < 6 {
		response.Error(c, 400, "password must be at least 6 characters")
		return
	}

	err := h.createPasswordUC.Execute(c, authapp.CreatePasswordRequest{
		ResetPasswordToken: token,
		Password:           body.Password,
	})

	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, 200, gin.H{
		"message": "password reset successfully",
	})
}

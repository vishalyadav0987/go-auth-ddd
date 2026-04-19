package http

import (
	"github.com/gin-gonic/gin"
	"github.com/vishalyadav0987/authentication/internal/interfaces/http/handler"
)

func SetUpRouter(authHandler *handler.AuthHandler) *gin.Engine {
	router := gin.Default()

	auth := router.Group("api/v1/auth")
	{
		auth.POST("/register", authHandler.RegisterUser)
		auth.POST("/login-with-password", authHandler.LoginUser)
		auth.POST("/login-with-mpin", authHandler.LoginWithMpin)
		auth.POST("/login-with-otp", authHandler.SendOtp)
		auth.POST("/verify-otp", authHandler.VerifyOTP)
		auth.POST("/password/reset/request", authHandler.SendResetOtp)
		auth.POST("/password/reset/verify-otp", authHandler.VerifyResetOtp)
		auth.POST("/password/reset/confirm", authHandler.CreateNewPassword)
	}

	return router
}

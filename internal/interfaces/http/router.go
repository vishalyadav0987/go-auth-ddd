package http

import (
	"github.com/gin-gonic/gin"
	"github.com/vishalyadav0987/authentication/internal/interfaces/http/handler"
)

func SetUpRouter(authHandler *handler.AuthHandler, orderHandler *handler.OrderHandler, paymentHandler *handler.PaymentHandler, webhookHandler *handler.PaymentWebhookHandler, deadLetterHandler *handler.DeadLetterHandler, deadLetterDetailHandler *handler.DeadLetterDetailHandler,
	retryDeadLetterHandler *handler.RetryDeadLetterHandler,
	userNotificationHandler *handler.UserNotificationHandler,
	sendNotificationHandler *handler.SendNotificationHandler,
	requireAccessToken gin.HandlerFunc,
	requireAdmin gin.HandlerFunc,
) *gin.Engine {
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

	order := router.Group("api/v1/order")
	{
		order.POST("/create", orderHandler.CreateOrder)
	}

	payment := router.Group("api/v1/payment")
	{
		payment.POST("/create", paymentHandler.CreatePayment)
	}

	// Dead-letter queue administration: authenticated admins only.
	notification := router.Group(
		"api/v1/notification",
		requireAccessToken,
		requireAdmin,
	)
	{
		notification.GET("/dead-letter-deliveries", deadLetterHandler.GetDeadLetters)
		notification.GET("/dead-letter-deliveries/:deliveryID", deadLetterDetailHandler.GetDeadLetterDetail)
		notification.POST("/dead-letters/:deliveryID/retry", retryDeadLetterHandler.RetryDeadLetterDelivery)
		notification.POST("/send", sendNotificationHandler.SendNotification)
	}

	// The authenticated user's own notifications, preferences and devices.
	notifications := router.Group("api/v1/notifications", requireAccessToken)
	{
		notifications.GET("", userNotificationHandler.ListNotifications)
		notifications.PATCH("/read-all", userNotificationHandler.MarkAllAsRead)

		notifications.GET("/preferences", userNotificationHandler.GetPreferences)
		notifications.PUT("/preferences", userNotificationHandler.UpdatePreferences)

		notifications.POST("/devices", userNotificationHandler.RegisterDevice)
		notifications.DELETE("/devices/:deviceID", userNotificationHandler.RemoveDevice)

		notifications.GET("/:notificationID", userNotificationHandler.GetNotification)
		notifications.PATCH("/:notificationID/read", userNotificationHandler.MarkAsRead)
	}

	router.POST(
		"/api/v1/payments/webhook",
		webhookHandler.Handle,
	)

	return router
}

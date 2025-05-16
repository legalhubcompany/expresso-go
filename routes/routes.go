package routes

import (
	"shollu/controllers"
	"shollu/middlewares"

	"github.com/gofiber/fiber/v2"
)

func ApiKeyMiddleware(c *fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")          // Ambil API Key dari header
	validApiKey := "shollusemakindidepan" // Ganti dengan API Key yang aman

	if apiKey != validApiKey {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: Invalid API Key"})
	}
	return c.Next()
}

func SetupRoutes(app *fiber.App) {

	auth := app.Group("/api/public/v1")
	auth.Post("/login", controllers.Login)
	auth.Post("/register", controllers.Register)

	// new Login Via Whatsapp
	new_auth := app.Group("/api/auth/v2")
	new_auth.Post("/login/request", controllers.WhatsAppLoginRequest)
	new_auth.Post("/login/whatsapp-bot", controllers.WhatsAppBotCallback)
	new_auth.Get("/login/gateway", controllers.WhatsAppGateway)
	new_auth.Post("/login/validate", controllers.ValidateWhatsAppLoginToken)

	apiV1 := app.Group("/api/v1", middlewares.JWTMiddleware())
	apiV1.Get("/user/profile", controllers.GetProfile)
	apiV1.Put("/user/profile", controllers.UpdateProfile)
	apiV1.Post("/user/profile/upload-photo", controllers.UploadProfilePicture)
}

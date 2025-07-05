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

	gateway := app.Group("/gateway")
	gateway.Get("/whatsapp-login", controllers.WhatsAppGateway)
	gateway.Get("/webhook", controllers.WebhookVerify)
	gateway.Post("/webhook", controllers.WebhookReceiver)

	// new Login Via Whatsapp
	new_auth := app.Group("/api/auth/v2")
	new_auth.Post("/login/request", controllers.WhatsAppLoginRequest)
	new_auth.Post("/login/whatsapp-bot", controllers.WhatsAppBotCallback)
	new_auth.Post("/login/validate", controllers.ValidateWhatsAppLoginToken)

	// midtrans webhook
	vendors := app.Group("/vendors/v1")
	vendors.Post("/midtrans-webhook", controllers.MidtransWebhookCallback)

	privateAPI := app.Group("/api/v1", middlewares.JWTMiddleware())
	privateAPI.Put("/user/profile", controllers.UpdateProfile)
	privateAPI.Post("/create-payment", controllers.CreatePayment)
	privateAPI.Post("/create-transaksi", controllers.CreateTransaksi)
	privateAPI.Get("/master-data/pekerjaan", controllers.GetMasterPekerjaanList)
}

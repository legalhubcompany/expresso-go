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
	api := app.Group("/api")
	api.Post("/register-itikaf", controllers.RegisterPesertaItikaf)
	api.Get("/register-masjid/:id_event", controllers.GetMasjidList)
	api.Get("/rekap-absen/:id_masjid", controllers.GetRekapAbsen)
	api.Get("/get-masjid/:id_masjid", controllers.GetMasjidByID)
	api.Get("/statistics-event", controllers.GetEventStatistics)
	api.Get("/dashboard", controllers.GetNewRegistrantStatistics)
	api.Get("/statistics-event-all", controllers.GetAttendanceStatistics)

	auth := app.Group("/api/public/v1")
	auth.Post("/login", controllers.Login)
	auth.Post("/register", controllers.Register)

	// apiV1.Post("/absent-qr", ApiKeyMiddleware, controllers.SaveAbsenQR)
	// apiV1.Post("/collections-create", controllers.CreateCollection)
	// apiV1.Get("/collections-get-absensi/:slug", controllers.ViewCollection)
	// apiV1.Get("/collections-get", controllers.GetCollectionsMeta)
	// apiV1.Get("/collections-get-meta/:slug", controllers.GetCollectionsMetaDetail)

	apiV1 := app.Group("/api/v1", middlewares.JWTMiddleware())
	apiV1.Get("/user/profile", controllers.GetProfile)
	apiV1.Put("/user/profile", controllers.UpdateProfile)
	apiV1.Post("/user/profile/upload-photo", controllers.UploadProfilePicture)
}

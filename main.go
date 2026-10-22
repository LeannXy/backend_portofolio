package main

import (
	"log"
	"os"
	"strings"

	"portfolio-api/config"
	"portfolio-api/database"
	"portfolio-api/database/seed" // Import folder seeders
	"portfolio-api/middleware"
	"portfolio-api/routes"
	"portfolio-api/storage"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main() {
	// 1. Muat konfigurasi
	config.LoadConfig()
	if os.Getenv("ADMIN_USERNAME") == "" || os.Getenv("ADMIN_PASSWORD") == "" {
		log.Fatal("ADMIN_USERNAME dan ADMIN_PASSWORD wajib diatur di environment.")
	}
	if len(os.Getenv("SESSION_SECRET")) < 32 {
		log.Fatal("SESSION_SECRET wajib berisi minimal 32 karakter acak.")
	}
	if !middleware.ValidCookieSameSite(os.Getenv("COOKIE_SAME_SITE")) {
		log.Fatal("COOKIE_SAME_SITE harus Strict, Lax, atau None.")
	}
	if strings.EqualFold(os.Getenv("COOKIE_SAME_SITE"), "none") &&
		!strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true") {
		log.Fatal("COOKIE_SECURE=true wajib digunakan saat COOKIE_SAME_SITE=None.")
	}
	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		if strings.TrimSpace(os.Getenv("FRONTEND_ORIGINS")) == "" {
			log.Fatal("FRONTEND_ORIGINS wajib diatur di production.")
		}
		if err := middleware.ValidateAllowedOrigins(middleware.AllowedOrigins()); err != nil {
			log.Fatal("FRONTEND_ORIGINS tidak valid: ", err)
		}
		if !strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true") {
			log.Fatal("COOKIE_SECURE=true wajib digunakan di production.")
		}
		if strings.EqualFold(os.Getenv("DB_SSLMODE"), "disable") || os.Getenv("DB_SSLMODE") == "" {
			log.Fatal("DB_SSLMODE harus mengaktifkan TLS di production.")
		}
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if len(adminPassword) < 16 || strings.Contains(strings.ToLower(adminPassword), "replace-with") {
			log.Fatal("ADMIN_PASSWORD production harus unik dan berisi minimal 16 karakter.")
		}
		if strings.Contains(strings.ToLower(os.Getenv("SESSION_SECRET")), "replace-with") {
			log.Fatal("SESSION_SECRET contoh tidak boleh digunakan di production.")
		}
		cloudinary, err := storage.NewCloudinaryFromEnv()
		if err != nil {
			log.Fatal("Konfigurasi Cloudinary tidak valid: ", err)
		}
		if cloudinary == nil {
			log.Fatal("Cloudinary wajib dikonfigurasi di production agar upload bertahan setelah restart.")
		}
	}

	// 2. Hubungkan Database
	database.ConnectDB()

	// 3. Jalankan Seeder
	seeders.SeedProjects()

	// 4. Inisialisasi Fiber
	app := fiber.New(fiber.Config{BodyLimit: 13 * 1024 * 1024})

	// 5. Pasang Middleware (CORS & Logger)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     middleware.AllowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))
	app.Use(middleware.RequestLogger())

	// 6. Tes Rute
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Backend Portofolio API Berjalan (Clean Architecture)!")
	})

	// 7. Muat Rute Utama
	routes.SetupRoutes(app)

	// 8. Jalankan Server
	port := strings.TrimSpace(config.GetEnv("PORT", "3000"))
	log.Printf("Server berjalan di port %s", port)
	log.Fatal(app.Listen(":" + port))
}

package routes

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"portfolio-api/controllers"
	"portfolio-api/middleware"
	"portfolio-api/repositories"
	"portfolio-api/services"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/google/uuid"
)

func SetupRoutes(app *fiber.App, emailSender services.EmailSender) {
	api := app.Group("/api")

	// Dependency Injection (Merakit komponen)
	projectRepo := repositories.NewProjectRepository()
	projectService := services.NewProjectService(projectRepo)
	projectController := controllers.NewProjectController(projectService)
	adminController := controllers.NewAdminController(projectService)
	contactController := controllers.NewContactController(emailSender)

	loginLimiter := limiter.New(limiter.Config{
		Max:                    10,
		Expiration:             15 * time.Minute,
		SkipSuccessfulRequests: true,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Terlalu banyak percobaan login. Coba lagi dalam 15 menit.",
			})
		},
	})
	contactLimiter := limiter.New(limiter.Config{
		Max:        3,
		Expiration: 15 * time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Terlalu banyak pesan dikirim. Coba lagi beberapa menit.",
			})
		},
	})

	api.Get("/projects", projectController.GetProjects)
	api.Get("/content", controllers.GetSiteContent)
	api.Get("/auth/session", adminController.Session)
	api.Post("/contact", contactLimiter, middleware.RequireAllowedOrigin, contactController.Submit)
	api.Post("/auth/login", loginLimiter, middleware.RequireAllowedOrigin, adminController.Login)
	api.Post("/auth/logout", middleware.AdminAuth, middleware.RequireAllowedOrigin, adminController.Logout)

	admin := api.Group("/admin", middleware.AdminAuth, middleware.RequireAllowedOrigin)
	admin.Post("/projects", adminController.CreateProject)
	admin.Put("/projects/:id", adminController.UpdateProject)
	admin.Delete("/projects/:id", adminController.DeleteProject)
	admin.Post("/uploads", adminController.Upload)
	admin.Put("/content/:key", controllers.SaveSiteContent)

	uploadDirectory := os.Getenv("UPLOAD_DIR")
	if uploadDirectory == "" {
		uploadDirectory = "public/uploads"
	}
	app.Get("/uploads/:filename", func(c fiber.Ctx) error {
		filename := c.Params("filename")
		if filepath.Base(filename) != filename {
			return c.SendStatus(fiber.StatusNotFound)
		}
		extension := filepath.Ext(filename)
		if extension != ".jpg" && extension != ".webp" {
			return c.SendStatus(fiber.StatusNotFound)
		}
		if _, err := uuid.Parse(strings.TrimSuffix(filename, extension)); err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.SendFile(filepath.Join(uploadDirectory, filename))
	})
}

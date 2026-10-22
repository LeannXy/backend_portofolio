package controllers

import (
	"portfolio-api/services"

	"github.com/gofiber/fiber/v3"
)

type ProjectController struct {
	Service *services.ProjectService
}

func NewProjectController(service *services.ProjectService) *ProjectController {
	return &ProjectController{Service: service}
}

// GetProjects menangani request GET /api/projects
func (ctrl *ProjectController) GetProjects(c fiber.Ctx) error {
	projects, err := ctrl.Service.GetAllProjects()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data proyek",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   projects,
	})
}

package services

import (
	"portfolio-api/models"
	"portfolio-api/repositories"
)

type ProjectService struct {
	Repo *repositories.ProjectRepository
}

func NewProjectService(repo *repositories.ProjectRepository) *ProjectService {
	return &ProjectService{Repo: repo}
}

// GetAllProjects mengambil daftar proyek melalui repository
func (s *ProjectService) GetAllProjects() ([]models.Project, error) {
	// Di sini Anda bisa menambahkan logika tambahan,
	// misalnya memfilter proyek yang tidak aktif, dll.
	return s.Repo.FindAll()
}

func (s *ProjectService) CreateProject(project *models.Project) error {
	return s.Repo.Create(project)
}

func (s *ProjectService) UpdateProject(id uint, values map[string]any) (*models.Project, error) {
	return s.Repo.Update(id, values)
}

func (s *ProjectService) DeleteProject(id uint) (*models.Project, error) {
	return s.Repo.Delete(id)
}

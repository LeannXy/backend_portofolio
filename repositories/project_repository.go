package repositories

import (
	"portfolio-api/database"
	"portfolio-api/models"

	"gorm.io/gorm"
)

// ProjectRepository mengelola interaksi database untuk Project
type ProjectRepository struct{}

func NewProjectRepository() *ProjectRepository {
	return &ProjectRepository{}
}

// FindAll mengambil semua data proyek dari database
func (r *ProjectRepository) FindAll() ([]models.Project, error) {
	var projects []models.Project
	result := database.DB.Order("sort_order ASC").Order("id ASC").Find(&projects)
	return projects, result.Error
}

func (r *ProjectRepository) Create(project *models.Project) error {
	var sortOrder int
	if err := database.DB.Model(&models.Project{}).Select("COALESCE(MAX(sort_order), -1) + 1").Scan(&sortOrder).Error; err != nil {
		return err
	}
	project.SortOrder = sortOrder
	return database.DB.Create(project).Error
}

func (r *ProjectRepository) Update(id uint, values map[string]any) (*models.Project, error) {
	var project models.Project
	if err := database.DB.First(&project, id).Error; err != nil {
		return nil, err
	}
	if err := database.DB.Model(&project).Updates(values).Error; err != nil {
		return nil, err
	}
	return &project, database.DB.First(&project, id).Error
}

func (r *ProjectRepository) Delete(id uint) (*models.Project, error) {
	var project models.Project
	if err := database.DB.First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, database.DB.Delete(&project).Error
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

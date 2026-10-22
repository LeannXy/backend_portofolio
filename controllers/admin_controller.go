package controllers

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"image"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"portfolio-api/database"
	"portfolio-api/middleware"
	"portfolio-api/models"
	"portfolio-api/repositories"
	"portfolio-api/services"
	"portfolio-api/storage"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
)

type AdminController struct {
	Service *services.ProjectService
}

func NewAdminController(service *services.ProjectService) *AdminController {
	return &AdminController{Service: service}
}

func (ctrl *AdminController) Session(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"authenticated": middleware.IsAdminToken(c.Cookies("portfolio_admin_session"))})
}

func (ctrl *AdminController) Login(c fiber.Ctx) error {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&credentials); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username dan password wajib diisi."})
	}

	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")
	if !constantTimeEqual(credentials.Username, username) || !constantTimeEqual(credentials.Password, password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Username atau password salah."})
	}

	expiresAt := time.Now().Add(8 * time.Hour)
	c.Cookie(middleware.AdminCookie(middleware.NewAdminToken(), expiresAt))
	return c.JSON(fiber.Map{"authenticated": true})
}

func (ctrl *AdminController) Logout(c fiber.Ctx) error {
	c.Cookie(middleware.AdminCookie("", time.Unix(1, 0)))
	return c.SendStatus(fiber.StatusNoContent)
}

type projectInput struct {
	Title       string `json:"title"`
	Category    string `json:"category"`
	TechStack   string `json:"tech_stack"`
	Year        string `json:"year"`
	Description string `json:"description"`
	Image       string `json:"image"`
	ImageURL    string `json:"image_url"`
	Link        string `json:"link"`
	TextSide    string `json:"textSide"`
	TextSideAlt string `json:"text_side"`
}

func (input projectInput) model() (models.Project, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Category = strings.TrimSpace(input.Category)
	if input.Category == "" {
		input.Category = strings.TrimSpace(input.TechStack)
	}
	input.Year = strings.TrimSpace(input.Year)
	input.Description = strings.TrimSpace(input.Description)
	input.Image = strings.TrimSpace(input.Image)
	if input.Image == "" {
		input.Image = strings.TrimSpace(input.ImageURL)
	}
	input.Link = strings.TrimSpace(input.Link)
	if input.TextSide == "" {
		input.TextSide = input.TextSideAlt
	}
	if input.TextSide == "" {
		input.TextSide = "left"
	}

	if input.Title == "" || input.Category == "" || input.Year == "" || input.Description == "" || input.Image == "" {
		return models.Project{}, fmt.Errorf("Judul, kategori, tahun, deskripsi, dan gambar wajib diisi.")
	}
	if len(input.Title) > 160 || len(input.Category) > 100 || len(input.Year) > 12 || len(input.Description) > 4000 {
		return models.Project{}, fmt.Errorf("Beberapa field melebihi panjang maksimal.")
	}
	if input.TextSide != "left" && input.TextSide != "right" {
		return models.Project{}, fmt.Errorf("Nilai posisi panel tidak valid.")
	}
	if !validWebURL(input.Link, true) || !validWebURL(input.Image, false) {
		return models.Project{}, fmt.Errorf("URL proyek atau gambar harus menggunakan HTTP atau HTTPS.")
	}

	return models.Project{
		Title:       input.Title,
		Category:    input.Category,
		TechStack:   input.Category,
		Year:        input.Year,
		Description: input.Description,
		ImageURL:    input.Image,
		Link:        input.Link,
		TextSide:    input.TextSide,
	}, nil
}

func (ctrl *AdminController) CreateProject(c fiber.Ctx) error {
	var input projectInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data proyek tidak valid."})
	}
	project, err := input.model()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := ctrl.Service.CreateProject(&project); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan proyek."})
	}
	return c.Status(fiber.StatusCreated).JSON(project)
}

func (ctrl *AdminController) UpdateProject(c fiber.Ctx) error {
	id, ok := projectID(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID proyek tidak valid."})
	}
	var input projectInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data proyek tidak valid."})
	}
	project, err := input.model()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	updated, err := ctrl.Service.UpdateProject(id, map[string]any{
		"title": project.Title, "category": project.Category, "tech_stack": project.TechStack,
		"year": project.Year, "description": project.Description, "image_url": project.ImageURL,
		"link": project.Link, "text_side": project.TextSide,
	})
	if repositories.IsNotFound(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Proyek tidak ditemukan."})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui proyek."})
	}
	return c.JSON(updated)
}

func (ctrl *AdminController) DeleteProject(c fiber.Ctx) error {
	id, ok := projectID(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID proyek tidak valid."})
	}
	project, err := ctrl.Service.DeleteProject(id)
	if repositories.IsNotFound(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Proyek tidak ditemukan."})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus proyek."})
	}
	if strings.HasPrefix(project.ImageURL, "/uploads/") {
		filename := filepath.Base(project.ImageURL)
		stillUsed := int64(0)
		if err := database.DB.Model(&models.Project{}).Where("image_url = ?", project.ImageURL).Count(&stillUsed).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Proyek terhapus, namun gambar gagal diperiksa untuk pembersihan."})
		}
		if stillUsed == 0 {
			if err := os.Remove(filepath.Join(uploadDirectory(), filename)); err != nil && !os.IsNotExist(err) {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Proyek terhapus, namun file gambar gagal dibersihkan."})
			}
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (ctrl *AdminController) Upload(c fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pilih gambar untuk diunggah."})
	}
	if file.Size > 12*1024*1024 {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "Ukuran gambar maksimal 12 MB."})
	}
	purpose := c.FormValue("purpose")
	if purpose == "" {
		purpose = "project"
	}
	if purpose != "project" && purpose != "profile" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Jenis gambar tidak valid."})
	}
	source, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File gambar tidak dapat dibaca."})
	}
	defer source.Close()

	config, _, err := image.DecodeConfig(source)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File WebP tidak valid."})
	}
	width, height := config.Width, config.Height
	if width < 16 || height < 16 || width > 12000 || height > 12000 || int64(width)*int64(height) > 24000000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Dimensi gambar terlalu besar atau terlalu kecil."})
	}
	if purpose == "project" && absFloat(float64(width)/float64(height)-16.0/9.0) > 0.01 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Gambar proyek harus menggunakan rasio 16:9."})
	}
	if _, err := source.Seek(0, 0); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File gambar tidak dapat dibaca."})
	}
	_, format, err := image.Decode(source)
	if err != nil || format != "webp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Gambar harus berupa file WebP yang valid."})
	}

	if _, err := source.Seek(0, 0); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File gambar tidak dapat dibaca."})
	}
	cloudinary, err := storage.NewCloudinaryFromEnv()
	if err != nil {
		log.Printf("cloud image storage configuration error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Penyimpanan gambar belum dikonfigurasi dengan benar."})
	}
	if cloudinary != nil {
		imageURL, err := cloudinary.UploadWebP(c.Context(), uuid.NewString(), source)
		if err != nil {
			log.Printf("cloud image upload failed: %v", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Gambar gagal disimpan ke penyimpanan cloud."})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"image": imageURL, "width": width, "height": height,
		})
	}

	if err := os.MkdirAll(uploadDirectory(), 0o755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Folder upload tidak dapat dibuat."})
	}
	filename := uuid.NewString() + ".webp"
	destination, err := os.Create(filepath.Join(uploadDirectory(), filename))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gambar gagal disimpan."})
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(filepath.Join(uploadDirectory(), filename))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gambar WebP gagal disimpan."})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"image": "/uploads/" + filename, "width": width, "height": height,
	})
}

func projectID(c fiber.Ctx) (uint, bool) {
	value, err := strconv.ParseUint(c.Params("id"), 10, 32)
	return uint(value), err == nil && value > 0
}

func constantTimeEqual(left, right string) bool {
	leftHash := sha256.Sum256([]byte(left))
	rightHash := sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1
}

func validWebURL(value string, optional bool) bool {
	if value == "" {
		return optional
	}
	if strings.HasPrefix(value, "/uploads/") {
		return validUploadedImagePath(value)
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

func validUploadedImagePath(value string) bool {
	filename := strings.TrimPrefix(value, "/uploads/")
	if filename == value || filepath.Base(filename) != filename {
		return false
	}
	extension := filepath.Ext(filename)
	if extension != ".jpg" && extension != ".webp" {
		return false
	}
	_, err := uuid.Parse(strings.TrimSuffix(filename, extension))
	return err == nil
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func uploadDirectory() string {
	if directory := os.Getenv("UPLOAD_DIR"); directory != "" {
		return directory
	}
	return filepath.Join("public", "uploads")
}

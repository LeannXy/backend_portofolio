package controllers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"portfolio-api/database"

	"github.com/gofiber/fiber/v3"
)

var contentKeys = map[string]bool{
	"history":      true,
	"technologies": true,
	"socials":      true,
	"profileImage": true,
}

func GetSiteContent(c fiber.Ctx) error {
	content := make(map[string]json.RawMessage, len(contentKeys))
	rows, err := database.DB.Raw("SELECT key, value FROM site_content").Rows()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat konten situs."})
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membaca konten situs."})
		}
		content[key] = value
	}
	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membaca konten situs."})
	}
	return c.JSON(content)
}

func SaveSiteContent(c fiber.Ctx) error {
	key := c.Params("key")
	if !contentKeys[key] {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Jenis konten tidak ditemukan."})
	}
	var payload struct {
		Value json.RawMessage `json:"value"`
	}
	if err := c.Bind().Body(&payload); err != nil || len(payload.Value) == 0 || len(payload.Value) > 1<<20 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Data konten tidak valid atau melebihi 1 MB."})
	}
	if err := validateContent(key, payload.Value); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := database.DB.Exec(
		`INSERT INTO site_content (key, value, updated_at) VALUES (?, ?::jsonb, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		key, string(payload.Value),
	).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan konten situs."})
	}
	return c.JSON(fiber.Map{"key": key, "value": json.RawMessage(payload.Value)})
}

func validateContent(key string, raw json.RawMessage) error {
	switch key {
	case "history":
		var entries []struct {
			ID        int      `json:"id"`
			Year      string   `json:"year"`
			Title     string   `json:"title"`
			Text      string   `json:"text"`
			Details   []string `json:"details"`
			TechStack []string `json:"techStack"`
			Ratio     float64  `json:"ratio"`
		}
		if err := json.Unmarshal(raw, &entries); err != nil || len(entries) == 0 {
			return fmt.Errorf("Histori harus berupa daftar yang memiliki minimal satu item.")
		}
		seenIDs := make(map[int]bool, len(entries))
		for _, entry := range entries {
			if entry.ID < 0 || entry.Year == "" || entry.Title == "" || entry.Text == "" || entry.Details == nil || entry.TechStack == nil {
				return fmt.Errorf("Setiap histori perlu ID, tahun, judul, deskripsi, detail, dan teknologi.")
			}
			if entry.Ratio < 0 || entry.Ratio > 1 {
				return fmt.Errorf("Rasio posisi histori harus berada di antara 0 dan 1.")
			}
			if seenIDs[entry.ID] {
				return fmt.Errorf("Setiap milestone harus memiliki ID yang unik.")
			}
			seenIDs[entry.ID] = true
		}
	case "technologies":
		var entries []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Tag      string `json:"tag"`
			Role     string `json:"role"`
			Desc     string `json:"desc"`
		}
		if err := json.Unmarshal(raw, &entries); err != nil || len(entries) == 0 {
			return fmt.Errorf("Teknologi harus berupa daftar yang memiliki minimal satu item.")
		}
		for _, entry := range entries {
			if entry.Name == "" || entry.Category == "" || entry.Tag == "" || entry.Role == "" || entry.Desc == "" {
				return fmt.Errorf("Setiap teknologi perlu nama, kategori, tag, peran, dan deskripsi.")
			}
			if entry.Category != "backend" && entry.Category != "database" && entry.Category != "frontend" && entry.Category != "design" {
				return fmt.Errorf("Kategori teknologi harus backend, database, frontend, atau design.")
			}
		}
	case "socials":
		var entries map[string]struct {
			Label      string `json:"label"`
			HoverLabel string `json:"hoverLabel"`
			URL        string `json:"url"`
		}
		if err := json.Unmarshal(raw, &entries); err != nil {
			return fmt.Errorf("Media sosial harus berupa objek JSON yang valid.")
		}
		for _, platform := range []string{"github", "linkedin", "tiktok", "instagram"} {
			entry, ok := entries[platform]
			if !ok || entry.Label == "" || entry.HoverLabel == "" {
				return fmt.Errorf("Lengkapi label, label hover, dan URL untuk %s.", platform)
			}
			if entry.URL != "" {
				parsed, err := url.ParseRequestURI(entry.URL)
				if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || strings.ContainsAny(entry.URL, "\r\n") {
					return fmt.Errorf("URL %s harus berupa alamat HTTP atau HTTPS yang valid.", platform)
				}
			}
		}
	case "profileImage":
		var imagePath string
		if err := json.Unmarshal(raw, &imagePath); err != nil {
			return fmt.Errorf("Foto profil harus berupa path gambar yang valid.")
		}
		if imagePath != "/profile.webp" && !validUploadedProfileImage(imagePath) {
			return fmt.Errorf("Foto profil harus menggunakan gambar WebP yang diunggah melalui dashboard.")
		}
	}
	return nil
}

func validUploadedProfileImage(imagePath string) bool {
	if strings.HasSuffix(imagePath, ".webp") && validUploadedImagePath(imagePath) {
		return true
	}

	parsed, err := url.Parse(imagePath)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "res.cloudinary.com" ||
		parsed.User != nil || parsed.Fragment != "" || !strings.HasSuffix(parsed.Path, ".webp") {
		return false
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	return len(segments) >= 5 && segments[0] != "" &&
		segments[1] == "image" && segments[2] == "upload"
}

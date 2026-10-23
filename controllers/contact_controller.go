package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os"
	"strings"
	"time"

	"portfolio-api/database"
	"portfolio-api/services"

	"github.com/gofiber/fiber/v3"
)

type ContactController struct {
	sender services.EmailSender
}

func NewContactController(sender services.EmailSender) *ContactController {
	return &ContactController{sender: sender}
}

func (controller *ContactController) Submit(c fiber.Ctx) error {
	var request struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Message string `json:"message"`
		Website string `json:"website"`
	}
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format pesan tidak valid."})
	}

	if strings.TrimSpace(request.Website) != "" {
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"sent": true})
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Email = strings.TrimSpace(request.Email)
	request.Message = strings.TrimSpace(request.Message)
	if request.Name == "" || len(request.Name) > 120 || strings.ContainsAny(request.Name, "\r\n") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama wajib diisi dan maksimal 120 karakter."})
	}
	if !validContactEmail(request.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Masukkan alamat email yang valid."})
	}
	if request.Message == "" || len(request.Message) > 5000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pesan wajib diisi dan maksimal 5000 karakter."})
	}
	if controller.sender == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Layanan email belum dikonfigurasi."})
	}

	recipient, err := contactRecipient()
	if err != nil {
		log.Printf("contact email recipient unavailable: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Email kontak belum dikonfigurasi oleh admin."})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 25*time.Second)
	defer cancel()

	safeName := html.EscapeString(request.Name)
	safeEmail := html.EscapeString(request.Email)
	safeMessage := strings.ReplaceAll(html.EscapeString(request.Message), "\n", "<br>")
	if err := controller.sender.Send(ctx, services.EmailMessage{
		To:      recipient,
		ReplyTo: request.Email,
		Subject: "Pesan baru dari portfolio",
		HTML: fmt.Sprintf(
			"<h2>Pesan baru dari portfolio</h2><p><strong>Nama:</strong> %s</p><p><strong>Email:</strong> %s</p><p><strong>Pesan:</strong><br>%s</p>",
			safeName, safeEmail, safeMessage,
		),
		Text: fmt.Sprintf("Pesan baru dari portfolio\n\nNama: %s\nEmail: %s\n\nPesan:\n%s", request.Name, request.Email, request.Message),
	}); err != nil {
		log.Printf("contact notification email failed: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Pesan belum berhasil dikirim. Silakan coba lagi nanti."})
	}

	if err := controller.sender.Send(ctx, services.EmailMessage{
		To:      request.Email,
		Subject: "Pesan Anda sudah diterima",
		HTML: fmt.Sprintf(
			"<p>Halo %s,</p><p>Terima kasih sudah menghubungi saya melalui portfolio. Pesan Anda sudah berhasil saya terima dan akan saya baca.</p><p>Salam,<br>Billean</p>",
			safeName,
		),
		Text: fmt.Sprintf("Halo %s,\n\nTerima kasih sudah menghubungi saya melalui portfolio. Pesan Anda sudah berhasil saya terima dan akan saya baca.\n\nSalam,\nBillean", request.Name),
	}); err != nil {
		log.Printf("contact confirmation email failed: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error":  "Pesan sudah diteruskan, tetapi email konfirmasi belum berhasil dikirim.",
			"sent":   true,
			"notice": "Silakan coba lagi nanti untuk memastikan inbox Anda menerima konfirmasi.",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"sent":    true,
		"message": "Pesan berhasil dikirim dan email konfirmasi sudah dikirim ke alamat Anda.",
	})
}

func contactRecipient() (string, error) {
	if database.DB != nil {
		var raw []byte
		err := database.DB.Raw("SELECT value FROM site_content WHERE key = ?", "contact").Row().Scan(&raw)
		if err == nil {
			var contact struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(raw, &contact); err != nil || !validContactEmail(contact.Email) {
				return "", fmt.Errorf("stored contact email is invalid")
			}
			return contact.Email, nil
		}
	}
	if strings.TrimSpace(os.Getenv("CONTACT_EMAIL")) != "" {
		fallback := strings.TrimSpace(os.Getenv("CONTACT_EMAIL"))
		if validContactEmail(fallback) {
			return fallback, nil
		}
	}
	return "", fmt.Errorf("contact email not found")
}

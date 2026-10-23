package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"portfolio-api/services"

	"github.com/gofiber/fiber/v3"
)

type emailSenderFunc func(context.Context, services.EmailMessage) error

func (sender emailSenderFunc) Send(ctx context.Context, message services.EmailMessage) error {
	return sender(ctx, message)
}

func TestContactSubmissionSendsOwnerNoticeAndConfirmation(t *testing.T) {
	t.Setenv("CONTACT_EMAIL", "owner@example.com")
	messages := make([]services.EmailMessage, 0, 2)
	controller := NewContactController(emailSenderFunc(func(_ context.Context, message services.EmailMessage) error {
		messages = append(messages, message)
		return nil
	}))
	app := fiber.New()
	app.Post("/api/contact", controller.Submit)

	request := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(
		`{"name":"<Sam>","email":"sender@example.com","message":"Hello <script>alert(1)</script>"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if response.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected %d, got %d", fiber.StatusAccepted, response.StatusCode)
	}
	if len(messages) != 2 {
		t.Fatalf("expected owner notice and sender confirmation, got %d messages", len(messages))
	}
	if messages[0].To != "owner@example.com" || messages[0].ReplyTo != "sender@example.com" {
		t.Fatalf("owner notice has wrong recipient or reply-to: %#v", messages[0])
	}
	if strings.Contains(messages[0].HTML, "<script>") || !strings.Contains(messages[0].HTML, "&lt;script&gt;") {
		t.Fatal("user-provided message should be HTML-escaped")
	}
	if messages[1].To != "sender@example.com" || !strings.Contains(messages[1].Text, "sudah berhasil saya terima") {
		t.Fatalf("sender confirmation is incorrect: %#v", messages[1])
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["sent"] != true {
		t.Fatal("expected successful contact response")
	}
}

func TestContactSubmissionRejectsInvalidEmail(t *testing.T) {
	controller := NewContactController(emailSenderFunc(func(context.Context, services.EmailMessage) error {
		t.Fatal("email must not be sent for invalid input")
		return nil
	}))
	app := fiber.New()
	app.Post("/api/contact", controller.Submit)

	request := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(
		`{"name":"Sam","email":"not-an-email","message":"Hello"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected %d, got %d", fiber.StatusBadRequest, response.StatusCode)
	}
}

func TestContactHoneypotDoesNotSendEmail(t *testing.T) {
	controller := NewContactController(emailSenderFunc(func(context.Context, services.EmailMessage) error {
		t.Fatal("honeypot submissions should not trigger email")
		return nil
	}))
	app := fiber.New()
	app.Post("/api/contact", controller.Submit)

	request := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(
		`{"name":"Bot","email":"bot@example.com","message":"Spam","website":"filled"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if response.StatusCode != fiber.StatusAccepted {
		t.Fatalf("expected %d, got %d", fiber.StatusAccepted, response.StatusCode)
	}
}

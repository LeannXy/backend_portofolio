package services

import (
	"strings"
	"testing"
)

func TestNewSMTPMailerFromEnv(t *testing.T) {
	t.Setenv("MAIL_HOST", "smtp.gmail.com")
	t.Setenv("MAIL_PORT", "587")
	t.Setenv("MAIL_USERNAME", "owner@gmail.com")
	t.Setenv("MAIL_PASSWORD", "app-password")
	t.Setenv("MAIL_ENCRYPTION", "tls")
	t.Setenv("MAIL_FROM_ADDRESS", "owner@gmail.com")
	t.Setenv("MAIL_FROM_NAME", "Portfolio")

	mailer, err := NewSMTPMailerFromEnv()
	if err != nil {
		t.Fatalf("create SMTP mailer: %v", err)
	}
	if mailer == nil || mailer.host != "smtp.gmail.com" || mailer.port != "587" || mailer.encryption != "tls" {
		t.Fatalf("unexpected SMTP configuration: %#v", mailer)
	}
}

func TestNewSMTPMailerFromEnvRejectsIncompleteConfiguration(t *testing.T) {
	t.Setenv("MAIL_HOST", "smtp.gmail.com")
	t.Setenv("MAIL_PORT", "587")
	t.Setenv("MAIL_USERNAME", "owner@gmail.com")
	t.Setenv("MAIL_PASSWORD", "")
	t.Setenv("MAIL_ENCRYPTION", "tls")
	t.Setenv("MAIL_FROM_ADDRESS", "owner@gmail.com")
	t.Setenv("MAIL_FROM_NAME", "")

	if _, err := NewSMTPMailerFromEnv(); err == nil {
		t.Fatal("expected incomplete SMTP configuration to fail")
	}
}

func TestSMTPMailerFormatsAlternativeMessage(t *testing.T) {
	mailer := &SMTPMailer{
		fromAddress: "owner@gmail.com",
		fromName:    "Portfolio",
	}
	message, err := mailer.formatMessage(EmailMessage{
		To:      "sender@example.com",
		ReplyTo: "sender@example.com",
		Subject: "Pesan diterima",
		Text:    "Terima kasih",
		HTML:    "<p>Terima kasih</p>",
	})
	if err != nil {
		t.Fatalf("format message: %v", err)
	}
	formatted := string(message)
	for _, expected := range []string{
		`From: "Portfolio" <owner@gmail.com>`,
		"To: <sender@example.com>",
		"Reply-To: <sender@example.com>",
		"Subject:",
		"multipart/alternative",
		"text/plain; charset=UTF-8",
		"text/html; charset=UTF-8",
	} {
		if !strings.Contains(formatted, expected) {
			t.Errorf("formatted message missing %q", expected)
		}
	}
}

func TestSMTPMailerRejectsSubjectHeaderInjection(t *testing.T) {
	mailer := &SMTPMailer{
		host:        "smtp.gmail.com",
		port:        "587",
		username:    "owner@gmail.com",
		password:    "app-password",
		encryption:  "tls",
		fromAddress: "owner@gmail.com",
	}
	err := mailer.Send(t.Context(), EmailMessage{
		To:      "sender@example.com",
		Subject: "Hello\r\nBcc: attacker@example.com",
		HTML:    "<p>Hello</p>",
	})
	if err == nil || strings.Contains(err.Error(), "connect to SMTP server") {
		t.Fatalf("expected header injection to fail before connecting, got %v", err)
	}
}

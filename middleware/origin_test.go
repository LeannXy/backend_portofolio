package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRequireAllowedOrigin(t *testing.T) {
	t.Setenv("FRONTEND_ORIGINS", "https://portfolio.example.com, https://preview.example.com")

	app := fiber.New()
	app.Post("/admin", RequireAllowedOrigin, func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	tests := []struct {
		name           string
		origin         string
		expectedStatus int
	}{
		{name: "configured origin", origin: "https://portfolio.example.com", expectedStatus: fiber.StatusNoContent},
		{name: "unconfigured origin", origin: "https://attacker.example", expectedStatus: fiber.StatusForbidden},
		{name: "missing origin", expectedStatus: fiber.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/admin", nil)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}

			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if response.StatusCode != test.expectedStatus {
				t.Fatalf("expected status %d, got %d", test.expectedStatus, response.StatusCode)
			}
		})
	}
}

func TestValidCookieSameSite(t *testing.T) {
	for _, value := range []string{"", "Strict", "Lax", "None", "strict"} {
		if !ValidCookieSameSite(value) {
			t.Errorf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"Invalid", "none; Secure"} {
		if ValidCookieSameSite(value) {
			t.Errorf("expected %q to be invalid", value)
		}
	}
}

func TestValidateAllowedOrigins(t *testing.T) {
	if err := ValidateAllowedOrigins([]string{"https://portfolio.example.com"}); err != nil {
		t.Fatalf("expected valid origin list: %v", err)
	}
	for _, origins := range [][]string{
		nil,
		{"*"},
		{"https://portfolio.example.com/admin"},
		{"https://user@portfolio.example.com"},
	} {
		if err := ValidateAllowedOrigins(origins); err == nil {
			t.Errorf("expected invalid origin list %q to fail", origins)
		}
	}
}

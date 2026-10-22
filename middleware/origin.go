package middleware

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func AllowedOrigins() []string {
	configured := strings.TrimSpace(os.Getenv("FRONTEND_ORIGINS"))
	if configured == "" {
		return []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}

	origins := strings.Split(configured, ",")
	for index := range origins {
		origins[index] = strings.TrimRight(strings.TrimSpace(origins[index]), "/")
	}
	return origins
}

func ValidateAllowedOrigins(origins []string) error {
	if len(origins) == 0 {
		return fmt.Errorf("at least one frontend origin is required")
	}
	for _, origin := range origins {
		parsed, err := url.ParseRequestURI(origin)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") ||
			parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
			parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return fmt.Errorf("invalid frontend origin %q", origin)
		}
	}
	return nil
}

func RequireAllowedOrigin(c fiber.Ctx) error {
	origin := strings.TrimSpace(c.Get("Origin"))
	for _, allowedOrigin := range AllowedOrigins() {
		if origin != "" && origin == allowedOrigin {
			return c.Next()
		}
	}
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Origin tidak diizinkan."})
}

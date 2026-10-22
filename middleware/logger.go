package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
)

// RequestLogger mencatat waktu dan jalur setiap request
func RequestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log.Printf("[%s] %s %s - %v", c.Method(), c.Path(), c.IP(), time.Since(start))
		return err
	}
}
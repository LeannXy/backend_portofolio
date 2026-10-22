package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

const adminCookieName = "portfolio_admin_session"

func AdminAuth(c fiber.Ctx) error {
	if IsAdminToken(c.Cookies(adminCookieName)) {
		return c.Next()
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Silakan login sebagai admin."})
}

func IsAdminToken(token string) bool {
	return validAdminToken(token)
}

func NewAdminToken() string {
	payload := strconv.FormatInt(time.Now().Add(8*time.Hour).Unix(), 10)
	signature := sign(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "." + signature))
}

func validAdminToken(token string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	parts := strings.Split(string(decoded), ".")
	if len(parts) != 2 {
		return false
	}
	expiresAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() >= expiresAt {
		return false
	}
	expected, err := hex.DecodeString(sign(parts[0]))
	if err != nil {
		return false
	}
	actual, err := hex.DecodeString(parts[1])
	return err == nil && hmac.Equal(actual, expected)
}

func sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(os.Getenv("SESSION_SECRET")))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func AdminCookie(token string, expiresAt time.Time) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     adminCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HTTPOnly: true,
		Secure:   strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true"),
		SameSite: cookieSameSite(),
	}
}

func cookieSameSite() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SAME_SITE"))) {
	case "lax":
		return "Lax"
	case "none":
		return "None"
	default:
		return "Strict"
	}
}

func ValidCookieSameSite(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "strict", "lax", "none":
		return true
	default:
		return false
	}
}

package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadConfig memuat file .env saat aplikasi dimulai
func LoadConfig() {
	if _, err := os.Stat(".env"); err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatal("Gagal memeriksa file .env: ", err)
	}
	if err := godotenv.Load(); err != nil {
		log.Fatal("Gagal memuat file .env: ", err)
	}
}

// GetEnv mengambil nilai dari environment, atau mengembalikan nilai default jika kosong
func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

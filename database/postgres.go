package database

import (
	"database/sql" // Diperlukan untuk golang-migrate
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	"portfolio-api/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Driver untuk membaca file migrasi
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	dsn, err := databaseDSN()
	if err != nil {
		log.Fatal("Konfigurasi koneksi PostgreSQL tidak valid: ", err)
	}

	// 1. Koneksi GORM (Untuk operasi CRUD biasa)
	db, err := gorm.Open(gormPostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke database PostgreSQL!\n", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Gagal mengakses pool koneksi PostgreSQL: ", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	fmt.Println("Koneksi ke PostgreSQL berhasil!")
	DB = db

	// 2. Jalankan Migrasi SQL Murni
	runMigrations(dsn)
}

func databaseDSN() (string, error) {
	if connectionURL := strings.TrimSpace(config.GetEnv("DATABASE_URL", "")); connectionURL != "" {
		dsnURL, err := url.Parse(connectionURL)
		if err != nil || (dsnURL.Scheme != "postgres" && dsnURL.Scheme != "postgresql") ||
			dsnURL.Host == "" || dsnURL.User == nil || dsnURL.Path == "" {
			return "", fmt.Errorf("DATABASE_URL must be a PostgreSQL connection URL")
		}
		query := dsnURL.Query()
		query.Del("channel_binding")
		if strings.EqualFold(strings.TrimSpace(config.GetEnv("APP_ENV", "")), "production") {
			query.Set("sslmode", config.GetEnv("DB_SSLMODE", "require"))
		} else if query.Get("sslmode") == "" {
			query.Set("sslmode", config.GetEnv("DB_SSLMODE", "disable"))
		}
		dsnURL.RawQuery = query.Encode()
		return dsnURL.String(), nil
	}

	dsnURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.GetEnv("DB_USER", "postgres"), config.GetEnv("DB_PASSWORD", "")),
		Host:   net.JoinHostPort(config.GetEnv("DB_HOST", "localhost"), config.GetEnv("DB_PORT", "5432")),
		Path:   "/" + config.GetEnv("DB_NAME", "portfolio_db"),
	}
	query := url.Values{}
	query.Set("sslmode", config.GetEnv("DB_SSLMODE", "disable"))
	dsnURL.RawQuery = query.Encode()
	return dsnURL.String(), nil
}

func runMigrations(dsn string) {
	// Buka koneksi SQL standar (bukan GORM) karena golang-migrate membutuhkannya
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Gagal membuka koneksi SQL untuk migrasi:", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("Gagal membuat instance driver postgres:", err)
	}

	// Tentukan lokasi folder migrasi (pastikan jalurnya benar dari root proyek)
	m, err := migrate.NewWithDatabaseInstance(
		"file://database/migrations", // Path ke folder migrasi
		"postgres", driver)
	if err != nil {
		log.Fatal("Gagal inisialisasi migrasi:", err)
	}

	// Eksekusi semua file "up"
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal("Gagal menjalankan migrasi up:", err)
	}

	if err == migrate.ErrNoChange {
		fmt.Println("Migrasi: Tidak ada perubahan database baru.")
	} else {
		fmt.Println("Migrasi berhasil dijalankan!")
	}
}

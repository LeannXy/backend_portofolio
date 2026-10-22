package seeders

import (
	"log"

	"portfolio-api/database"
	"portfolio-api/models"
)

// SeedProjects menyuntikkan data percobaan awal ke dalam database
func SeedProjects() {
	var count int64
	database.DB.Model(&models.Project{}).Count(&count)

	if count == 0 {
		projects := []models.Project{
			{Title: "E-Commerce 3D", Description: "Toko online dengan pengalaman WebGL", ImageURL: "https://picsum.photos/seed/ecommerce3d/960/540", TechStack: "React, Three.js, Golang", Category: "SPATIAL & INTERACTION", Year: "2026", TextSide: "left", SortOrder: 0},
			{Title: "Origami Landing Page", Description: "Landing page interaktif dengan efek lipatan ruang", ImageURL: "https://picsum.photos/seed/origami/960/540", TechStack: "React, GSAP, R3F", Category: "CREATIVE ENGINEERING", Year: "2025", TextSide: "right", SortOrder: 1},
			{Title: "Black Hole Dashboard", Description: "Dashboard admin dengan transisi ekstrem", ImageURL: "https://picsum.photos/seed/blackhole/960/540", TechStack: "Next.js, Tailwind, Golang", Category: "DESIGN SYSTEM & TOOLS", Year: "2025", TextSide: "left", SortOrder: 2},
		}

		if err := database.DB.Create(&projects).Error; err != nil {
			log.Println("Gagal menyuntikkan data awal:", err)
		} else {
			log.Println("Berhasil menyuntikkan data percobaan portofolio!")
		}
	} else {
		log.Println("Data portofolio sudah ada, melewati proses seeding.")
	}
}

package controllers

import (
	"testing"
)

func TestProjectInputAcceptsPortfolioAndLegacyFields(t *testing.T) {
	project, err := (projectInput{
		Title:       "Project",
		Category:    "CREATIVE ENGINEERING",
		Year:        "2026",
		Description: "Portfolio entry",
		Image:       "/uploads/123e4567-e89b-12d3-a456-426614174000.jpg",
		TextSide:    "right",
	}).model()
	if err != nil {
		t.Fatalf("expected current portfolio format to validate: %v", err)
	}
	if project.Category != "CREATIVE ENGINEERING" || project.ImageURL == "" || project.TextSide != "right" {
		t.Fatalf("project fields were not mapped correctly: %+v", project)
	}

	legacyProject, err := (projectInput{
		Title:       "Legacy",
		TechStack:   "React, Fiber",
		Year:        "2025",
		Description: "Legacy API entry",
		ImageURL:    "https://example.com/project.jpg",
		TextSideAlt: "left",
	}).model()
	if err != nil {
		t.Fatalf("expected legacy API fields to validate: %v", err)
	}
	if legacyProject.Category != "React, Fiber" || legacyProject.ImageURL != "https://example.com/project.jpg" {
		t.Fatalf("legacy project fields were not mapped: %+v", legacyProject)
	}
}

func TestValidUploadedImagePath(t *testing.T) {
	for _, test := range []struct {
		path string
		want bool
	}{
		{path: "/uploads/123e4567-e89b-12d3-a456-426614174000.webp", want: true},
		{path: "/uploads/123e4567-e89b-12d3-a456-426614174000.jpg", want: true},
		{path: "/uploads/not-a-uuid.webp", want: false},
		{path: "/uploads/../image.webp", want: false},
	} {
		if got := validUploadedImagePath(test.path); got != test.want {
			t.Errorf("validUploadedImagePath(%q) = %v, want %v", test.path, got, test.want)
		}
	}
}

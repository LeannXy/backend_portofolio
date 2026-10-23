package controllers

import (
	"encoding/json"
	"testing"
)

func TestValidateSiteContent(t *testing.T) {
	tests := []struct {
		name string
		key  string
		data string
		ok   bool
	}{
		{
			name: "history",
			key:  "history",
			data: `[{"id":0,"year":"2024","title":"Developer","text":"Built software","details":["APIs"],"techStack":["Go"],"ratio":0.5}]`,
			ok:   true,
		},
		{
			name: "history ratio out of range",
			key:  "history",
			data: `[{"id":0,"year":"2024","title":"Developer","text":"Built software","details":["APIs"],"techStack":["Go"],"ratio":1.2}]`,
			ok:   false,
		},
		{
			name: "technologies",
			key:  "technologies",
			data: `[{"name":"Go","category":"backend","tag":"API","role":"Server","desc":"Build APIs"}]`,
			ok:   true,
		},
		{
			name: "unsupported technology category",
			key:  "technologies",
			data: `[{"name":"Go","category":"unknown","tag":"API","role":"Server","desc":"Build APIs"}]`,
			ok:   false,
		},
		{
			name: "socials",
			key:  "socials",
			data: `{"github":{"label":"GitHub","hoverLabel":"Open","url":"https://github.com/example"},"linkedin":{"label":"LinkedIn","hoverLabel":"Open","url":""},"tiktok":{"label":"TikTok","hoverLabel":"Open","url":""},"instagram":{"label":"Instagram","hoverLabel":"Open","url":""}}`,
			ok:   true,
		},
		{
			name: "unsafe social URL",
			key:  "socials",
			data: `{"github":{"label":"GitHub","hoverLabel":"Open","url":"javascript:alert(1)"},"linkedin":{"label":"LinkedIn","hoverLabel":"Open","url":""},"tiktok":{"label":"TikTok","hoverLabel":"Open","url":""},"instagram":{"label":"Instagram","hoverLabel":"Open","url":""}}`,
			ok:   false,
		},
		{
			name: "default profile image",
			key:  "profileImage",
			data: `"/profile.webp"`,
			ok:   true,
		},
		{
			name: "uploaded profile WebP",
			key:  "profileImage",
			data: `"/uploads/123e4567-e89b-12d3-a456-426614174000.webp"`,
			ok:   true,
		},
		{
			name: "Cloudinary profile WebP",
			key:  "profileImage",
			data: `"https://res.cloudinary.com/demo/image/upload/v123/portfolio/profile.webp"`,
			ok:   true,
		},
		{
			name: "Cloudinary profile image must use HTTPS",
			key:  "profileImage",
			data: `"http://res.cloudinary.com/demo/image/upload/v123/portfolio/profile.webp"`,
			ok:   false,
		},
		{
			name: "profile image cannot use another external host",
			key:  "profileImage",
			data: `"https://attacker.example/demo/image/upload/profile.webp"`,
			ok:   false,
		},
		{
			name: "profile image must be WebP",
			key:  "profileImage",
			data: `"/uploads/123e4567-e89b-12d3-a456-426614174000.jpg"`,
			ok:   false,
		},
		{
			name: "contact settings",
			key:  "contact",
			data: `{"email":"owner@example.com","githubRepo":"https://github.com/owner/project"}`,
			ok:   true,
		},
		{
			name: "contact email must be valid",
			key:  "contact",
			data: `{"email":"not-an-email","githubRepo":"https://github.com/owner/project"}`,
			ok:   false,
		},
		{
			name: "contact repository must use GitHub HTTPS",
			key:  "contact",
			data: `{"email":"owner@example.com","githubRepo":"https://attacker.example/owner/project"}`,
			ok:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateContent(test.key, json.RawMessage(test.data)); (err == nil) != test.ok {
				t.Fatalf("validateContent() error = %v, want success %v", err, test.ok)
			}
		})
	}
}

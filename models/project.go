package models

import "gorm.io/gorm"

type Project struct {
	gorm.Model
	Title       string `json:"title" gorm:"size:255;not null"`
	Description string `json:"description" gorm:"type:text;not null"`
	ImageURL    string `json:"image_url" gorm:"size:255;not null"`
	TechStack   string `json:"tech_stack" gorm:"size:255;not null"`
	Category    string `json:"category" gorm:"size:100;not null;default:''"`
	Year        string `json:"year" gorm:"size:12;not null;default:''"`
	Link        string `json:"link" gorm:"type:text"`
	TextSide    string `json:"text_side" gorm:"size:5;not null;default:'left'"`
	SortOrder   int    `json:"sort_order" gorm:"not null;default:0"`
}

package models

import "gorm.io/gorm"

type Track struct {
	gorm.Model
	ISRC       string   `json:"isrc" gorm:"unique_index"`
	ImageURL   string   `json:"image_uri"`
	Name       string   `json:"title"`
	Popularity int      `json:"popularity"`
	Artists    []Artist `json:"artists" gorm:"foreignkey:TrackID"`
}

type Artist struct {
	gorm.Model
	Name    string `json:"name"`
	TrackID uint   `json:"-"`
}

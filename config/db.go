package config

import (
	"fmt"
	"log"

	"github.com/faizan/spotify/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	username := "root"
	password := "medika@123"
	host := "localhost"
	port := "3306"
	dbName := "spotify" //local db

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, host, port, dbName)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	DB.AutoMigrate(&models.Track{}, &models.Artist{})
}

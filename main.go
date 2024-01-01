package main

import (
	"log"

	"github.com/faizan/spotify/config"
	"github.com/faizan/spotify/handlers"
)

func main() {
	config.Init()

	router := handlers.SetupRouter()

	err := router.Run(":8080")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

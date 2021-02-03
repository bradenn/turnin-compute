package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

func Init() {
	if os.Getenv("ENV") == "production" {
		// 	Do production things
	} else {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}
}

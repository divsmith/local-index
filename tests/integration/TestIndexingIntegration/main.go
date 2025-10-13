package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	config := LoadConfig()

	if err := StartServer(config); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Println("Server started successfully")
}

func LoadConfig() *Config {
	return &Config{
		Port:    8080,
		Timeout: 30,
	}
}

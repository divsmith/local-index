package main

import (
	"code-search/tests/integration/resources/TestIndexingIntegration/config"
	"code-search/tests/integration/resources/TestIndexingIntegration/server"
	"fmt"
	"log"
)

func main() {
	cfg := LoadConfig()

	if err := StartServer(cfg); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Println("Server started successfully")
}

func LoadConfig() *config.Config {
	return &config.Config{
		Port:    8080,
		Timeout: 30,
		Host:    "localhost",
		Debug:   false,
	}
}

func StartServer(cfg *config.Config) error {
	srv := server.NewServer(cfg)
	srv.SetupRoutes()
	return srv.Start()
}


// New function added for incremental indexing test
func NewFunction() {
	fmt.Println("This is a new function")
}

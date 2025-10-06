package main

import "fmt"

func main() {
	config := LoadConfig()
	if err := StartServer(config); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

func LoadConfig() *Config {
	return &Config{Port: 8080, Timeout: 30}
}

func StartServer(config *Config) error {
	fmt.Printf("Server starting on port %d\n", config.Port)
	return nil
}

type Config struct {
	Port    int
	Timeout int
}

package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/robmartinson/pg2lite/internal/config"
)

func main() {
	// load the .env file if it exists
	godotenv.Load()

	if err := config.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

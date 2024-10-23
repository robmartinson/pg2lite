package main

import (
	"fmt"
	"os"

	"github.com/robmartinson/pg2lite/internal/config"
)

func main() {
	if err := config.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

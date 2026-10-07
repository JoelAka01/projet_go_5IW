package main

import (
	"fmt"
	"os"

	"ecommerce-cli/ui/admin"
)

func main() {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "http://localhost:8080"
	}

	if err := admin.Run(serverAddr); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur:", err)
		os.Exit(1)
	}
}

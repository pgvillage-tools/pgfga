// Package main will hold the main function
package main

import (
	"log"

	"github.com/pgvillage-tools/pgfga/internal/handler"
)

func main() {
	handler.Initialize()

	fga, err := handler.NewPgFgaHandler()
	if err != nil {
		log.Fatalf("Error occurred on getting config: %e", err)
	}

	err = fga.Handle()
	if err != nil {
		log.Fatalf("Error occurred while managing resources: %e", err)
	}
}

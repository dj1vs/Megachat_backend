package main

// @BasePath /

import (
	"context"
	"log"
	"megachat/internal/pkg/app"
	// http-swagger middleware
)

func main() {

	log.Println("Megachat backend started!")

	a, err := app.New(context.Background())
	if err != nil {
		log.Println(err)

		return
	}

	a.StartServer()

	log.Println("Megachat backend is shutting down...")
}

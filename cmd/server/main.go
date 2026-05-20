package main

import (
	"log"

	"github.com/kirban/social-media/internal/app"
)

func main() {
	server, err := app.NewAppServer()

	if err != nil {
		log.Fatal(err)
	}

	server.Run()
}

package main

import (
	"log"

	"social-network/backend/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

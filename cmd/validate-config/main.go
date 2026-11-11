package main

import (
	"log"

	"github.com/jonsten/morgonfikan"
)

func main() {
	if _, err := morgonfikan.LoadEmbeddedConfig(); err != nil {
		log.Fatalf("invalid menu configuration: %v", err)
	}
	log.Print("menu configuration is valid")
}

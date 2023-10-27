package main

import (
	"log"

	"github.com/anmol1vw13/go-bank/api"
)

func main() {
	store, err := api.NewPostgresStore()
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Init(); err != nil {
		log.Fatal(err)
	}
	server := api.NewAPIServer("8005", store)
	server.Run()
}

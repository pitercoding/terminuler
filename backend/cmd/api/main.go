package main

import (
	"log"
	"net/http"

	"github.com/pitercoding/terminuler/internal/routes"
)

func main() {
	mux := http.NewServeMux()

	routes.RegisterRoutes(mux)

	log.Println("Terminuler API running on http://localhost:8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

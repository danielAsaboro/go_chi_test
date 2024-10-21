package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	// Define router
	r := chi.NewRouter()

	r.HandleFunc("/users/{id:[0-9]+}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		name := getUser(id)

		if name == "unknown" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		reply := fmt.Sprintf("user %s (id %s)\n", name, id)
		w.Write([]byte(reply))
	}))

	// Serve router
	log.Println("Starting server on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getUser(id string) string {
	if id == "123" {
		return "chi tester"
	}
	return "unknown"
}

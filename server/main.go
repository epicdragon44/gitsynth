package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Define the port to listen on
	port := ":8080"

	// Create a new HTTP server mux
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/api/hello", HelloHandler)
	mux.HandleFunc("/api/run", RunHandler)

	// Start the server
	fmt.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}

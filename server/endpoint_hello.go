package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HelloHandler handles GET and POST requests to /api/hello
func HelloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Handle GET request
		response := Response{
			Message: "Hello, World!",
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// Handle POST request
		var requestBody struct {
			Name string `json:"name"`
		}

		// Parse the request body
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{Message: "Invalid request payload"})
			return
		}

		// Create and send response
		name := requestBody.Name
		if name == "" {
			name = "stranger"
		}

		response := Response{
			Message: fmt.Sprintf("Hello, %s!", name),
			Data: map[string]string{
				"timestamp": fmt.Sprintf("%v", time.Now()),
			},
		}
		json.NewEncoder(w).Encode(response)

	default:
		// Handle unsupported methods
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{Message: fmt.Sprintf("Method %s not allowed", r.Method)})
	}
}

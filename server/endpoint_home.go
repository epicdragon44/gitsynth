package main

import (
	"fmt"
	"net/http"
)

// HomeHandler handles requests to the root path
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		fmt.Fprintf(w, "Welcome to the Go Web Server!")
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Method %s not allowed", r.Method)
	}
}

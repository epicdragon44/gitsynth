package main

import (
	"html/template"
	"net/http"
	"path/filepath"
)

// HomeHandler handles requests to the root path
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html")

		// Load the HTML template from file
		tmplPath := filepath.Join("templates", "home.html")
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Method " + r.Method + " not allowed"))
	}
}

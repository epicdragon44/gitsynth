package main

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/rs/zerolog"
)

// HomeHandler handles requests to the root path
type HomeHandler struct {}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		logger.Info().Str("user-agent", r.Header.Get("User-Agent")).Msg("Home page accessed")
		
		w.Header().Set("Content-Type", "text/html")

		// Load the HTML template from file
		tmplPath := filepath.Join("templates", "home.html")
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to parse template")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Method " + r.Method + " not allowed"))
	}
}

// type assertion
var _ http.Handler = &HomeHandler{}

package handlers

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
)

func render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := c.Render(r.Context(), w); err != nil {
		log.Printf("render: %v", err)
	}
}

func serverError(w http.ResponseWriter, what string, err error) {
	log.Printf("%s: %v", what, err)
	http.Error(w, "something went wrong", http.StatusInternalServerError)
}
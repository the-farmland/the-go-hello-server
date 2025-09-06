package handler

import (
	"fmt"
	"net/http"
)

// Handler is the main entry point for the Vercel serverless function.
// Vercel expects a function with this exact signature to handle HTTP requests.
func Handler(w http.ResponseWriter, r *http.Request) {
	// Set the content type header to indicate we are sending HTML.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Fprintf writes a formatted string to the ResponseWriter.
	// This will be the body of the HTTP response.
	fmt.Fprintf(w, "<h1>Hello, World from Go on Vercel!</h1>")
}

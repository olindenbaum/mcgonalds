package middleware

import (
	"fmt"
	"net/http"

	"github.com/fatih/color"
	"github.com/gorilla/mux"
)

func DebugMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use colors for different parts of the log
		methodColor := color.New(color.FgCyan).SprintFunc()
		pathColor := color.New(color.FgGreen).SprintFunc()
		routeColor := color.New(color.FgYellow).SprintFunc()

		// Get the route being processed
		route := mux.CurrentRoute(r)
		routePath, _ := route.GetPathTemplate()

		// Print the request method, path, and route
		fmt.Printf("Processing %s request for %s (Route: %s)\n",
			methodColor(r.Method),
			pathColor(r.URL.Path),
			routeColor(routePath),
		)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

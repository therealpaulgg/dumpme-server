package router

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/therealpaulgg/dumpme-server/router/download"
	"github.com/therealpaulgg/dumpme-server/router/upload"
)

// Router represents what is launched at the beginning of the application
type Router struct {
	Port  int
	Env   string
	Proto string
}

// Launch the application by gathering all routes together
func (info *Router) Launch() {
	r := chi.NewRouter()
	r.Use(middleware.SetHeader("Content-Type", "application/json"))
	r.Use(middleware.Logger)
	r.Use(middleware.StripSlashes)
	// CORS options - this is an API, should be usable from specified front-end endpoints (temporarily *)
	// TODO: change AllowedOrigins
	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	r.Use(cors.Handler)

	// Mount routers here
	r.Mount("/upload", upload.Router())
	r.Mount("/download", download.Router())

	// TODO: Add NotFound and MethodNotAllowed and other necessary errors as custom responses
	fmt.Printf("Listening on port %d\n", info.Port)
	fmt.Println("Running in mode: " + info.Proto)
	fmt.Printf("Environment: %s\n", info.Env)
	err := http.ListenAndServe(fmt.Sprintf(":%d", info.Port), r)
	log.Fatal(err)
}

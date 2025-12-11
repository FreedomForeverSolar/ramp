package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ramp/internal/uiapi"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	port := flag.Int("port", 37429, "Port to run the server on")
	flag.Parse()

	// Create router
	router := mux.NewRouter()

	// Create API server
	server := uiapi.NewServer()

	// Register routes
	apiRouter := router.PathPrefix("/api").Subrouter()

	// Project routes
	apiRouter.HandleFunc("/projects", server.ListProjects).Methods("GET")
	apiRouter.HandleFunc("/projects", server.AddProject).Methods("POST")
	apiRouter.HandleFunc("/projects/reorder", server.ReorderProjects).Methods("PUT")
	apiRouter.HandleFunc("/projects/{id}", server.RemoveProject).Methods("DELETE")
	apiRouter.HandleFunc("/projects/{id}/favorite", server.ToggleFavorite).Methods("PUT")

	// Feature routes
	apiRouter.HandleFunc("/projects/{id}/features", server.ListFeatures).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/features", server.CreateFeature).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/features/prune", server.PruneFeatures).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/features/{name}", server.DeleteFeature).Methods("DELETE")

	// Config routes (local preferences)
	apiRouter.HandleFunc("/projects/{id}/config/status", server.GetConfigStatus).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/config", server.GetConfig).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/config", server.SaveConfig).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/config", server.ResetConfig).Methods("DELETE")

	// Command routes
	apiRouter.HandleFunc("/projects/{id}/commands", server.ListCommands).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/commands/{commandName}/run", server.RunCommand).Methods("POST")

	// Source repos routes
	apiRouter.HandleFunc("/projects/{id}/source-repos", server.GetSourceRepos).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/source-repos/refresh", server.RefreshSourceRepos).Methods("POST")

	// Terminal routes
	apiRouter.HandleFunc("/terminal/open", server.OpenTerminal).Methods("POST")

	// App settings routes
	apiRouter.HandleFunc("/settings", server.GetAppSettings).Methods("GET")
	apiRouter.HandleFunc("/settings", server.SaveAppSettings).Methods("POST")

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// WebSocket for real-time updates
	router.HandleFunc("/ws/logs", server.HandleWebSocket)

	// Enable CORS for development
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Ramp UI backend starting on port %d", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

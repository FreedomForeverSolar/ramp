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
	apiRouter.HandleFunc("/projects/{id}", server.RemoveProject).Methods("DELETE")

	// Feature routes
	apiRouter.HandleFunc("/projects/{id}/features", server.ListFeatures).Methods("GET")
	apiRouter.HandleFunc("/projects/{id}/features", server.CreateFeature).Methods("POST")
	apiRouter.HandleFunc("/projects/{id}/features/{name}", server.DeleteFeature).Methods("DELETE")

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

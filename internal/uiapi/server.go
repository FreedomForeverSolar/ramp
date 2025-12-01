package uiapi

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Server holds the API server state
type Server struct {
	// WebSocket connections for broadcasting updates
	wsConnections map[*websocket.Conn]bool
	wsMutex       sync.RWMutex

	// WebSocket upgrader
	upgrader websocket.Upgrader
}

// NewServer creates a new API server
func NewServer() *Server {
	return &Server{
		wsConnections: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from our frontend
				origin := r.Header.Get("Origin")
				return origin == "http://localhost:5173" ||
					origin == "http://localhost:3000" ||
					origin == "" // Allow no origin (e.g., from Electron)
			},
		},
	}
}

// broadcast sends a message to all connected WebSocket clients
func (s *Server) broadcast(message interface{}) {
	s.wsMutex.RLock()
	defer s.wsMutex.RUnlock()

	for conn := range s.wsConnections {
		if err := conn.WriteJSON(message); err != nil {
			// Connection will be cleaned up by read loop
			continue
		}
	}
}

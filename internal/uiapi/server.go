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

	// Per-project locks for serializing feature operations
	// Prevents concurrent create/delete operations from conflicting at git level
	projectLocks sync.Map // map[projectID]*sync.Mutex
}

// NewServer creates a new API server
func NewServer() *Server {
	return &Server{
		wsConnections: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from our frontend
				origin := r.Header.Get("Origin")
				// Allow dev servers
				if origin == "http://localhost:5173" || origin == "http://localhost:3000" {
					return true
				}
				// Allow Electron production (file:// protocol) and empty origin
				if origin == "" || origin == "file://" {
					return true
				}
				// Allow null origin (some browsers send this for file://)
				if origin == "null" {
					return true
				}
				return false
			},
		},
	}
}

// broadcast sends a message to all connected WebSocket clients
// Uses full lock (not RLock) because gorilla/websocket WriteJSON is not
// safe for concurrent calls on the same connection
func (s *Server) broadcast(message interface{}) {
	s.wsMutex.Lock()
	defer s.wsMutex.Unlock()

	for conn := range s.wsConnections {
		if err := conn.WriteJSON(message); err != nil {
			// Connection will be cleaned up by read loop
			continue
		}
	}
}

// acquireProjectLock acquires a mutex for the given project ID.
// Returns an unlock function that must be called to release the lock.
// This serializes feature operations per-project to prevent git conflicts.
func (s *Server) acquireProjectLock(projectID string) func() {
	lock, _ := s.projectLocks.LoadOrStore(projectID, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	return func() { mu.Unlock() }
}

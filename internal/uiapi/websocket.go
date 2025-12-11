package uiapi

import (
	"log"
	"net/http"
)

// HandleWebSocket handles WebSocket connections for real-time updates
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Register connection
	s.wsMutex.Lock()
	s.wsConnections[conn] = true
	s.wsMutex.Unlock()

	log.Println("WebSocket client connected")

	// Send welcome message
	conn.WriteJSON(WSMessage{
		Type:    "connected",
		Message: "Connected to Ramp UI backend",
	})

	// Read loop to detect disconnection
	go func() {
		defer func() {
			s.wsMutex.Lock()
			delete(s.wsConnections, conn)
			s.wsMutex.Unlock()
			conn.Close()
			log.Println("WebSocket client disconnected")
		}()

		for {
			// Read messages (we mostly just detect disconnection)
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

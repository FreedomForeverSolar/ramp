package operations

// WSBroadcaster is a function that broadcasts a message to all WebSocket clients.
type WSBroadcaster func(msg interface{})

// WSProgressReporter implements ProgressReporter for WebSocket broadcasting.
type WSProgressReporter struct {
	broadcast WSBroadcaster
	operation string
}

// WSMessage is the message format for WebSocket broadcasts.
// This mirrors the structure in internal/uiapi/models.go.
type WSMessage struct {
	Type       string `json:"type"`
	Operation  string `json:"operation"`
	Message    string `json:"message"`
	Percentage int    `json:"percentage,omitempty"`
}

// NewWSProgressReporter creates a progress reporter for WebSocket usage.
func NewWSProgressReporter(operation string, broadcast WSBroadcaster) *WSProgressReporter {
	return &WSProgressReporter{
		broadcast: broadcast,
		operation: operation,
	}
}

func (r *WSProgressReporter) Start(message string) {
	r.broadcast(WSMessage{Type: "progress", Operation: r.operation, Message: message})
}

func (r *WSProgressReporter) Update(message string) {
	r.broadcast(WSMessage{Type: "progress", Operation: r.operation, Message: message})
}

func (r *WSProgressReporter) UpdateWithProgress(message string, percentage int) {
	r.broadcast(WSMessage{Type: "progress", Operation: r.operation, Message: message, Percentage: percentage})
}

func (r *WSProgressReporter) Success(message string) {
	r.broadcast(WSMessage{Type: "progress", Operation: r.operation, Message: message})
}

func (r *WSProgressReporter) Error(message string) {
	r.broadcast(WSMessage{Type: "error", Operation: r.operation, Message: message})
}

func (r *WSProgressReporter) Warning(message string) {
	r.broadcast(WSMessage{Type: "warning", Operation: r.operation, Message: message})
}

func (r *WSProgressReporter) Info(message string) {
	r.broadcast(WSMessage{Type: "info", Operation: r.operation, Message: message})
}

func (r *WSProgressReporter) Complete(message string) {
	r.broadcast(WSMessage{Type: "complete", Operation: r.operation, Message: message, Percentage: 100})
}

// WSOutputStreamer implements OutputStreamer for WebSocket broadcasting.
type WSOutputStreamer struct {
	broadcast WSBroadcaster
	operation string
}

// NewWSOutputStreamer creates an output streamer for WebSocket usage.
func NewWSOutputStreamer(operation string, broadcast WSBroadcaster) *WSOutputStreamer {
	return &WSOutputStreamer{
		broadcast: broadcast,
		operation: operation,
	}
}

func (s *WSOutputStreamer) WriteLine(line string) {
	s.broadcast(WSMessage{Type: "output", Operation: s.operation, Message: line})
}

func (s *WSOutputStreamer) WriteErrorLine(line string) {
	s.broadcast(WSMessage{Type: "output", Operation: s.operation, Message: "[stderr] " + line})
}

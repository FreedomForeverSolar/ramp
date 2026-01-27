package operations

// WSBroadcaster is a function that broadcasts a message to all WebSocket clients.
type WSBroadcaster func(msg interface{})

// WSProgressReporter implements ProgressReporter for WebSocket broadcasting.
type WSProgressReporter struct {
	broadcast WSBroadcaster
	operation string
	target    string // Feature name or other identifier for filtering
	command   string // Command name for run operations
}

// WSMessage is the message format for WebSocket broadcasts.
// This mirrors the structure in internal/uiapi/models.go.
type WSMessage struct {
	Type       string `json:"type"`
	Operation  string `json:"operation"`
	Message    string `json:"message"`
	Percentage int    `json:"percentage,omitempty"`
	Target     string `json:"target,omitempty"`  // Feature name for filtering messages
	Command    string `json:"command,omitempty"` // Command name for run operations
}

// NewWSProgressReporter creates a progress reporter for WebSocket usage.
// The target parameter identifies the specific feature for message filtering.
func NewWSProgressReporter(operation string, target string, broadcast WSBroadcaster) *WSProgressReporter {
	return &WSProgressReporter{
		broadcast: broadcast,
		operation: operation,
		target:    target,
	}
}

// NewWSProgressReporterWithCommand creates a progress reporter with command context.
func NewWSProgressReporterWithCommand(operation, target, command string, broadcast WSBroadcaster) *WSProgressReporter {
	return &WSProgressReporter{
		broadcast: broadcast,
		operation: operation,
		target:    target,
		command:   command,
	}
}

func (r *WSProgressReporter) Start(message string) {
	r.broadcast(WSMessage{Type: "progress", Operation: r.operation, Message: message, Target: r.target, Command: r.command})
}

func (r *WSProgressReporter) Update(message string) {
	r.broadcast(WSMessage{Type: "progress", Operation: r.operation, Message: message, Target: r.target, Command: r.command})
}

func (r *WSProgressReporter) UpdateWithProgress(message string, percentage int) {
	r.broadcast(WSMessage{Type: "progress", Operation: r.operation, Message: message, Percentage: percentage, Target: r.target, Command: r.command})
}

func (r *WSProgressReporter) Stop() {
	// No-op for WebSocket - there's no spinner to stop
}

func (r *WSProgressReporter) Success(message string) {
	r.broadcast(WSMessage{Type: "progress", Operation: r.operation, Message: message, Target: r.target, Command: r.command})
}

func (r *WSProgressReporter) Error(message string) {
	r.broadcast(WSMessage{Type: "error", Operation: r.operation, Message: message, Target: r.target, Command: r.command})
}

func (r *WSProgressReporter) Warning(message string) {
	r.broadcast(WSMessage{Type: "warning", Operation: r.operation, Message: message, Target: r.target, Command: r.command})
}

func (r *WSProgressReporter) Info(message string) {
	r.broadcast(WSMessage{Type: "info", Operation: r.operation, Message: message, Target: r.target, Command: r.command})
}

func (r *WSProgressReporter) Complete(message string) {
	r.broadcast(WSMessage{Type: "complete", Operation: r.operation, Message: message, Percentage: 100, Target: r.target, Command: r.command})
}

// WSOutputStreamer implements OutputStreamer for WebSocket broadcasting.
type WSOutputStreamer struct {
	broadcast WSBroadcaster
	operation string
	target    string
	command   string
}

// NewWSOutputStreamer creates an output streamer for WebSocket usage.
func NewWSOutputStreamer(operation string, broadcast WSBroadcaster) *WSOutputStreamer {
	return &WSOutputStreamer{
		broadcast: broadcast,
		operation: operation,
	}
}

// NewWSOutputStreamerWithContext creates an output streamer with target and command context.
func NewWSOutputStreamerWithContext(operation, target, command string, broadcast WSBroadcaster) *WSOutputStreamer {
	return &WSOutputStreamer{
		broadcast: broadcast,
		operation: operation,
		target:    target,
		command:   command,
	}
}

func (s *WSOutputStreamer) WriteLine(line string) {
	s.broadcast(WSMessage{Type: "output", Operation: s.operation, Message: line, Target: s.target, Command: s.command})
}

func (s *WSOutputStreamer) WriteErrorLine(line string) {
	s.broadcast(WSMessage{Type: "output", Operation: s.operation, Message: "[stderr] " + line, Target: s.target, Command: s.command})
}

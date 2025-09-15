#!/bin/bash

# Demo Logs Script
# Shows logs from all services (simulated in demo)

echo "📊 Viewing logs for microservices..."
echo "   Feature: $RAMP_WORKTREE_NAME"
echo "   Port Range: $RAMP_PORT, ${RAMP_PORT}1, ${RAMP_PORT}2"
echo ""

# Create a logs directory if it doesn't exist
mkdir -p "$RAMP_TREES_DIR/logs"

# Simulate log files (in a real scenario, these would be actual service logs)
echo "🎭 Simulating service logs..."

# Create demo log files
cat > "$RAMP_TREES_DIR/logs/json-server.log" << EOF
[$(date)] JSON Server starting on port $RAMP_PORT
[$(date)] Loading db.json
[$(date)] Ready on http://localhost:$RAMP_PORT
[$(date)] Mock REST API serving endpoints
EOF

cat > "$RAMP_TREES_DIR/logs/api-gateway.log" << EOF
[$(date)] TypeScript API Gateway starting on port ${RAMP_PORT}1
[$(date)] Node.js server initialized
[$(date)] Connected to JSON Server at http://localhost:$RAMP_PORT
[$(date)] TypeScript compilation completed
EOF

cat > "$RAMP_TREES_DIR/logs/hello-world.log" << EOF
[$(date)] Hello World service starting on port ${RAMP_PORT}2
[$(date)] Simple demo service initialized
[$(date)] Ready to respond with greetings
[$(date)] Hello World service ready
EOF

# Display the logs
echo "📋 JSON Server Logs:"
echo "----------------------------------------"
cat "$RAMP_TREES_DIR/logs/json-server.log"
echo ""

echo "📋 API Gateway Logs:"
echo "----------------------------------------"
cat "$RAMP_TREES_DIR/logs/api-gateway.log"
echo ""

echo "📋 Hello World Logs:"
echo "----------------------------------------"
cat "$RAMP_TREES_DIR/logs/hello-world.log"
echo ""

echo "💡 Log files created at: $RAMP_TREES_DIR/logs/"
echo "💡 In a real project, this would tail actual service logs"
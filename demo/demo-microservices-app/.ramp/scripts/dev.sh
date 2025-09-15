#!/bin/bash

# Demo Dev Script
# Simulates starting all development services

echo "🚀 Starting development services for '$RAMP_WORKTREE_NAME'..."
echo ""

# Show the environment
echo "🔧 Environment Configuration:"
echo "   Project:     $RAMP_WORKTREE_NAME"
echo "   Port Range:  $RAMP_PORT, ${RAMP_PORT}1, ${RAMP_PORT}2"
echo "   Trees Dir:   $RAMP_TREES_DIR"
echo ""

# In a real scenario, this would start actual services
echo "🎭 Demo Mode: Simulating service startup..."
echo ""

echo "▶️  JSON Server (Mock API)"
echo "   Port: $RAMP_PORT"
echo "   Command would be: cd $RAMP_TREES_DIR/json-server && npm run start"
echo "   Status: ✅ Ready"
echo ""

echo "▶️  API Gateway (TypeScript)"
echo "   Port: ${RAMP_PORT}1"
echo "   Command would be: cd $RAMP_TREES_DIR/node-typescript-boilerplate && npm run dev"
echo "   Status: ✅ Ready"
echo ""

echo "▶️  Auth Service (Hello World)"
echo "   Port: ${RAMP_PORT}2"
echo "   Command would be: cd $RAMP_TREES_DIR/Hello-World && echo 'Hello World Demo'"
echo "   Status: ✅ Ready"
echo ""

# Create a simple status file
cat > "$RAMP_TREES_DIR/dev-status.json" << EOF
{
  "feature": "$RAMP_WORKTREE_NAME",
  "status": "running",
  "services": {
    "json-server": {
      "port": $RAMP_PORT,
      "url": "http://localhost:$RAMP_PORT",
      "status": "ready"
    },
    "api-gateway": {
      "port": ${RAMP_PORT}1,
      "url": "http://localhost:${RAMP_PORT}1",
      "status": "ready"
    },
    "hello-world": {
      "port": ${RAMP_PORT}2,
      "url": "http://localhost:${RAMP_PORT}2",
      "status": "ready"
    }
  },
  "started": "$(date -Iseconds)"
}
EOF

echo "🎯 Development environment ready!"
echo "   Status file: $RAMP_TREES_DIR/dev-status.json"
echo ""
echo "🔗 Service URLs:"
echo "   JSON Server:  http://localhost:$RAMP_PORT"
echo "   API Gateway:  http://localhost:${RAMP_PORT}1"
echo "   Hello World:  http://localhost:${RAMP_PORT}2"
echo ""
echo "💡 Next steps:"
echo "   ramp run open  # Open in VS Code"
echo "   ramp run logs  # View service logs"
echo "   ramp status    # Check project status"
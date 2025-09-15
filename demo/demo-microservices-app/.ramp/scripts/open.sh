#!/bin/bash

# Demo Open Script
# Opens the project in VS Code for development

echo "🔧 Opening demo microservices project..."
echo "   Feature: $RAMP_WORKTREE_NAME"
echo "   Trees Directory: $RAMP_TREES_DIR"

# Check if VS Code is available
if command -v code >/dev/null 2>&1; then
    echo "📝 Opening in VS Code..."
    code "$RAMP_TREES_DIR"
else
    echo "⚠️  VS Code not found. You can manually open:"
    echo "   $RAMP_TREES_DIR"
fi

# Show some helpful information
echo ""
echo "🎯 Quick Navigation:"
echo "   JSON Server: cd $RAMP_TREES_DIR/json-server"
echo "   API Gateway: cd $RAMP_TREES_DIR/node-typescript-boilerplate"
echo "   Hello World: cd $RAMP_TREES_DIR/Hello-World"
echo ""
echo "🔗 Service URLs (when running):"
echo "   JSON Server: http://localhost:$RAMP_PORT"
echo "   API Gateway: http://localhost:${RAMP_PORT}1"
echo "   Hello World: http://localhost:${RAMP_PORT}2"
echo ""
echo "💡 Try: ramp run dev (to start all services)"
echo "💡 Try: ramp run logs (to view service logs)"
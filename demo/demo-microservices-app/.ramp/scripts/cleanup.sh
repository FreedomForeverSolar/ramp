#!/bin/bash

# Demo Microservices Cleanup Script
# This script demonstrates cleanup operations when tearing down a feature branch

set -e

echo "🧹 Cleaning up demo microservices environment..."
echo "   Feature: $RAMP_WORKTREE_NAME"
echo "   Trees Dir: $RAMP_TREES_DIR"

# Stop any running processes (in a real scenario)
echo "🛑 Stopping services..."
# pkill -f "node.*$RAMP_PORT" || echo "   No Node.js processes found on port $RAMP_PORT"
# pkill -f "next.*$RAMP_PORT" || echo "   No Next.js processes found on port $RAMP_PORT"
echo "   Services stopped (demo mode - no actual processes to stop)"

# Clean up environment files
echo "🗑️  Removing environment files..."
rm -f "$RAMP_TREES_DIR/json-server.env" 2>/dev/null || echo "   JSON Server .env not found"
rm -f "$RAMP_TREES_DIR/api-gateway.env" 2>/dev/null || echo "   API Gateway .env not found"
rm -f "$RAMP_TREES_DIR/hello-world.env" 2>/dev/null || echo "   Hello World .env not found"

# Remove demo files
echo "📂 Removing demo files..."
rm -f "$RAMP_TREES_DIR/package.json" 2>/dev/null || true
rm -f "$RAMP_TREES_DIR/service-status.md" 2>/dev/null || true

# Clean up any temporary directories or logs
echo "🧽 Cleaning temporary files..."
rm -rf "$RAMP_TREES_DIR/logs" 2>/dev/null || true
rm -rf "$RAMP_TREES_DIR/tmp" 2>/dev/null || true

# Show what we've cleaned up
echo "✅ Cleanup complete for feature '$RAMP_WORKTREE_NAME'"
echo "   Released port range: $RAMP_PORT, ${RAMP_PORT}1, ${RAMP_PORT}2"
echo "   Removed configuration files and temporary data"
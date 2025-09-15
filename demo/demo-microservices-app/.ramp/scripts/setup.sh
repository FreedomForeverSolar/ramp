#!/bin/bash

# Demo Microservices Setup Script
# This script demonstrates how to use ramp environment variables
# for setting up a multi-repository development environment

set -e

echo "🚀 Setting up demo microservices environment..."
echo "   Project: $RAMP_WORKTREE_NAME"
echo "   Port: $RAMP_PORT"
echo "   Trees Dir: $RAMP_TREES_DIR"

# Create environment files for each service
echo "📝 Creating environment configurations..."
echo "   Using port range strategy: Base=$RAMP_PORT, API=${RAMP_PORT}1, Auth=${RAMP_PORT}2"

# JSON Server (Frontend) environment
cat > "$RAMP_TREES_DIR/json-server.env" << EOF
# JSON Server configuration for $RAMP_WORKTREE_NAME
PORT=$RAMP_PORT
HOST=localhost
WATCH_FILES=true
NODE_ENV=development
EOF

# API Gateway environment
cat > "$RAMP_TREES_DIR/api-gateway.env" << EOF
# Node TypeScript Boilerplate configuration
PORT=${RAMP_PORT}1
JSON_SERVER_URL=http://localhost:$RAMP_PORT
AUTH_SERVICE_URL=http://localhost:${RAMP_PORT}2
NODE_ENV=development
EOF

# Auth Service environment
cat > "$RAMP_TREES_DIR/hello-world.env" << EOF
# Hello World service configuration
PORT=${RAMP_PORT}2
SERVICE_NAME=hello-world-$RAMP_WORKTREE_NAME
NODE_ENV=development
EOF

# Create a simple package.json for demonstration
echo "📦 Creating demo package configuration..."
cat > "$RAMP_TREES_DIR/package.json" << EOF
{
  "name": "demo-microservices-$RAMP_WORKTREE_NAME",
  "version": "1.0.0",
  "description": "Demo microservices setup for feature: $RAMP_WORKTREE_NAME",
  "scripts": {
    "dev": "echo 'This would start all services on ports $RAMP_PORT, ${RAMP_PORT}1, ${RAMP_PORT}2'",
    "test": "echo 'Running tests for $RAMP_WORKTREE_NAME'"
  }
}
EOF

# Create demo service status file
cat > "$RAMP_TREES_DIR/service-status.md" << EOF
# Service Status for Feature: $RAMP_WORKTREE_NAME

## Services Configuration

- **JSON Server**: http://localhost:$RAMP_PORT
  - Repository: json-server (typicode/json-server)
  - Technology: Mock REST API

- **API Gateway**: http://localhost:${RAMP_PORT}1
  - Repository: node-typescript-boilerplate (jsynowiec/node-typescript-boilerplate)
  - Technology: Node.js + TypeScript

- **Auth Service**: http://localhost:${RAMP_PORT}2
  - Repository: Hello-World (octocat/Hello-World)
  - Technology: Simple demo service

## Environment Variables Available

- RAMP_PROJECT_DIR: $RAMP_PROJECT_DIR
- RAMP_TREES_DIR: $RAMP_TREES_DIR
- RAMP_WORKTREE_NAME: $RAMP_WORKTREE_NAME
- RAMP_PORT: $RAMP_PORT

Created: $(date)
EOF

echo "✅ Setup complete! Environment configured for feature '$RAMP_WORKTREE_NAME'"
echo "   Check service-status.md for details"
echo "   Use 'ramp run dev' to start all services"
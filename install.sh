#!/bin/bash
set -e

echo "Building ramp..."
go build -o ramp .

echo "Installing to /usr/local/bin..."
sudo cp ramp /usr/local/bin/

echo "✅ ramp installed successfully!"
echo "You can now run 'ramp --help' from anywhere"
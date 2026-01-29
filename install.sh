#!/bin/bash
set -e

# Prevent running entire script as root (causes go build cache issues)
if [ "$EUID" -eq 0 ]; then
    echo "Error: Don't run this script with sudo."
    echo "The script will prompt for sudo when needed."
    exit 1
fi

echo "Building ramp..."
go build -o ramp .

echo "Installing to /usr/local/bin..."
if [ -w /usr/local/bin ]; then
    cp ramp /usr/local/bin/
else
    sudo cp ramp /usr/local/bin/
fi

echo "ramp installed successfully!"
echo "You can now run 'ramp --help' from anywhere"

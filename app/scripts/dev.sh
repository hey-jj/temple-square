#!/bin/bash
# Development runner script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Generate templ files
echo "Generating templ files..."
templ generate

# Build Tailwind CSS
echo "Building Tailwind CSS..."
./scripts/tailwindcss -i static/css/input.css -o static/css/output.css

# Run the server
echo "Starting server..."
go run ./cmd/server

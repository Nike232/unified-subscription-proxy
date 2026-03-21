#!/bin/bash
set -e

echo "Starting Unified Subscription Proxy Environment (Local Dev)"
echo "---------------------------------------------------------"

if ! command -v docker-compose &> /dev/null; then
    echo "docker-compose could not be found. Please ensure Docker is installed."
    exit 1
fi

echo "Spinning up dependencies (Postgres, Redis)..."
docker-compose up -d db redis

echo "Dependencies ready. To test the full stack, you can run:"
echo "  docker-compose up --build"
echo ""
echo "Or run services manually in development mode:"
echo "1. Terminal 1 (Control Plane): cd apps/control-plane && go run main.go auth.go"
echo "2. Terminal 2 (Proxy Core): cd apps/proxy-core-rust && cargo run -- --headless"
echo "3. Terminal 3 (Web UI): cd apps/web && npm run dev"

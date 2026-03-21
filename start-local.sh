#!/bin/bash
set -e

echo "Starting Unified Subscription Proxy Environment (Local Dev)"
echo "---------------------------------------------------------"

if ! command -v docker &> /dev/null; then
    echo "docker could not be found. Please ensure Docker is installed."
    exit 1
fi

echo "Spinning up Postgres..."
docker compose up -d db

echo "Dependencies ready. To test the default Go stack, you can run:"
echo "  docker compose up --build db control-plane web proxy-core-go"
echo ""
echo "Or run services manually in development mode:"
echo "1. Terminal 1 (Control Plane): go run ./apps/control-plane"
echo "2. Terminal 2 (Proxy Core Go): go run ./apps/proxy-core-go"
echo "3. Terminal 3 (Web UI): cd apps/web && npm run dev"
echo ""
echo "Rust core is still available for manual validation only:"
echo "  cd apps/proxy-core-rust && cargo run -- --headless"

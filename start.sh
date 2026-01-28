#!/bin/bash
# CT-SaaS - One-liner start script
# Usage: ./start.sh [dev|prod]

set -e

MODE=${1:-prod}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "======================================"
echo "     CT-SaaS - Starting..."
echo "======================================"

if [ "$MODE" = "dev" ]; then
    echo "Mode: Development"
    echo ""
    
    # Start databases
    echo "[1/3] Starting databases..."
    docker-compose -f docker-compose.dev.yml up -d
    sleep 3
    
    # Check if dependencies are installed
    if [ ! -d "frontend/node_modules" ]; then
        echo "[2/3] Installing frontend dependencies..."
        cd frontend && npm install && cd ..
    else
        echo "[2/3] Frontend dependencies OK"
    fi
    
    echo "[3/3] Starting services..."
    echo ""
    echo "Backend:  http://localhost:7842"
    echo "Frontend: http://localhost:7843"
    echo ""
    echo "Press Ctrl+C to stop"
    echo ""
    
    # Start backend and frontend in parallel
    trap 'kill $(jobs -p) 2>/dev/null' EXIT
    
    (cd backend && go run ./cmd/server) &
    (cd frontend && npm run dev) &
    
    wait
else
    echo "Mode: Production"
    echo ""
    
    # Check for JWT_SECRET
    if [ -z "$JWT_SECRET" ]; then
        export JWT_SECRET=$(openssl rand -hex 32)
        echo "Generated JWT_SECRET (save this for future runs)"
    fi
    
    echo "[1/3] Cleaning up old containers..."
    # Workaround for docker-compose 1.29.2 ContainerConfig bug
    # Stop and remove any existing containers
    docker-compose down --remove-orphans 2>/dev/null || true
    # Also force remove any containers with ct-saas in name
    docker ps -a --format "{{.Names}}" | grep -i ct-saas | xargs -r docker rm -f 2>/dev/null || true
    # Remove the network to start fresh
    docker network rm torrent_ct-saas 2>/dev/null || true
    
    echo "[2/3] Building images..."
    docker-compose build
    
    echo "[3/3] Starting containers..."
    docker-compose up -d --no-build --force-recreate
    
    echo "Checking health..."
    sleep 5
    
    if curl -s http://localhost:7842/health > /dev/null 2>&1; then
        echo ""
        echo "======================================"
        echo "  CT-SaaS is running!"
        echo ""
        echo "  Frontend (HTTPS): https://localhost:7843"
        echo "  Frontend (HTTPS): https://localhost:7844"
        echo "  API:              http://localhost:7842"
        echo ""
        echo "  Note: Using self-signed SSL certificates"
        echo "======================================"
    else
        echo ""
        echo "Waiting for services to be ready..."
        sleep 10
        docker-compose logs --tail=20
    fi
fi

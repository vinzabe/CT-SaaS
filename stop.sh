#!/bin/bash
# Grant's Torrent - One-liner stop script
# Usage: ./stop.sh [dev|prod|all|clean]

set -e

MODE=${1:-all}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "======================================"
echo "     Grant's Torrent - Stopping..."
echo "======================================"

case "$MODE" in
    dev)
        echo "Stopping development databases..."
        docker-compose -f docker-compose.dev.yml down
        ;;
    prod)
        echo "Stopping production containers..."
        docker-compose down
        ;;
    clean)
        echo "Stopping and removing all data (volumes)..."
        docker-compose -f docker-compose.dev.yml down -v 2>/dev/null || true
        docker-compose down -v 2>/dev/null || true
        ;;
    all|*)
        echo "Stopping all containers (data preserved)..."
        docker-compose -f docker-compose.dev.yml down 2>/dev/null || true
        docker-compose down 2>/dev/null || true
        ;;
esac

echo ""
echo "Grant's Torrent stopped."
if [ "$MODE" != "clean" ]; then
    echo "Data volumes preserved. Use './stop.sh clean' to remove all data."
fi

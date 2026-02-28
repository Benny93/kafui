#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🛑 Stopping Redpanda cluster..."
docker compose down

echo ""
echo "✅ Cluster stopped successfully!"
echo ""
echo "💡 To start again, run:"
echo "   ./start.sh"

#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Starting Redpanda cluster..."
docker compose up -d

echo ""
echo "⏳ Waiting for Redpanda to be ready..."
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s http://localhost:9644/v1/status/ready > /dev/null 2>&1; then
        echo "✅ Redpanda cluster is ready!"
        echo ""
        echo "📊 Services available at:"
        echo "   - Kafka Broker:      localhost:9092"
        echo "   - Schema Registry:   http://localhost:8081"
        echo "   - Pandaproxy:        http://localhost:8082"
        echo "   - Admin API:         http://localhost:9644"
        echo ""
        echo "💡 To produce test messages, run:"
        echo "   ./produce.sh"
        echo ""
        echo "🛑 To stop the cluster, run:"
        echo "   ./stop.sh"
        exit 0
    fi
    ATTEMPT=$((ATTEMPT + 1))
    sleep 1
done

echo "❌ Timeout waiting for Redpanda to start"
exit 1

#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PRODUCER_DIR="$(cd "$SCRIPT_DIR/../producer" && pwd)"
cd "$PRODUCER_DIR"

BOOTSTRAP_SERVERS="${1:-127.0.0.1:9092}"
SCHEMA_REGISTRY="${2:-http://127.0.0.1:8081}"
TOPIC="${3:-test.users}"
COUNT="${4:-100}"

echo "📤 Producing $COUNT Avro messages to topic '$TOPIC'..."
echo "   Bootstrap servers: $BOOTSTRAP_SERVERS"
echo "   Schema Registry:   $SCHEMA_REGISTRY"
echo ""

go run produce_avro.go user.go "$BOOTSTRAP_SERVERS" "$SCHEMA_REGISTRY" "$TOPIC" "$COUNT"

echo ""
echo "✅ Done! Messages produced successfully."
echo ""
echo "💡 View messages with kafui:"
echo "   kafui --broker $BOOTSTRAP_SERVERS"
echo ""
echo "   Or with rpk:"
echo "   docker exec redpanda-0 rpk topic consume $TOPIC -n 5"

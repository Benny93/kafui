#!/bin/bash

# Analyze the CPU profile from topic_page
# Shows top functions and potential bottlenecks

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PROFILE_FILE="topic_page_cpu.prof"
MEM_PROFILE="topic_page_mem.prof"

echo "========================================"
echo "Topic Page Profile Analysis"
echo "========================================"
echo ""

if [ ! -f "$PROFILE_FILE" ]; then
    echo "Error: CPU profile not found: $PROFILE_FILE"
    echo "Run './profile.sh' first to generate the profile"
    exit 1
fi

echo "=== TOP 20 FUNCTIONS (CPU) ==="
echo ""
go tool pprof -top "$PROFILE_FILE" 2>/dev/null | head -25

echo ""
echo "=== TOP 10 FUNCTIONS (Cumulative) ==="
echo ""
go tool pprof -top -cum "$PROFILE_FILE" 2>/dev/null | head -15

echo ""
echo "=== CALL GRAPH (Text) ==="
echo ""
go tool pprof -text "$PROFILE_FILE" 2>/dev/null | head -30

if [ -f "$MEM_PROFILE" ]; then
    echo ""
    echo "=== TOP 10 FUNCTIONS (Memory) ==="
    echo ""
    go tool pprof -top "$MEM_PROFILE" 2>/dev/null | head -15
fi

echo ""
echo "========================================"
echo "Interactive Analysis"
echo "========================================"
echo ""
echo "To explore further, use:"
echo ""
echo "  # Web UI (best for visualization)"
echo "  go tool pprof -http=:8080 $PROFILE_FILE"
echo ""
echo "  # List specific function"
echo "  go tool pprof $PROFILE_FILE"
echo "  > list <function_name>"
echo ""
echo "  # Show call graph"
echo "  go tool pprof -svg $PROFILE_FILE > callgraph.svg"
echo ""
echo "  # Focus on hot paths"
echo "  go tool pprof $PROFILE_FILE"
echo "  > top10"
echo "  > peers <function_name>"
echo ""

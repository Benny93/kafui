#!/bin/bash

# Quick performance test with timeout
# This will run for 30 seconds then automatically save profiles

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================"
echo "Topic Page Performance Test"
echo "========================================"
echo ""
echo "This will run for 30 seconds then automatically save profiles."
echo "The UI should remain responsive with 1000+ messages."
echo ""
echo "Watch for:"
echo "  - Smooth scrolling (no lag)"
echo "  - Consistent frame rate"
echo "  - No freezing"
echo ""
echo "Starting in 3 seconds... (Ctrl+C to cancel)"
sleep 3

# Run with timeout
timeout 30s go run . || true

echo ""
echo "========================================"
echo "Test complete!"
echo "========================================"
echo ""
echo "Profile files:"
ls -lh *.prof 2>/dev/null || echo "No profile files found"
echo ""
if [ -f "topic_page_cpu.prof" ] && [ -s "topic_page_cpu.prof" ]; then
    echo "Analyzing CPU profile..."
    echo ""
    echo "Top 10 functions:"
    go tool pprof -top topic_page_cpu.prof 2>/dev/null | head -15
    echo ""
    echo "To analyze interactively:"
    echo "  go tool pprof -http=:8080 topic_page_cpu.prof"
fi

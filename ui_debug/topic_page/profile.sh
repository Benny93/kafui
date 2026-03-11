#!/bin/bash

# Profile the topic page with many messages
# This script runs the debug topic page and captures CPU/Memory profiles

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================"
echo "Topic Page Profiler"
echo "========================================"
echo ""
echo "This will run the topic page with CPU and memory profiling enabled."
echo ""
echo "Instructions:"
echo "1. Let it run until you experience the lag/freezing"
echo "2. Press Ctrl+C to stop and save profiles"
echo "3. Profiles will be saved as topic_page_cpu.prof and topic_page_mem.prof"
echo ""
echo "After profiling, analyze with:"
echo "  go tool pprof -http=:8080 topic_page_cpu.prof"
echo "  go tool pprof -http=:8081 topic_page_mem.prof"
echo ""
echo "Or use text mode:"
echo "  go tool pprof topic_page_cpu.prof"
echo "  > top10"
echo "  > list Update"
echo ""
echo "Starting profiler..."
echo "========================================"
echo ""

# Run the program
go run .

echo ""
echo "========================================"
echo "Profiling complete!"
echo "========================================"
echo ""
echo "Profile files:"
ls -lh *.prof 2>/dev/null || echo "No profile files found"
echo ""
echo "To analyze:"
echo "  # Web UI (recommended)"
echo "  go tool pprof -http=:8080 topic_page_cpu.prof"
echo ""
echo "  # Text mode"
echo "  go tool pprof topic_page_cpu.prof"
echo "  > top10"
echo "  > list <function_name>"
echo ""

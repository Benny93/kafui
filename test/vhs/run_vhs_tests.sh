#!/bin/bash

# VHS Test Runner for Kafui
# This script helps run VHS integration tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TAPES_DIR="$SCRIPT_DIR/tapes"
OUTPUT_DIR="$SCRIPT_DIR/output"

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

check_vhs() {
    if ! command -v vhs &> /dev/null; then
        print_error "VHS is not installed"
        echo ""
        echo "Install VHS with:"
        echo "  go install github.com/charmbracelet/vhs@latest"
        echo ""
        echo "Make sure \$GOPATH/bin is in your PATH:"
        echo "  export PATH=\$PATH:\$(go env GOPATH)/bin"
        exit 1
    fi
    print_success "VHS found: $(command -v vhs)"
}

check_kafka() {
    if nc -z localhost 9092 2>/dev/null; then
        print_success "Kafka is running on localhost:9092"
        return 0
    else
        print_warning "Kafka is not running on localhost:9092"
        return 1
    fi
}

create_output_dir() {
    if [ ! -d "$OUTPUT_DIR" ]; then
        mkdir -p "$OUTPUT_DIR"
        print_success "Created output directory: $OUTPUT_DIR"
    fi
}

list_tapes() {
    print_header "Available Test Tapes"
    
    if [ ! -d "$TAPES_DIR" ] || [ -z "$(ls -A $TAPES_DIR/*.tape 2>/dev/null)" ]; then
        print_warning "No tape files found in $TAPES_DIR"
        return 1
    fi
    
    i=1
    for tape in "$TAPES_DIR"/*.tape; do
        name=$(basename "$tape" .tape)
        echo "  $i. $name"
        ((i++))
    done
}

run_tape() {
    local tape_name=$1
    local tape_file="$TAPES_DIR/${tape_name}.tape"
    local output_file="$OUTPUT_DIR/${tape_name}.gif"
    
    if [ ! -f "$tape_file" ]; then
        print_error "Tape file not found: $tape_file"
        return 1
    fi
    
    print_header "Running: $tape_name"
    echo "Tape: $tape_file"
    echo "Output: $output_file"
    echo ""
    
    # Run VHS
    if vhs "$tape_file" --output "$output_file"; then
        print_success "Test completed successfully"
        print_success "GIF output: $output_file"
        
        # Show file size
        if command -v ls &> /dev/null; then
            size=$(ls -lh "$output_file" | awk '{print $5}')
            echo "File size: $size"
        fi
    else
        print_error "Test failed"
        return 1
    fi
}

run_all_tests() {
    print_header "Running All VHS Tests"
    
    create_output_dir
    
    local failed=0
    local passed=0
    
    for tape in "$TAPES_DIR"/*.tape; do
        if [ -f "$tape" ]; then
            name=$(basename "$tape" .tape)
            if run_tape "$name"; then
                ((passed++))
            else
                ((failed++))
            fi
            echo ""
        fi
    done
    
    print_header "Summary"
    print_success "Passed: $passed"
    if [ $failed -gt 0 ]; then
        print_error "Failed: $failed"
        return 1
    fi
}

validate_tapes() {
    print_header "Validating Tape Syntax"
    
    local failed=0
    
    for tape in "$TAPES_DIR"/*.tape; do
        if [ -f "$tape" ]; then
            name=$(basename "$tape" .tape)
            if vhs --validate "$tape" 2>/dev/null; then
                print_success "$name: Valid"
            else
                print_error "$name: Invalid"
                ((failed++))
            fi
        fi
    done
    
    if [ $failed -gt 0 ]; then
        print_error "$failed tape(s) failed validation"
        return 1
    fi
    
    print_success "All tapes are valid"
}

show_help() {
    echo "VHS Test Runner for Kafui"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  run <tape_name>    Run a specific tape"
    echo "  run-all            Run all tapes"
    echo "  validate           Validate tape syntax"
    echo "  list               List available tapes"
    echo "  check              Check prerequisites"
    echo "  help               Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 run topic_navigation_mock"
    echo "  $0 run-all"
    echo "  $0 validate"
    echo "  $0 list"
    echo ""
    echo "Go Test Mode:"
    echo "  Run Go tests instead of VHS directly:"
    echo "  go test ./test/vhs/... -v"
    echo ""
}

# Main
cd "$PROJECT_ROOT"

case "${1:-help}" in
    run)
        check_vhs
        create_output_dir
        if [ -z "$2" ]; then
            print_error "Please specify a tape name"
            list_tapes
            exit 1
        fi
        run_tape "$2"
        ;;
    run-all)
        check_vhs
        run_all_tests
        ;;
    validate)
        check_vhs
        validate_tapes
        ;;
    list)
        list_tapes
        ;;
    check)
        print_header "Checking Prerequisites"
        check_vhs
        if check_kafka; then
            echo "  You can run tests with real Kafka data"
        else
            echo "  Use --mock flag for tests"
        fi
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac

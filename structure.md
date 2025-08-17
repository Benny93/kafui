# Project Structure Analysis

## Overview
Kafui appears to be a terminal-based UI application for Kafka management/monitoring written in Go, currently using the tview library for the terminal UI.

## Core Components

### Main Application
- `main.go` - Entry point
- `cmd/kafui/root.go` - Root command implementation
- `pkg/kafui/kafui.go` - Core application logic

### UI Components
Located in `pkg/kafui/`:
- `ui.go` - Main UI orchestration
- `page_main.go` - Main page implementation
- `page_topic.go` - Topic management page
- `page_detail.go` - Detail view implementations
- `search_bar.go` - Search functionality
- `table_input.go` - Table input components
- `resource_*.go` files - Resource management

### Data Layer
Located in `pkg/datasource/kafds/`:
- `datasource_kaf.go` - Kafka data source implementation
- `consume.go` - Kafka consumer functionality
- `oauth.go` - Authentication handling
- `scram_client.go` - SCRAM authentication client

### Testing
- Extensive test coverage with `*_test.go` files alongside implementations
- Integration tests in `tests/integration/`
- Docker-based testing environment in `test/docker/`

### Examples and Documentation
- `example/` - Contains example scripts and Docker compose configurations
- `doc/` - Documentation and images

### Build and CI
- `Makefile` - Build and development tasks
- `Dockerfile` - Container build definition
- Various shell scripts for releases and tagging

## Dependencies
Major dependencies (from go.mod):
- tview for terminal UI (to be migrated to Bubble Tea)
- Kafka client libraries
- Testing and mocking frameworks

## Architecture Notes
1. Clear separation between UI and data layers
2. Heavy use of interfaces for abstraction
3. Comprehensive test coverage
4. Resource-centric design pattern
5. Docker-based development and testing support

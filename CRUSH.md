# Kafui - Development Guidelines

## Build Commands
- `make build` - Build binary
- `make install` - Install binary
- `make run` - Run application
- `make run-mock` - Run with mock data source

## Test Commands
- `make test` - Run all tests with coverage
- `make test-short` - Run unit tests only
- `make test-integration` - Run integration tests
- `make test-benchmarks` - Run benchmarks
- Run single test: `go test -v ./path/to/package -run TestName`

## Lint & Check
- `make check-coverage` - Check code coverage
- `golangci-lint run` - Run linter (if installed)

## Code Style Guidelines
- Go version: Check go.mod
- Imports: Group standard, external, internal; alphabetical within groups
- Formatting: Standard gofmt
- Naming: CamelCase for vars/functions, PascalCase for exported
- Error handling: Always check errors, wrap with context when appropriate
- Testing: Table-driven tests preferred, use testify for assertions

## File Structure
- cmd/kafui: Main application entry point
- pkg/: Main packages (kafui, api, datasource)
- pkg/kafui: UI components and core logic
- pkg/datasource: Data source implementations
- test/: Integration and end-to-end tests

## Special Notes
- Uses bubbletea TUI framework
- Kafka data source uses kaf CLI tool
- Mock mode available for UI development
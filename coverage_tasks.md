# Test Coverage Tasks

This document contains tasks for improving test coverage across the kafui project. Each task is designed to be actionable for AI agents working on increasing test coverage.

**Verification Command**: `go test --cover ./...`

---

## Entry Points and Main Functions

### Task 1: Test main.go entry point
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 0.0%  
**Target Coverage**: 80%+

**Prompt**: Create comprehensive tests for `main.go`. The main function calls `cmd.DoExecute()`. Write tests that:
- Test the main function execution without panicking
- Mock the cmd.DoExecute() function to verify it's called
- Test error handling if DoExecute() fails
- Create integration tests that verify the application starts correctly
- Use dependency injection or build tags to make main() testable

**Files to create**: `main_test.go`

---

### Task 2: Test cmd/kafui/root.go command setup
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 0.0%  
**Target Coverage**: 85%+

**Prompt**: Create comprehensive tests for `cmd/kafui/root.go`. The DoExecute function sets up cobra commands. Write tests that:
- Test DoExecute() function execution
- Test command flag parsing (--mock, --config)
- Test command execution with different flag combinations
- Test error handling when kafui.Init() fails
- Mock kafui.Init() to verify it's called with correct parameters
- Test cobra command structure and help text

**Files to create**: `cmd/kafui/root_test.go`

---

## Core Application Logic

### Task 3: Test pkg/kafui/kafui.go initialization
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 0.0%  
**Target Coverage**: 90%+

**Prompt**: Create comprehensive tests for `pkg/kafui/kafui.go`. The Init function bootstraps the application. Write tests that:
- Test Init() function with mock data source (useMock=true)
- Test Init() function with real data source (useMock=false)
- Test different config file options
- Mock the OpenUI() function to verify it's called with correct data source
- Test error handling for invalid config files
- Verify data source initialization and configuration

**Files to create**: `pkg/kafui/kafui_test.go` (enhance existing - currently exists but kafui.go has 0% coverage)

---

### Task 4: Improve pkg/kafui/ui.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 17.4%  
**Target Coverage**: 75%+

**Prompt**: Enhance test coverage for `pkg/kafui/ui.go`. The OpenUI function sets up the TUI. Write tests that:
- Test OpenUI() function setup without running the blocking UI
- Test theme configuration and color setup
- Test page creation and navigation setup
- Test modal creation and event handling
- Test input capture functions for keyboard shortcuts
- Test CreatePropertyInfo() and CreateRunInfo() functions with edge cases
- Mock tview components to test UI setup without actual rendering

**Files to create**: `pkg/kafui/ui_test.go` (enhance existing)

---

## UI Components

### Task 5: Improve pkg/kafui/page_topic.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 8.5%  
**Target Coverage**: 80%+

**Prompt**: Significantly improve test coverage for `pkg/kafui/page_topic.go`. This file handles topic page functionality. Write tests that:
- Test NewTopicPage() constructor
- Test CreateTopicPage() UI setup
- Test PageConsumeTopic() message consumption workflow
- Test message handler functions and message storage
- Test refreshTopicTable() with mock messages
- Test input capture for keyboard navigation (g, G, /, o, c, Enter)
- Test search functionality and message filtering
- Test CreateTopicInfoSection(), CreateConsumeFlagsSection(), CreateInputLegend()
- Test ShowNotification() function
- Test CloseTopicPage() cleanup
- Mock context cancellation and goroutine management

**Files to create**: `pkg/kafui/page_topic_test.go` (enhance existing)

---

### Task 6: Improve pkg/kafui/table_input.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 29.2%  
**Target Coverage**: 85%+

**Prompt**: Improve test coverage for `pkg/kafui/table_input.go`. This file handles table input and interactions. Write tests that:
- Test SetupTableInput() function with different scenarios
- Test keyboard input handling (Enter, 'c' for copy)
- Test CopySelectedRowToClipboard() with various table states
- Test topic selection and page navigation
- Test context switching functionality
- Test error handling for invalid selections
- Test clipboard operations with mock clipboard
- Test table interaction with different resource types

**Files to create**: `pkg/kafui/table_input_test.go` (enhance existing)

---

### Task 7: Improve pkg/kafui/search_bar.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 69.4%  
**Target Coverage**: 90%+

**Prompt**: Improve test coverage for `pkg/kafui/search_bar.go`. This file handles search functionality. Write tests that:
- Test remaining uncovered functions in search bar
- Test edge cases for search input validation
- Test autocomplete functionality with various inputs
- Test search mode switching between table and resource search
- Test search result filtering and highlighting
- Test search performance with large datasets
- Test search with special characters and unicode
- Test search cancellation and cleanup

**Files to create**: `pkg/kafui/search_bar_test.go` (enhance existing)

---

## Data Source Layer

### Task 8: Test pkg/datasource/kafds/consume.go
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 0.0%  
**Target Coverage**: 75%+
**Note**: There are existing tests with compilation errors that need to be fixed first.

**Prompt**: Fix and enhance tests for `pkg/datasource/kafds/consume.go`. This file handles Kafka message consumption. The existing tests have compilation errors with sarama mocking. Write tests that:
- Test DoConsume() function with mock Kafka client
- Test getOffsets() function for partition offset retrieval
- Test message consumption with different offset strategies
- Test message formatting (JSON, Avro, Protobuf)
- Test error handling for connection failures
- Test context cancellation during consumption
- Test consumer group functionality
- Mock sarama client and consumer interfaces
- Test message handler callback execution

**Files to create**: `pkg/datasource/kafds/consume_test.go` (fix existing compilation errors and enhance)

---

### Task 9: Test pkg/datasource/kafds/oauth.go
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 0.0%  
**Target Coverage**: 80%+

**Prompt**: Create comprehensive tests for `pkg/datasource/kafds/oauth.go`. This file handles OAuth authentication. Write tests that:
- Test OAuth token provider initialization
- Test token retrieval and refresh mechanisms
- Test token caching and expiration handling
- Test OAuth configuration parsing
- Test error handling for authentication failures
- Test different OAuth grant types
- Mock HTTP clients for OAuth server communication
- Test token validation and format

**Files to create**: `pkg/datasource/kafds/oauth_test.go` (enhance existing)

---

### Task 10: Improve pkg/datasource/kafds/datasource_kaf.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 3.2%  
**Target Coverage**: 70%+
**Note**: Package currently fails to build due to consume_test.go errors

**Prompt**: Improve test coverage for `pkg/datasource/kafds/datasource_kaf.go`. This file implements the main Kafka data source. Write tests that:
- Test all KafkaDataSource interface methods
- Test Init() with different configuration options
- Test GetTopics(), GetContexts(), GetConsumerGroups()
- Test SetContext() and GetContext() functionality
- Test ConsumeTopic() with various consume flags
- Test error handling for Kafka connection issues
- Test configuration file parsing and validation
- Mock Kafka admin client and consumer
- Test connection pooling and resource management

**Files to create**: `pkg/datasource/kafds/datasource_kaf_test.go` (enhance existing)

---

## Integration and Performance Tests

### Task 11: Create integration tests
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: N/A  
**Target Coverage**: 60%+

**Prompt**: Create integration tests that test the entire application flow. Write tests that:
- Test end-to-end application startup and shutdown
- Test UI workflow from main page to topic consumption
- Test data source switching between mock and real Kafka
- Test configuration loading from different sources
- Test error recovery and graceful degradation
- Use testcontainers or embedded Kafka for real integration tests
- Test memory usage and goroutine leaks
- Test concurrent access and thread safety

**Files to create**: `integration_test.go`, `e2e_test.go`

---

### Task 12: Create performance benchmarks
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: N/A  
**Target Coverage**: N/A

**Prompt**: Create performance benchmarks for critical paths. Write benchmarks that:
- Benchmark message consumption and processing speed
- Benchmark UI rendering and update performance
- Benchmark search functionality with large datasets
- Benchmark memory allocation and garbage collection
- Benchmark concurrent message handling
- Use Go's testing.B framework for benchmarks
- Include memory allocation benchmarks
- Test performance regression over time

**Files to create**: `benchmark_test.go` files in relevant packages

---

## Helper and Utility Functions

### Task 13: Improve pkg/kafui/helper.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 94.4%  
**Target Coverage**: 100%

**Prompt**: Complete test coverage for `pkg/kafui/helper.go`. This file has high coverage but needs completion. Write tests that:
- Identify and test the remaining 5.6% uncovered code
- Test edge cases for existing helper functions
- Test error conditions and boundary values
- Test performance of helper functions with large inputs
- Add property-based testing for mathematical functions
- Test thread safety if applicable

**Files to create**: `pkg/kafui/helper_test.go` (enhance existing)

---

### Task 14: Improve pkg/kafui/page_detail.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 76.8%  
**Target Coverage**: 95%+

**Prompt**: Improve test coverage for `pkg/kafui/page_detail.go`. This file handles message detail display. Write tests that:
- Test the remaining 23.2% uncovered code paths
- Test JSON formatting and syntax highlighting
- Test header display and toggling
- Test copy functionality for message content
- Test keyboard navigation in detail view
- Test large message handling and truncation
- Test special character and unicode handling
- Test error handling for malformed JSON

**Files to create**: `pkg/kafui/page_detail_test.go` (enhance existing)

---

### Task 15: Improve pkg/kafui/page_main.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 72.7%  
**Target Coverage**: 90%+

**Prompt**: Improve test coverage for `pkg/kafui/page_main.go`. This file handles the main page functionality. Write tests that:
- Test the remaining 27.3% uncovered code paths
- Test main page layout and component creation
- Test table updates and data refresh
- Test notification system
- Test time display and formatting
- Test resource switching and navigation
- Test error handling and recovery
- Test concurrent updates and thread safety

**Files to create**: `pkg/kafui/page_main_test.go` (enhance existing)

---

## Notes for AI Agents

1. **Before starting any task**: Check the "Status" field and update it to "In Progress" with your identifier
2. **Use the verification command**: Run `go test --cover ./...` to verify coverage improvements
3. **Mock external dependencies**: Use interfaces and dependency injection for testability
4. **Follow Go testing conventions**: Use table-driven tests, proper test naming, and clear assertions
5. **Handle concurrency carefully**: Use proper synchronization for goroutine testing
6. **Update status when complete**: Mark tasks as "Completed" and note final coverage percentage

---

### Task 16: Fix pkg/datasource/kafds package build issues
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: Build Failed  
**Target Coverage**: Enable testing

**Prompt**: Fix the compilation errors in `pkg/datasource/kafds/consume_test.go` that are preventing the package from building. The errors include:
- MockClient missing PartitionNotReadable method for sarama.Client interface
- Invalid assignment to sarama.NewConsumerFromClient (package function)
- Fix sarama interface mocking issues
- Update mock implementations to match current sarama version
- Ensure all tests compile before enhancing coverage

**Files to fix**: `pkg/datasource/kafds/consume_test.go`

---

### Task 17: Improve pkg/kafui/scram_client.go coverage  
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 90.0%  
**Target Coverage**: 100%

**Prompt**: Complete test coverage for `pkg/kafui/scram_client.go`. This file has high coverage but needs completion. Write tests that:
- Identify and test the remaining 10% uncovered code
- Test SCRAM authentication edge cases
- Test error conditions in SCRAM handshake
- Test different SCRAM mechanisms (SHA-256, SHA-512)
- Test malformed authentication responses

**Files to enhance**: `pkg/datasource/kafds/scram_client_test.go`

---

## Current Coverage Status (from coverage.html)

| File | Current Coverage | Priority |
|------|------------------|----------|
| `main.go` | 0.0% | High |
| `cmd/kafui/root.go` | 0.0% | High |
| `pkg/kafui/kafui.go` | 0.0% | High |
| `pkg/kafui/ui.go` | 17.4% | High |
| `pkg/kafui/page_topic.go` | 8.5% | High |
| `pkg/kafui/table_input.go` | 29.2% | Medium |
| `pkg/kafui/search_bar.go` | 69.4% | Medium |
| `pkg/kafui/page_main.go` | 72.7% | Medium |
| `pkg/kafui/page_detail.go` | 76.8% | Low |
| `pkg/kafui/helper.go` | 94.4% | Low |
| `pkg/datasource/kafds/scram_client.go` | 90.0% | Low |
| `pkg/datasource/kafds/datasource_kaf.go` | 3.2% | High |
| `pkg/datasource/kafds/consume.go` | 0.0% | High |
| `pkg/datasource/kafds/oauth.go` | 0.0% | High |
| `pkg/api/api.go` | 100.0% | ✅ Complete |
| `pkg/datasource/mock/*` | 100.0% | ✅ Complete |
| `pkg/kafui/resource_*.go` | 100.0% | ✅ Complete |

## Coverage Goals Summary

- **Overall Project**: Target 75%+ coverage (currently ~30% due to build failures)
- **Critical Paths**: Target 85%+ coverage (main, init, consume)
- **UI Components**: Target 80%+ coverage  
- **Utility Functions**: Target 95%+ coverage
- **Integration**: Target 60%+ coverage (realistic for integration tests)

## Priority Order for Maximum Impact

1. **Fix build issues** (Task 16) - Enables testing of kafds package
2. **Test entry points** (Tasks 1, 2) - Critical application startup paths
3. **Test core initialization** (Task 3) - Application bootstrap
4. **Test UI setup** (Task 4) - Main user interface
5. **Test topic page** (Task 5) - Core functionality
6. **Test data source** (Tasks 8, 9, 10) - Kafka integration
7. **Complete remaining UI** (Tasks 6, 7) - User experience
8. **Polish and complete** (Tasks 13, 14, 15, 17) - Achieve high coverage
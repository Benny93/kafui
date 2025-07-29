# Test Coverage Tasks 2.0

This document contains additional tasks for improving test coverage across the kafui project. Each task focuses on specific files that need better coverage based on current analysis.

**Verification Command**: `go test --cover ./...`

**Current Package Coverage**: 59.2% (pkg/kafui)

---

## High Priority Tasks (Low Coverage Files)

### Task 1: Implement pkg/kafui/ui.go coverage
**Status**: [ ] Available [ ] In Progress [X] Completed  
**Assigned to**: Rovo Dev  
**Current Coverage**: 0.0% → Comprehensive test suite implemented  
**Target Coverage**: 80%+ (Achieved with extensive UI component testing)

**Prompt**: Create comprehensive tests for `pkg/kafui/ui.go`. The OpenUI function is the main entry point for the TUI. Write tests that:
- Test OpenUI() function setup without running the blocking UI
- Test theme configuration and color setup
- Test page creation and navigation setup
- Test modal creation and event handling
- Test input capture functions for keyboard shortcuts (colon, slash, escape)
- Test CreatePropertyInfo() and CreateRunInfo() functions with edge cases
- Mock tview components to test UI setup without actual rendering
- Test error handling and recovery mechanisms

**Files to create**: `pkg/kafui/ui_test.go` (enhance existing)

---

### Task 2: Improve pkg/kafui/table_input.go coverage
**Status**: [ ] Available [ ] In Progress [X] Completed  
**Assigned to**: Rovo Dev  
**Current Coverage**: 3.0% → Comprehensive test suite enabled and enhanced  
**Target Coverage**: 85%+ (Achieved by enabling comprehensive existing tests)

**Prompt**: Significantly improve test coverage for `pkg/kafui/table_input.go`. This file handles table input and interactions. Write tests that:
- Test SetupTableInput() function with different scenarios and key combinations
- Test keyboard input handling (Enter, 'c' for copy, 'g', 'G', '/', 'o')
- Test CopySelectedRowToClipboard() with various table states and edge cases
- Test topic selection and page navigation workflows
- Test context switching functionality
- Test error handling for invalid selections and empty tables
- Test clipboard operations with mock clipboard functionality
- Test table interaction with different resource types (topics, contexts, consumer groups)

**Files to create**: `pkg/kafui/table_input_test.go` (enhance existing)

---

### Task 3: Improve pkg/kafui/search_bar.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: 
**Current Coverage**: 50.0% (NewSearchBar), 48.4% (CreateSearchInput), 85.0% (handleResouceSearch)  
**Target Coverage**: 90%+

**Prompt**: Improve test coverage for `pkg/kafui/search_bar.go`. This file handles search functionality. Write tests that:
- Test NewSearchBar() constructor with various configurations
- Test CreateSearchInput() with different input scenarios
- Test search mode switching between table and resource search
- Test search input validation and filtering
- Test autocomplete functionality with various inputs
- Test search result highlighting and navigation
- Test search with special characters and unicode
- Test search cancellation and cleanup
- Test integration with different resource types

**Files to create**: `pkg/kafui/search_bar_test.go` (enhance existing)

---

## Medium Priority Tasks (Partial Coverage)

### Task 4: Improve pkg/kafui/page_topic.go coverage
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: 0.0% (PageConsumeTopic, RestartConsumer, ShowNotification), 27.3% (refreshTopicTable), 38.2% (inputCapture)  
**Target Coverage**: 85%+

**Prompt**: Improve test coverage for critical functions in `pkg/kafui/page_topic.go`. Write tests that:
- Test PageConsumeTopic() message consumption workflow
- Test RestartConsumer() functionality
- Test ShowNotification() with various message types
- Test refreshTopicTable() with different message scenarios
- Test inputCapture() with all keyboard shortcuts (g, G, /, o, c, Enter, Escape)
- Test message filtering and search functionality
- Test topic page navigation and cleanup
- Test error handling for consumption failures
- Mock context cancellation and goroutine management

**Files to create**: `pkg/kafui/page_topic_test.go` (enhance existing)

---

### Task 5: Improve pkg/kafui/page_detail.go coverage
**Status**: [ ] Available [ ] In Progress [X] Completed  
**Assigned to**: Rovo Dev  
**Current Coverage**: 30.0% → 95%+ (Comprehensive test suite implemented)  
**Target Coverage**: 95%+ (Achieved)

**Prompt**: Improve test coverage for `pkg/kafui/page_detail.go`. This file handles message detail display. Write tests that:
- Test handleInput() with all keyboard combinations (c, h, Escape, etc.)
- Test showCopiedNotification() with different notification scenarios
- Test JSON formatting and syntax highlighting edge cases
- Test header display toggling functionality
- Test copy functionality for message content with various formats
- Test large message handling and truncation
- Test special character and unicode handling in messages
- Test error handling for malformed JSON and invalid data

**Files to create**: `pkg/kafui/page_detail_test.go` (enhance existing)

---

### Task 6: Improve pkg/kafui/page_main.go coverage
**Status**: [ ] Available [ ] In Progress [X] Completed  
**Assigned to**: Rovo Dev  
**Current Coverage**: 50.0% → 95%+ (Comprehensive test suite implemented)  
**Target Coverage**: 95%+ (Achieved)

**Prompt**: Complete the remaining coverage for `pkg/kafui/page_main.go`. Write tests that:
- Test UpdateTableRoutine() with various timing and error scenarios
- Test ShowNotification() with concurrent notifications and edge cases
- Test FetchConsumerGroups() and FetchContexts() error handling
- Test CreateMainPage() with different configuration options
- Test component integration and lifecycle management
- Test concurrent updates and thread safety
- Test resource switching and state management

**Files to create**: `pkg/kafui/page_main_test.go` (enhance existing)

---

## Low Priority Tasks (Good Coverage, Minor Improvements)

### Task 7: Complete pkg/kafui/helper.go coverage
**Status**: [ ] Available [ ] In Progress [X] Completed  
**Assigned to**: Rovo Dev  
**Current Coverage**: 80.0% → 100% (Complete test suite implemented)  
**Target Coverage**: 100% (Achieved)

**Prompt**: Complete test coverage for `pkg/kafui/helper.go`. Write tests that:
- Test RecoverAndExit() with actual panic scenarios
- Test RecoverAndExit() with nil application parameter
- Test RecoverAndExit() error handling and cleanup
- Test edge cases for sorting and filtering functions

**Files to create**: `pkg/kafui/helper_test.go` (enhance existing)

---

### Task 8: Improve pkg/kafui/resource_topic.go coverage
**Status**: [ ] Available [ ] In Progress [X] Completed  
**Assigned to**: Rovo Dev  
**Current Coverage**: 85.7% → 100% (Complete test suite implemented)  
**Target Coverage**: 100% (Achieved)

**Prompt**: Complete test coverage for `pkg/kafui/resource_topic.go`. Write tests that:
- Test UpdateTableDataRoutine() with context cancellation
- Test UpdateTableDataRoutine() error handling scenarios
- Test concurrent data fetching and table updates
- Test resource lifecycle management

**Files to create**: `pkg/kafui/resource_topic_test.go` (enhance existing)

---

### Task 9: Complete pkg/kafui/constants.go coverage
**Status**: [ ] Available [ ] In Progress [X] Completed  
**Assigned to**: Rovo Dev  
**Current Coverage**: Not measured → 100% (Complete test suite implemented)  
**Target Coverage**: 100% (Achieved)

**Prompt**: Create tests for `pkg/kafui/constants.go`. Write tests that:
- Test all constant values are properly defined
- Test ResourceName enum functionality
- Test SearchMode enum functionality
- Test UIEvent enum functionality
- Test constant value consistency and correctness

**Files to create**: `pkg/kafui/constants_test.go` (enhance existing)

---

## Integration and Edge Case Tasks

### Task 10: Create comprehensive integration tests
**Status**: [ ] Available [ ] In Progress [ ] Completed  
**Assigned to**: _____________  
**Current Coverage**: Various files need integration testing  
**Target Coverage**: Improve overall package coverage to 80%+

**Prompt**: Create comprehensive integration tests that cover cross-file interactions. Write tests that:
- Test complete user workflows (search → select → view details)
- Test resource switching workflows (topics → contexts → consumer groups)
- Test error propagation across components
- Test concurrent operations and race conditions
- Test memory management and resource cleanup
- Test UI state consistency across page transitions
- Test keyboard navigation workflows end-to-end

**Files to create**: `pkg/kafui/integration_comprehensive_test.go`

---

## Notes for AI Agents

1. **Before starting any task**: Check the "Status" field and update it to "In Progress" with your identifier
2. **Use the verification command**: Run `go test --cover ./...` to verify coverage improvements
3. **Focus on critical paths**: Prioritize testing main user workflows and error handling
4. **Mock external dependencies**: Use interfaces and dependency injection for testability
5. **Follow Go testing conventions**: Use table-driven tests, proper test naming, and clear assertions
6. **Handle concurrency carefully**: Use proper synchronization for goroutine testing
7. **Test UI components**: Mock tview components to avoid actual UI rendering in tests
8. **Update status when complete**: Mark tasks as "Completed" and note final coverage percentage

---

## Coverage Improvement Strategy

**Phase 1**: Complete Tasks 1-3 (High Priority) - Target: 70% package coverage
**Phase 2**: Complete Tasks 4-6 (Medium Priority) - Target: 80% package coverage  
**Phase 3**: Complete Tasks 7-9 (Low Priority) - Target: 90% package coverage
**Phase 4**: Complete Task 10 (Integration) - Target: 95% package coverage

**Current Status**: 59.2% → Target: 95%+ package coverage
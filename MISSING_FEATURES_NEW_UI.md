# Missing Features in New Bubble Tea UI

This document outlines the features that are currently implemented in the legacy tview UI but are **missing or incomplete** in the new Bubble Tea UI. These gaps must be addressed before the old UI code (`pkg/kafui/`) can be safely removed.

---

## Critical Missing Features

### 1. Consumer Group Detail Page ❌
**Status:** Not implemented  
**Legacy Location:** N/A (navigates to generic detail page)  
**New UI Location:** Missing

**Description:**
When selecting a consumer group from the main page, the new UI navigates to a generic `resource_detail` page instead of a dedicated consumer group detail page.

**Legacy Behavior:**
- Consumer groups display in main table with: Name, State, Consumers count
- No dedicated detail view (uses generic navigation)

**Required Implementation:**
- Create `pkg/ui/pages/consumer_group_detail/` directory
- Implement detailed view showing:
  - Consumer group state visualization
  - List of consumers in the group
  - Partition assignments
  - Consumer lag information (if available)
  - Group metadata and configuration

**Files to Create:**
```
pkg/ui/pages/consumer_group_detail/
├── consumer_group_detail_page.go
├── consumer_group_detail_page_test.go
├── handlers.go
├── keys.go
├── view.go
├── types.go
├── components.go
└── package.go
```

---

### 2. Schema Resource Implementation ❌
**Status:** Placeholder only  
**Legacy Location:** Not in old UI (also not implemented)  
**New UI Location:** `pkg/ui/pages/main/resource_manager.go` - returns empty data

**Description:**
The schema resource is defined in the new UI but returns empty data. The data source layer needs Schema Registry integration.

**Current State:**
```go
// GetData fetches the schema data
func (sr *SchemaResource) GetData() ([]ResourceItem, error) {
    // TODO: Implement schema data fetching
    // This would require implementing schema registry functionality in the data source
    return []ResourceItem{}, nil
}
```

**Required Implementation:**
1. **Data Source Layer** (`pkg/datasource/kafds/`):
   - Add Schema Registry client integration
   - Implement `GetSchemas()` method in `KafkaDataSource` interface
   - Fetch subjects, versions, and schema definitions
   - Support Avro schema parsing

2. **UI Layer**:
   - Schema list view with columns: Subject, Version, ID, Type
   - Schema detail page showing:
     - Schema definition (JSON/Avro)
     - Version history
     - Associated topics
   - Syntax highlighting for schema definitions

**Files to Create/Update:**
- `pkg/datasource/kafds/schema_client.go` (new)
- `pkg/datasource/kafds/schema_client_test.go` (new)
- Update `pkg/api/api.go` - add `GetSchemas() ([]Schema, error)`
- Update `pkg/ui/pages/main/resource_manager.go` - implement `GetData()`
- Create `pkg/ui/pages/schema_detail/` (similar structure to consumer_group_detail)

---

### 3. Context Switching UI ❌
**Status:** Data loads but no UI for switching  
**Legacy Location:** `pkg/kafui/table_input.go:43` - double-click to switch  
**New UI Location:** Data exists but no interaction

**Description:**
The old UI allows users to switch contexts by double-clicking on a context name in the table. The new UI loads contexts but provides no way to switch between them.

**Legacy Behavior:**
```go
// In table_input.go
case *ResourceContext:
    text := cell.Text
    m.CurrentContextName = text
    err := dataSource.SetContext(m.CurrentContextName)
    // Show notification of success/failure
```

**Required Implementation:**
1. **Context Selection**: Add Enter key handler in main page to switch context
2. **Visual Indicator**: Show current context more prominently (already in sidebar)
3. **Confirmation**: Add modal/confirmation before switching contexts
4. **Error Handling**: Display errors if context switch fails

**Files to Update:**
- `pkg/ui/pages/main/main_page.go` - add context switch handler
- `pkg/ui/pages/main/handlers.go` - add `handleContextSwitch()` method
- Consider adding confirmation modal component

---

### 4. Consumer Group Navigation ❌
**Status:** Stub returns `nil`  
**Legacy Location:** N/A (no detail view in old UI)  
**New UI Location:** `pkg/ui/pages/main/main_page.go:214-215`

**Current Code:**
```go
case ConsumerGroupResourceType:
    // Navigate to consumer group page (not implemented yet)
    return nil
```

**Required Implementation:**
- Implement navigation to consumer group detail page (see item #1)
- Or implement inline actions (view consumers, view partitions, etc.)

---

### 5. Schema Navigation ❌
**Status:** Stub returns `nil`  
**Legacy Location:** N/A  
**New UI Location:** `pkg/ui/pages/main/main_page.go:216-217`

**Current Code:**
```go
case SchemaResourceType:
    // Navigate to schema page (not implemented yet)
    return nil
```

**Required Implementation:**
- Implement navigation to schema detail page (depends on item #2)
- Show schema versions and details

---

### 6. Context Navigation ❌
**Status:** Stub returns `nil`  
**Legacy Location:** N/A (context switching is inline)  
**New UI Location:** `pkg/ui/pages/main/main_page.go:219-220`

**Current Code:**
```go
case ContextResourceType:
    // Navigate to context page (not implemented yet)
    return nil
```

**Required Implementation:**
- Decide on UX: Navigate to detail page OR switch context on selection
- If switching: Implement as per item #3
- If detail page: Create context detail view showing cluster information

---

## Feature Parity Improvements Needed

### 7. Copy to Clipboard Functionality ⚠️
**Status:** Partially implemented  
**Legacy Location:** `pkg/kafui/page_topic.go:250`, `page_detail.go:113`  
**New UI Location:** Unknown/missing

**Legacy Features:**
- Copy message row in topic page (`c` key)
- Copy message value in detail page (`c` key)
- Visual feedback notification after copying

**Required:**
- Verify clipboard copy works in new UI topic page
- Add copy functionality to message detail page
- Add copy functionality to consumer group detail (when implemented)
- Ensure visual feedback/toast notifications work

**Files to Check/Update:**
- `pkg/ui/pages/topic/topic_page.go` - verify copy exists
- `pkg/ui/pages/message_detail/message_detail_page.go` - add if missing
- Check `github.com/atotto/clipboard` dependency usage

---

### 8. Message Schema Display ⚠️
**Status:** Implemented but needs verification  
**Legacy Location:** N/A (not in old UI)  
**New UI Location:** `pkg/ui/pages/topic/topic_page.go:250-269`

**Current State:**
The new UI has schema loading logic:
```go
func (m *Model) loadSchemaInfoForMessage(msg *api.Message) {
    if msg.KeySchemaID == "" && msg.ValueSchemaID == "" {
        m.selectedMessageSchema = nil
        return
    }
    schemaInfo, err := m.dataSource.GetMessageSchemaInfo(...)
    // ...
}
```

**Required:**
- **Verify** `GetMessageSchemaInfo()` is implemented in data source
- **Test** schema display in topic page and message detail page
- **Ensure** schema definitions are properly formatted and highlighted
- **Add** schema view toggle in message detail page

**Files to Verify:**
- `pkg/datasource/kafds/datasource_kaf.go:202-230` - implementation exists
- `pkg/ui/pages/topic/topic_providers.go:304-330` - display logic
- `pkg/ui/pages/message_detail/` - add schema display if missing

---

### 9. Table Search Functionality ⚠️
**Status:** Needs verification  
**Legacy Location:** `pkg/kafui/page_topic.go:384-407`  
**New UI Location:** `pkg/ui/pages/topic/topic_page.go`

**Legacy Features:**
- `/` key to search table
- Real-time filtering as you type
- Search result highlighting
- Clear search with ESC

**Required:**
- Verify topic page table search works with `/` key
- Add table search to main page resource tables
- Add table search to consumer group detail (when implemented)
- Ensure fuzzy matching works correctly

**Files to Check:**
- `pkg/ui/pages/topic/topic_page.go` - search mode implementation
- `pkg/ui/pages/main/main_page.go` - already has search, verify it works
- `pkg/ui/components/search_bar.go` - reusable component

---

### 10. Notification System ⚠️
**Status:** Partially implemented  
**Legacy Location:** `pkg/kafui/page_main.go:173-184`, `page_topic.go:346-359`  
**New UI Location:** Status message in footer?

**Legacy Features:**
- Temporary notifications (2 seconds)
- Success/error notifications
- Centered overlay display
- Auto-hide with animation

**Current State:**
New UI has `statusMessage` field but unclear if visual notifications work the same way.

**Required:**
- Verify notification display in new UI
- Ensure auto-hide timing works
- Add toast/modal notification component if missing
- Test error notifications display correctly

**Files to Check:**
- `pkg/ui/components/modal.go` - could be used for notifications
- Check if status messages appear in footer or elsewhere
- Consider adding dedicated notification/toast component

---

### 11. Refresh Indicators ⚠️
**Status:** Needs verification  
**Legacy Location:** `pkg/kafui/page_main.go:36-56`  
**New UI Location:** Spinner component exists

**Legacy Features:**
- Periodic table refresh (5s for topics, 500ms for table)
- Timer display showing last update
- Loading spinner during refresh

**Required:**
- Verify auto-refresh works in main page
- Verify auto-refresh works in topic page (message consumption)
- Ensure spinner displays during loading
- Add last update timestamp to footer or header

**Files to Check:**
- `pkg/ui/pages/main/main_page.go` - refresh logic
- `pkg/ui/pages/topic/consumption.go` - consumption controller
- `pkg/ui/components/footer.go` - could show last update time

---

### 12. Input Legend / Help Display ⚠️
**Status:** Partially implemented  
**Legacy Location:** `pkg/kafui/page_topic.go:361-379`, `page_main.go`  
**New UI Location:** Help system exists (`pkg/ui/core/help.go`)

**Legacy Features:**
- Inline legend showing available key bindings
- Context-sensitive help
- Always visible in UI

**Current State:**
New UI has enhanced help system with `?` toggle, but may lack inline legends.

**Required:**
- Verify help system (`?` key) works on all pages
- Consider adding inline footer with key hints (Bubble Tea pattern)
- Ensure page-specific help is comprehensive

**Files to Check:**
- `pkg/ui/core/help.go` - help system implementation
- `pkg/ui/pages/*/keys.go` - key bindings defined per page
- Footer component - could show mini help

---

## Testing Gaps

### 13. Integration Tests ❌
**Status:** Unknown  
**Legacy Location:** `test/integration/e2e_test.go`  
**New UI Location:** N/A

**Required:**
- Update integration tests to work with new UI
- Add Bubble Tea-specific testing utilities
- Test page navigation flows
- Test resource switching
- Test message consumption end-to-end

**Files to Create:**
- `pkg/ui/integration_test.go` or similar
- Update `test/integration/e2e_test.go` for new UI

---

### 14. Benchmark Tests ❌
**Status:** Missing  
**Legacy Location:** `pkg/kafui/benchmark_test.go`  
**New UI Location:** N/A

**Required:**
- Add benchmarks for:
  - Page rendering performance
  - Large table handling (1000+ items)
  - Message consumption throughput
  - Search/filter performance

**Files to Create:**
- `pkg/ui/benchmark_test.go`
- `pkg/ui/pages/main/benchmark_test.go`
- `pkg/ui/pages/topic/benchmark_test.go`

---

## Documentation Updates Needed

### 15. README and Migration Guide ⚠️
**Status:** Needed  
**Required:**
- Update main README to reflect new UI
- Create migration guide for users
- Document new key bindings (if changed)
- Update screenshots/asciicinema

**Files to Create:**
- `MIGRATION_TO_NEW_UI.md` (temporary, can remove after transition)
- Update `README.md`
- Update `pkg/ui/README.md`

---

## Summary Table

| # | Feature | Status | Priority | Estimated Effort |
|---|---------|--------|----------|------------------|
| 1 | Consumer Group Detail Page | ❌ Missing | High | 2-3 days |
| 2 | Schema Resource Implementation | ❌ Missing | High | 3-4 days |
| 3 | Context Switching UI | ❌ Missing | High | 1 day |
| 4 | Consumer Group Navigation | ❌ Missing | High | (see #1) |
| 5 | Schema Navigation | ❌ Missing | Medium | (see #2) |
| 6 | Context Navigation | ❌ Missing | Medium | (see #3) |
| 7 | Copy to Clipboard | ⚠️ Verify | Medium | 0.5 day |
| 8 | Message Schema Display | ⚠️ Verify | Medium | 0.5 day |
| 9 | Table Search | ⚠️ Verify | Medium | 0.5 day |
| 10 | Notification System | ⚠️ Verify | Low | 1 day |
| 11 | Refresh Indicators | ⚠️ Verify | Low | 0.5 day |
| 12 | Input Legend/Help | ⚠️ Verify | Low | 0.5 day |
| 13 | Integration Tests | ❌ Missing | High | 2 days |
| 14 | Benchmark Tests | ❌ Missing | Low | 1 day |
| 15 | Documentation | ⚠️ Needed | Medium | 1 day |

**Total Estimated Effort: ~15-20 working days**

---

## Recommended Removal Phases

### Phase 1: Critical Features (Weeks 1-3)
- [ ] Implement Consumer Group Detail Page (#1)
- [ ] Implement Schema Resource (#2)
- [ ] Implement Context Switching (#3)
- [ ] Add integration tests (#13)

### Phase 2: Feature Verification (Week 4)
- [ ] Verify clipboard functionality (#7)
- [ ] Verify schema display (#8)
- [ ] Verify table search (#9)
- [ ] Verify notifications (#10)
- [ ] Verify refresh indicators (#11)

### Phase 3: Polish & Documentation (Week 5)
- [ ] Add benchmark tests (#14)
- [ ] Update documentation (#15)
- [ ] Final testing and bug fixes

### Phase 4: Removal (Week 6)
- [ ] Feature freeze on old UI
- [ ] Final comparison testing
- [ ] Remove `pkg/kafui/` directory
- [ ] Clean up imports and dependencies
- [ ] Update `go.mod` to remove tview/tcell dependencies

---

## Dependencies to Remove

After deleting old UI, these dependencies can be removed from `go.mod`:

```
github.com/rivo/tview
github.com/gdamore/tcell/v2
github.com/gdamore/encoding (if only used by tcell)
```

**Note:** Ensure all functionality is working in new UI before removing these!

---

## Risk Mitigation

### High-Risk Areas
1. **Message Consumption**: Core feature, must work flawlessly
2. **Context Switching**: Users rely on multi-cluster support
3. **Schema Registry**: Complex integration, needs thorough testing

### Mitigation Strategies
1. Keep old UI in codebase during development
2. Add feature flag to switch between old/new UI
3. Comprehensive integration testing before removal
4. Beta release with new UI before final removal
5. Gather user feedback on new UI

---

## Success Criteria

Before removing old UI, verify:

- [ ] All resource types (Topics, Consumer Groups, Schemas, Contexts) are fully functional
- [ ] Navigation between pages works correctly
- [ ] Message consumption is stable with error recovery
- [ ] Search and filtering work on all pages
- [ ] Copy to clipboard works everywhere needed
- [ ] Context switching is functional
- [ ] Schema information displays correctly
- [ ] All integration tests pass
- [ ] Performance is equal or better than old UI
- [ ] Documentation is updated
- [ ] User feedback is positive (if beta released)

---

**Document Created:** 2026-02-24  
**Last Updated:** 2026-02-24  
**Owner:** Development Team

# UI Debug Tools

This folder contains standalone debug applications for testing UI components in isolation.

## Topic Page Debug

Run the topic page with mock data:

```bash
cd ui_debug/topic_page
go run main.go
```

Or use the pre-built binary:

```bash
./topic_debug
```

### What it tests

- Topic page layout with mock message consumption
- Table rendering with varying message sizes
- Column width adaptation to terminal size
- Message truncation with ANSI codes
- 20-message limit enforcement

### Controls

- `j/k` or `↑/↓` - Navigate messages
- `/` - Search/filter messages
- `Space` - Pause/resume consumption
- `r` - Restart consumption
- `Enter` - View message details
- `Esc` - Go back
- `q` or `Ctrl+C` - Quit
- `t` or `Ctrl+S` - Toggle sidebar (requires width >= 81 chars)

### Sidebar Visibility

The sidebar is shown automatically when:
- Terminal width >= 120 characters AND
- Terminal height >= 30 rows

If your terminal is smaller, the sidebar is hidden to preserve content space.
You can still toggle it with `t` or `Ctrl+S` when width >= 81 characters.

**For best debugging experience, resize your terminal to at least 120x30.**

### Mock Data

The mock data source generates messages with varying content lengths:
- Short JSON messages
- Long user event data
- Order details with nested arrays
- Log entries with long error messages
- Metrics with multiple fields

This tests the table's ability to handle different text lengths while maintaining layout integrity.

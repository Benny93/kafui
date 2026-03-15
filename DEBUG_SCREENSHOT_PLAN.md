# Debug Screenshot Feature Plan

This document describes a plan to add a debug feature that captures the current TUI screen content to a text file for visualization and layout debugging purposes.

---

## Overview

The debug screenshot feature will allow developers to capture the exact character-by-character output of the current TUI screen to a text file. This is invaluable for:

- Debugging layout issues without screen recording
- Sharing visual bugs in bug reports
- Comparing expected vs actual rendering
- Testing on headless CI environments
- Documenting UI states

---

## Key Binding

### Recommended Key: `Ctrl+D`

**Rationale:**
- Mnemonic: **D**ebug / **D**ump
- Avoids conflict with `Ctrl+S` which is used for sidebar toggling and can conflict with terminal flow control (XOFF).
- Fits well with other debug-related shortcuts.
- Easily accessible with one hand.

### Configuration

The key binding should be:
- **Hard-coded** as `Ctrl+D` by default in debug mode.
- **Documented** in the help view (`?`) under a specialized debug section.
- **Disabled** in production builds (via build tag) to prevent accidental data leaks.

---

## File Output

### Location & Configuration

1. **Default Location:** `$TMPDIR/kafui-screenshot-<timestamp>.txt`
2. **Configurable Path:** Users can specify a `screenshot_dir` in `config.yaml`.
3. **Filename Convention:** `kafui-screenshot-YYYYMMDD-HHMMSS.txt`

### Output Formats (Enhanced)

The system should support capturing in two formats (configurable or simultaneous):

1. **Plain Text (`.txt`):** Stripped of all ANSI escape codes for easy reading and diffing.
2. **ANSI Color (`.ansi` or `.txt` with codes):** Preserves all styling, colors, and formatting. This allows viewing the "screenshot" back in a terminal using `cat`.

**Example Metadata Header:**
```txt
# Kafui Debug Screenshot
# Timestamp: 2026-03-15 14:30:22 UTC
# Version: v1.2.3
# Platform: darwin/arm64
# Terminal: xterm-256color
# Dimensions: 120x40
# Current Page: topic_list
# Page Context: cluster="prod-west", topic_count=142
# ================================================================================
```

---

## Feature Enhancements

### 1. Data Redaction (Sanitization)
To allow sharing screenshots without exposing sensitive internal data, add a "Redact Sensitive Data" mode.
- If enabled via config, replace topic names, group IDs, and message contents with generic placeholders (e.g., `TOPIC-XXXX`, `MSG-MASKED`).
- Keybinding: `Ctrl+Alt+D` for a redacted screenshot (optional).

### 2. Environment Context
Include more comprehensive environment details:
- Go version, OS version.
- TTY type.
- Active configuration flags.

### 3. Screenshot Flash Effect
Provide visual feedback when a screenshot is taken.
- Briefly invert the screen colors or flash the status bar.
- Prevents ambiguity about whether the keypress was registered.

---

## Implementation Plan

### Phase 1: Core Screenshot Service

**File:** `pkg/ui/debug/screenshot.go`

**Responsibilities:**
- Capture current view buffer.
- Optional: Strip ANSI codes for plain text version.
- Optional: Redact data using a simple pattern matcher.
- Handle file I/O with proper permissions (0600).

```go
type ScreenshotFormat int
const (
    FormatPlainText ScreenshotFormat = iota
    FormatANSI
)

func Capture(view string, options CaptureOptions) (string, error)
```

### Phase 2: Integration with Model

**File:** `pkg/ui/ui.go`

- Add `handleDebugKey` method.
- Check build tags to ensure this logic is only compiled in `debug` builds.

```go
case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+d":
        if isDebugBuild {
            return m.takeScreenshot(false)
        }
    case "ctrl+alt+d":
        if isDebugBuild {
            return m.takeScreenshot(true) // redacted
        }
    }
```

### Phase 3: Help System Integration

**File:** `pkg/ui/keys/keys.go`

- Add `DebugScreenshot` and `DebugScreenshotRedacted` to `GlobalKeyMap`.
- Ensure they are only shown in the help menu when `debug` build tag is active.

```go
DebugScreenshot: key.NewBinding(
    key.WithKeys("ctrl+d"),
    key.WithHelp("ctrl+d", "save screenshot"),
),
```

### Phase 4: Configuration & Build Tags

**File:** `pkg/ui/core/config.go`

- Add `screenshot_dir` to the `Config` struct.
- Default to os.TempDir() if not specified.

**Build Tags:**
- Use `//go:build debug` for the entire `pkg/ui/debug` package.
- Provide a stub implementation for non-debug builds to avoid compilation errors.

### Phase 5: Testing & Verification

**Unit Tests:**
- Test ANSI code stripping logic.
- Test redaction regex/patterns.
- Test file naming and path resolution.

**Integration Tests:**
- Verify key presses trigger file creation in a mock environment.
- Check metadata header completeness.

---

## Acceptance Criteria

- [x] `Ctrl+D` captures current screen.
- [x] Support for both Plain Text and ANSI formats.
- [x] Configurable output directory.
- [x] Metadata includes environment context (OS, Dimensions, Page).
- [x] Visual feedback (status message + optional flash).
- [x] File permissions set to `0600`.
- [x] Excluded from production binaries via `//go:build debug`.
- [x] Optional: Redaction of sensitive data.

---

## Related Documentation

- Update `README.md` with instructions for using debug mode.
- Document `screenshot_dir` in `example-config.yaml`.

---

## Summary

| Aspect | Decision |
|--------|----------|
| **Key Binding** | `Ctrl+D` (Standard) / `Ctrl+Alt+D` (Redacted) |
| **Output Location** | `$TMPDIR` or configurable `screenshot_dir` |
| **File Format** | Plain text and/or ANSI Color |
| **Build Tag** | `debug` (excluded from release) |
| **File Permissions** | `0600` (owner read/write only) |
| **Help Display** | Full help only, under "Debug" section |

---

**Status:** Plan ready for implementation
**Priority:** Medium (developer productivity feature)
**Estimated Effort:** 0.5-1 day


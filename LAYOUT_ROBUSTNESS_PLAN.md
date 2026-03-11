# Layout Robustness Plan

## Executive Summary

This document analyzes the differences between the original CRUSH project's UI implementation and our current implementation in `pkg/ui/template`. The goal is to make our layout system more robust when handling varying content sizes, preventing layout breakage during window resizing or when content exceeds available space.

## Analysis: CRUSH vs. Our Implementation

### 1. Content Width Management

#### CRUSH Approach
CRUSH uses a **capped width system** for content readability and layout stability:

```go
// From crush/internal/ui/chat/messages.go
const (
    MessageLeftPaddingTotal = 2
    maxTextWidth = 120
)

func cappedMessageWidth(availableWidth int) int {
    return min(availableWidth-MessageLeftPaddingTotal, maxTextWidth)
}
```

**Key Insights:**
- Maximum text width of 120 characters ensures readability
- Consistent padding calculation across all components
- Content is truncated or wrapped when exceeding the cap

#### Our Current Approach
Our implementation passes full width to content providers without capping:

```go
// From pkg/ui/template/ui/components/content.go
func (c *content) View() string {
    // ...
    content = c.provider.RenderContent(c.width, c.height)
    // ...
    return style.
        Width(c.width - 2).   // Account for border
        Height(c.height - 2). // Account for border
        Padding(1).
        Render(content)
}
```

**Problem:** Content providers receive the full width but may not handle it properly, causing:
- Text overflow beyond container bounds
- Misaligned UI elements when content is too wide
- Broken layouts when window is resized

### 2. Content Height and Scrolling

#### CRUSH Approach
CRUSH implements **explicit scrollbar rendering** and viewport management:

```go
// From crush/internal/ui/common/scrollbar.go
func Scrollbar(s *styles.Styles, height, contentSize, viewportSize, offset int) string {
    if height <= 0 || contentSize <= viewportSize {
        return ""
    }
    
    // Calculate thumb size (minimum 1 character)
    thumbSize := max(1, height*viewportSize/contentSize)
    
    // Calculate thumb position
    maxOffset := contentSize - viewportSize
    thumbPos := 0
    if trackSpace > 0 && maxOffset > 0 {
        thumbPos = min(trackSpace, offset*trackSpace/maxOffset)
    }
    
    // Render scrollbar...
}
```

**Key Insights:**
- Scrollbar only renders when content exceeds viewport
- Thumb size proportional to content/viewport ratio
- Explicit offset tracking for scroll position

#### Our Current Approach
We rely on lipgloss's `Height()` constraint but don't handle overflow:

```go
// From pkg/ui/template/ui/components/content.go
return style.
    Width(c.width - 2).
    Height(c.height - 2).
    Padding(1).
    Render(content)
```

**Problem:** 
- Content exceeding height is silently clipped without visual feedback
- No scrollbar indication for scrollable content
- Users unaware they can scroll to see more

### 3. Text Truncation

#### CRUSH Approach
CRUSH uses **explicit truncation** with ellipsis:

```go
// From crush/internal/ui/common/elements.go
func Status(t *styles.Styles, opts StatusOpts, width int) string {
    // ...
    if description != "" {
        extraContentWidth := lipgloss.Width(opts.ExtraContent)
        if extraContentWidth > 0 {
            extraContentWidth += 1
        }
        description = ansi.Truncate(description, width-lipgloss.Width(icon)-lipgloss.Width(title)-2-extraContentWidth, "…")
        description = t.Base.Foreground(descriptionColor).Render(description)
    }
    // ...
}
```

**Key Insights:**
- Calculates exact available width before rendering
- Uses `ansi.Truncate()` with custom ellipsis character
- Accounts for all elements (icon, title, extra content) in width calculation

#### Our Current Approach
We rarely truncate text, relying on container constraints:

```go
// From pkg/ui/template/ui/components/sidebar.go
func (s *sidebar) renderSection(section providers.SidebarSection, maxItems, width int) string {
    // ...
    line := fmt.Sprintf("%s %s", statusStyle.Render(item.Icon), item.Text)
    
    // Add value if there's space and it's not empty
    if item.Value != "" {
        valueText := t.S().Muted.Render(fmt.Sprintf("(%s)", item.Value))
        totalWidth := lipgloss.Width(line) + lipgloss.Width(valueText)
        if totalWidth <= width {
            spacing := width - totalWidth
            line = line + strings.Repeat(" ", spacing) + valueText
        }
    }
    // ...
}
```

**Problem:**
- Only checks if content fits, doesn't truncate if it doesn't
- No ellipsis indication for truncated content
- Long text can break layout alignment

### 4. Adaptive Layout Based on Available Space

#### CRUSH Approach
CRUSH calculates **dynamic item limits** based on available space:

```go
// From crush/internal/ui/model/sidebar.go
func getDynamicHeightLimits(availableHeight int) (maxFiles, maxLSPs, maxMCPs int) {
    const (
        minItemsPerSection      = 2
        defaultMaxFilesShown    = 10
        defaultMaxLSPsShown     = 8
        defaultMaxMCPsShown     = 8
        minAvailableHeightLimit = 10
    )

    // If we have very little space, use minimum values
    if availableHeight < minAvailableHeightLimit {
        return minItemsPerSection, minItemsPerSection, minItemsPerSection
    }

    // Distribute available height among sections
    totalSections := 3
    heightPerSection := availableHeight / totalSections

    // Calculate limits for each section
    maxFiles = max(minItemsPerSection, min(defaultMaxFilesShown, heightPerSection))
    // ...
}
```

**Key Insights:**
- Minimum items per section ensures usability
- Distributes space proportionally among sections
- Priority-based allocation (files > LSPs > MCPs)

#### Our Current Approach
We use a simpler calculation:

```go
// From pkg/ui/template/ui/components/sidebar.go
func (s *sidebar) calculateMaxItems(availableHeight, numSections int) int {
    headerSpace := numSections * 2
    itemSpace := availableHeight - headerSpace

    if itemSpace <= 0 {
        return 1
    }

    maxPerSection := itemSpace / numSections
    if maxPerSection < 2 {
        return 2
    }

    return maxPerSection
}
```

**Problem:**
- Equal distribution doesn't account for section importance
- No minimum height limit check
- Can result in 0 or 1 items when space is tight

### 5. Minimum Size Handling

#### CRUSH Approach
CRUSH has **multiple size modes** with appropriate fallbacks:

```go
// From crush/internal/ui/styles/theme.go (similar to our implementation)
const (
    MinimumWindowWidth  = 25
    MinimumWindowHeight = 15
    SmallScreenWidth    = 55
    SmallScreenHeight   = 20
    CompactModeWidth    = 120
    CompactModeHeight   = 30
)

func GetSizeMode(width, height int) SizeMode {
    if width < MinimumWindowWidth || height < MinimumWindowHeight {
        return SizeModeMinimum  // Shows "Window too small!" message
    }
    if width < SmallScreenWidth || height < SmallScreenHeight {
        return SizeModeSmall  // Minimal UI
    }
    // ...
}
```

**Key Insights:**
- Multiple breakpoints for different UI adaptations
- Clear "Window too small!" message at minimum size
- Progressive enhancement as size increases

#### Our Current Approach
We have similar size mode detection but inconsistent application:

```go
// From pkg/ui/template/ui/styles/utils.go
// We have the same constants and GetSizeMode function
```

**Problem:**
- Size mode is detected but not consistently used in content rendering
- Content providers don't adapt their output based on size mode
- Some components ignore size mode entirely

### 6. Layout Calculation and Space Accounting

#### CRUSH Approach
CRUSH uses **explicit space accounting** with layout utilities:

```go
// From crush/internal/ui/model/landing.go
func (m *UI) landingView() string {
    width := m.layout.main.Dx()
    cwd := common.PrettyPath(t, m.com.Config().WorkingDir(), width)
    
    parts := []string{cwd, "", m.modelInfo(width)}
    infoSection := lipgloss.JoinVertical(lipgloss.Left, parts...)
    
    // Calculate remaining space after header
    _, remainingHeightArea := layout.SplitVertical(m.layout.main, layout.Fixed(lipgloss.Height(infoSection)+1))
    
    // Split remaining width for columns
    mcpLspSectionWidth := min(30, (width-1)/2)
    
    lspSection := m.lspInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)
    mcpSection := m.mcpInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)
    
    // ...
}
```

**Key Insights:**
- Uses `layout.SplitVertical/Horizontal` for precise space division
- Calculates remaining space after each element
- Explicit width/height passing to each component

#### Our Current Approach
We use simpler lipgloss joins without explicit space calculation:

```go
// From pkg/ui/template/ui/reusable_app.go
func (a *ReusableApp) View() string {
    // ...
    if a.showSidebar && a.sizeMode >= styles.SizeModeNormal {
        middleArea = lipgloss.JoinHorizontal(
            lipgloss.Bottom,
            a.content.View(),
            a.sidebar.View(),
        )
    } else {
        middleArea = a.content.View()
    }
    // ...
    return lipgloss.JoinVertical(lipgloss.Left, components...)
}
```

**Problem:**
- No explicit space calculation before rendering
- Components don't know their exact allocated space
- Can lead to over-rendering and layout shifts

## Identified Issues in Our Implementation

### Critical Issues

1. **No Content Width Capping**
   - Content can exceed container width
   - Long lines break layout alignment
   - No maximum readability width

2. **Missing Text Truncation**
   - Long text overflows containers
   - No ellipsis indication
   - Breaks sidebar alignment

3. **No Scrollbar for Overflow Content**
   - Content clipped without indication
   - Users unaware of scrollable content
   - No visual feedback for scroll position

4. **Inconsistent Size Mode Usage**
   - Size mode detected but not always used
   - Content doesn't adapt to available space
   - Same content rendered regardless of screen size

### Medium Priority Issues

5. **Simplistic Space Distribution**
   - Equal distribution among sections
   - No priority-based allocation
   - Doesn't account for section importance

6. **Imprecise Height Calculation**
   - Doesn't account for all occupied space
   - Can result in negative content height
   - No minimum content height enforcement

7. **No Content Caching**
   - Re-renders content on every update
   - Inefficient for large content
   - No width-specific caching

## Implementation Plan

### Phase 1: Core Infrastructure (Week 1)

#### 1.1 Add Content Width Capping
**File:** `pkg/ui/template/ui/components/content.go`

```go
const (
    ContentLeftPadding = 4  // Border (2) + Padding (2)
    MaxContentWidth    = 120
)

func cappedContentWidth(availableWidth int) int {
    return min(availableWidth-ContentLeftPadding, MaxContentWidth)
}
```

**Changes:**
- Add constants for padding and max width
- Create `cappedContentWidth()` helper function
- Pass capped width to content providers

#### 1.2 Implement Text Truncation Utility
**File:** `pkg/ui/template/ui/styles/utils.go`

```go
import "github.com/charmbracelet/x/ansi"

// TruncateText truncates text to fit within the given width
func TruncateText(text string, availableWidth int, ellipsis string) string {
    if ellipsis == "" {
        ellipsis = "…"
    }
    return ansi.Truncate(text, availableWidth, ellipsis)
}

// TruncateWithEllipsis truncates text with standard ellipsis
func TruncateWithEllipsis(text string, availableWidth int) string {
    return TruncateText(text, availableWidth, "…")
}
```

**Changes:**
- Add truncation utility functions
- Export for use in all components
- Standardize ellipsis character

#### 1.3 Add Scrollbar Component
**File:** `pkg/ui/template/ui/components/scrollbar.go` (new file)

```go
package components

import (
    "strings"
    "github.com/Benny93/kafui/pkg/ui/template/ui/styles"
)

// Scrollbar renders a vertical scrollbar
func Scrollbar(height, contentSize, viewportSize, offset int) string {
    if height <= 0 || contentSize <= viewportSize {
        return ""
    }
    
    thumbSize := max(1, height*viewportSize/contentSize)
    maxOffset := contentSize - viewportSize
    if maxOffset <= 0 {
        return ""
    }
    
    trackSpace := height - thumbSize
    thumbPos := 0
    if trackSpace > 0 {
        thumbPos = min(trackSpace, offset*trackSpace/maxOffset)
    }
    
    var sb strings.Builder
    t := styles.CurrentTheme()
    
    for i := range height {
        if i > 0 {
            sb.WriteString("\n")
        }
        if i >= thumbPos && i < thumbPos+thumbSize {
            sb.WriteString(t.S().Muted.Render("│"))
        } else {
            sb.WriteString(t.S().Subtle.Render("│"))
        }
    }
    
    return sb.String()
}
```

**Changes:**
- Create new scrollbar component
- Integrate with content component
- Add scroll offset tracking to content interface

### Phase 2: Component Updates (Week 2)

#### 2.1 Update Content Component
**File:** `pkg/ui/template/ui/components/content.go`

```go
type content struct {
    width, height int
    focused       bool
    provider      providers.ContentProvider
    scrollOffset  int  // Add scroll tracking
}

func (c *content) View() string {
    if c.width == 0 || c.height == 0 {
        return ""
    }

    t := styles.CurrentTheme()
    
    // Calculate capped width for content
    contentWidth := cappedContentWidth(c.width)
    
    var content string
    if c.provider != nil {
        // Pass capped width to provider
        content = c.provider.RenderContent(contentWidth, c.height)
    }
    
    // Check if content needs scrollbar
    contentLines := strings.Split(content, "\n")
    contentHeight := len(contentLines)
    needsScrollbar := contentHeight > c.height-4  // Account for borders
    
    // Add scrollbar if needed
    if needsScrollbar {
        scrollbar := Scrollbar(c.height-4, contentHeight, c.height-4, c.scrollOffset)
        // Adjust content layout to include scrollbar
        content = lipgloss.JoinHorizontal(lipgloss.Top, content, scrollbar)
    }
    
    // Apply styling
    var style lipgloss.Style
    if c.focused {
        style = t.S().Base.
            Border(lipgloss.RoundedBorder()).
            BorderForeground(t.BorderFocus)
    } else {
        style = t.S().Base.
            Border(lipgloss.RoundedBorder()).
            BorderForeground(t.Border)
    }

    return style.
        Width(c.width - 2).
        Height(c.height - 2).
        Padding(1).
        Render(content)
}
```

**Changes:**
- Add scroll offset tracking
- Implement width capping
- Add scrollbar rendering
- Handle overflow content

#### 2.2 Update Sidebar Component
**File:** `pkg/ui/template/ui/components/sidebar.go`

```go
func (s *sidebar) renderSection(section providers.SidebarSection, maxItems, width int) string {
    t := styles.CurrentTheme()
    var lines []string

    // Section header with truncation
    title := styles.TruncateWithEllipsis(section.GetTitle(), width-2)
    header := styles.Section(title, width)
    lines = append(lines, header)

    // Get items from the section
    items := section.RenderItems(maxItems, width)

    // Render each item with truncation
    for _, item := range items {
        statusStyle := s.getItemStatusStyle(item.Status)
        
        // Calculate available width for text
        iconWidth := lipgloss.Width(statusStyle.Render(item.Icon))
        valueWidth := 0
        if item.Value != "" {
            valueWidth = lipgloss.Width(fmt.Sprintf(" (%s)", item.Value))
        }
        
        textAvailableWidth := width - iconWidth - valueWidth - 2  // 2 for spacing
        
        // Truncate text if needed
        itemText := styles.TruncateWithEllipsis(item.Text, textAvailableWidth)
        
        line := fmt.Sprintf("%s %s", statusStyle.Render(item.Icon), itemText)
        
        // Add value if there's space
        if item.Value != "" && lipgloss.Width(line)+valueWidth <= width {
            valueText := t.S().Muted.Render(fmt.Sprintf("(%s)", item.Value))
            spacing := width - lipgloss.Width(line) - lipgloss.Width(valueText)
            line = line + strings.Repeat(" ", max(0, spacing)) + valueText
        }
        
        lines = append(lines, line)
    }

    return strings.Join(lines, "\n")
}
```

**Changes:**
- Truncate section titles
- Truncate item text with ellipsis
- Calculate exact available width before rendering
- Handle long values gracefully

#### 2.3 Improve Space Distribution
**File:** `pkg/ui/template/ui/components/sidebar.go`

```go
func (s *sidebar) calculateMaxItems(availableHeight, numSections int) []int {
    const (
        minItemsPerSection   = 2
        defaultMaxItems      = 10
        headerSpacePerSection = 2  // Header + spacing
    )
    
    // If we have very little space, use minimum values
    if availableHeight < 10 {
        limits := make([]int, numSections)
        for i := range limits {
            limits[i] = minItemsPerSection
        }
        return limits
    }
    
    // Calculate total header space
    totalHeaderSpace := numSections * headerSpacePerSection
    itemSpace := availableHeight - totalHeaderSpace
    
    if itemSpace <= 0 {
        limits := make([]int, numSections)
        for i := range limits {
            limits[i] = minItemsPerSection
        }
        return limits
    }
    
    // Distribute space with priority (first sections get more)
    limits := make([]int, numSections)
    remainingSpace := itemSpace
    
    for i := 0; i < numSections; i++ {
        sectionsRemaining := numSections - i
        spacePerSection := remainingSpace / sectionsRemaining
        
        limits[i] = max(minItemsPerSection, min(defaultMaxItems, spacePerSection))
        remainingSpace -= limits[i]
    }
    
    return limits
}
```

**Changes:**
- Return slice of limits instead of single value
- Priority-based distribution (earlier sections get more space)
- Minimum height check
- Better space utilization

### Phase 3: Provider Updates (Week 3)

#### 3.1 Update Content Provider Interface
**File:** `pkg/ui/template/ui/providers/interfaces.go`

```go
// ContentProvider defines the interface for providing main content
type ContentProvider interface {
    // RenderContent returns the content to display in the main content area
    // width is already capped for readability, height accounts for borders
    RenderContent(width, height int) string

    // HandleContentUpdate allows the provider to handle messages and return commands
    HandleContentUpdate(msg tea.Msg) tea.Cmd

    // InitContent initializes the content provider
    InitContent() tea.Cmd
    
    // GetContentSize returns the actual content size for scrollbar calculation
    GetContentSize(width int) (lines int)  // New method
}
```

**Changes:**
- Add `GetContentSize()` method
- Document width/height expectations
- Enable scrollbar calculation

#### 3.2 Update Default Content Provider
**File:** `pkg/ui/template/ui/providers/default_providers.go`

```go
func (d *DefaultContentProvider) RenderContent(width, height int) string {
    t := styles.CurrentTheme()
    sizeMode := styles.GetSizeMode(width, height)
    
    // Adapt content based on size mode
    var description []string
    if sizeMode >= styles.SizeModeCompact {
        description = []string{
            // Full content for larger screens
        }
    } else if sizeMode == styles.SizeModeSmall {
        description = []string{
            // Minimal content for small screens
        }
    } else {
        description = []string{
            // Ultra-minimal for minimum size
        }
    }
    
    // Truncate lines to fit width
    var adaptedLines []string
    for _, line := range description {
        adaptedLine := styles.TruncateWithEllipsis(line, width-4)  // Account for padding
        adaptedLines = append(adaptedLines, adaptedLine)

    }
    
    // Limit lines to fit height
    maxLines := height - 4  // Account for borders
    if len(adaptedLines) > maxLines {
        adaptedLines = adaptedLines[:maxLines]
        // Add ellipsis to indicate more content
        adaptedLines[len(adaptedLines)-1] = styles.TruncateWithEllipsis(
            adaptedLines[len(adaptedLines)-1], 
            width-7,  // Account for "..."
        ) + "..."
    }
    
    // ... rest of rendering logic
}

func (d *DefaultContentProvider) GetContentSize(width int) int {
    // Return total lines of content for scrollbar calculation
    return len(d.getAllContentLines())
}
```

**Changes:**
- Adapt content based on size mode
- Truncate lines to fit width
- Limit lines to fit height
- Implement `GetContentSize()`

### Phase 4: Testing and Validation (Week 4)

#### 4.1 Create Test Cases
**File:** `pkg/ui/template/ui/components/content_test.go` (new file)

```go
package components

import (
    "testing"
    "github.com/Benny93/kafui/pkg/ui/template/ui/providers"
)

func TestContentWidthCapping(t *testing.T) {
    tests := []struct {
        name           string
        availableWidth int
        expectedWidth  int
    }{
        {"Small width", 50, 46},
        {"Normal width", 100, 96},
        {"Large width", 200, 120},  // Should cap at MaxContentWidth
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := cappedContentWidth(tt.availableWidth)
            if result != tt.expectedWidth {
                t.Errorf("Expected %d, got %d", tt.expectedWidth, result)
            }
        })
    }
}

func TestContentScrollbar(t *testing.T) {
    // Test scrollbar rendering
    scrollbar := Scrollbar(20, 50, 20, 0)
    if scrollbar == "" {
        t.Error("Expected scrollbar for overflow content")
    }
    
    // Test no scrollbar when content fits
    scrollbar = Scrollbar(20, 10, 20, 0)
    if scrollbar != "" {
        t.Error("Expected no scrollbar when content fits")
    }
}
```

#### 4.2 Manual Testing Scenarios

Create test scenarios for:

1. **Window Resizing**
   - Start with large window, gradually resize smaller
   - Verify content adapts without breaking
   - Check scrollbar appears/disappears appropriately

2. **Long Content**
   - Test with very long lines (200+ characters)
   - Verify truncation with ellipsis
   - Check layout alignment maintained

3. **Many Items**
   - Test sidebar with many items
   - Verify space distribution works
   - Check minimum items per section

4. **Size Mode Transitions**
   - Test all size mode breakpoints
   - Verify appropriate content for each mode
   - Check smooth transitions

## Success Criteria

### Functional Requirements

1. ✅ **No Layout Breakage**
   - Layout remains stable at all window sizes
   - No text overflow beyond container bounds
   - Consistent alignment regardless of content

2. ✅ **Visual Feedback**
   - Scrollbar appears when content overflows
   - Ellipsis indicates truncated text
   - "Window too small!" message at minimum size

3. ✅ **Adaptive Content**
   - Content adapts to size mode
   - Appropriate detail level for available space
   - Priority-based space distribution

4. ✅ **Performance**
   - No noticeable lag during resize
   - Efficient content rendering
   - Optional: Content caching for repeated widths

### Quality Metrics

- **Test Coverage:** >80% for new/modified components
- **Manual Testing:** All scenarios pass
- **User Feedback:** No layout-related issues reported
- **Code Review:** Approved by team

## Migration Guide

### For Existing Content Providers

1. **Update Interface Implementation**
   ```go
   // Add GetContentSize method
   func (p *YourProvider) GetContentSize(width int) int {
       return len(strings.Split(p.renderContent(width), "\n"))
   }
   ```

2. **Handle Width Capping**
   ```go
   // Content width is now capped, adjust rendering
   func (p *YourProvider) RenderContent(width, height int) string {
       // width is now capped at MaxContentWidth
       // No need to handle extremely wide content
   }
   ```

3. **Use Truncation Utilities**
   ```go
   // Replace manual truncation with utility
   import "github.com/Benny93/kafui/pkg/ui/template/ui/styles"
   
   truncated := styles.TruncateWithEllipsis(longText, availableWidth)
   ```

### For New Components

1. **Always Use Width Capping**
   ```go
   contentWidth := cappedContentWidth(availableWidth)
   ```

2. **Implement Scrollbar for Scrollable Content**
   ```go
   if contentHeight > viewportHeight {
       scrollbar := Scrollbar(...)
   }
   ```

3. **Truncate Text Proactively**
   ```go
   text := styles.TruncateWithEllipsis(longText, width)
   ```

## Future Enhancements

1. **Content Caching**
   - Cache rendered content by width
   - Invalidate cache on content change
   - Memory-efficient cache eviction

2. **Horizontal Scrolling**
   - Support for wide content that can't be truncated
   - Horizontal scrollbar component
   - Keyboard navigation for horizontal scroll

3. **Responsive Typography**
   - Adjust font size based on available space
   - Conditional rendering of decorative elements
   - Adaptive spacing and padding

4. **Layout Profiling**
   - Performance metrics for layout rendering
   - Identify bottlenecks in content rendering
   - Optimize for large content sets

## Conclusion

By implementing these changes, our UI will match CRUSH's robustness in handling varying content sizes. The key principles are:

1. **Cap widths** for readability and stability
2. **Truncate text** with clear visual indicators
3. **Show scrollbars** for overflow content
4. **Adapt to size modes** with appropriate content
5. **Calculate space explicitly** before rendering

This will result in a more professional, stable UI that handles edge cases gracefully and provides clear visual feedback to users.

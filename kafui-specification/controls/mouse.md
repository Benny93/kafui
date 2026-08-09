# Controls — Normative Mouse Map

This is the complete, normative mouse table referenced by [spec.md](spec.md).
The governing rule is that the mouse and the keyboard are peers: nothing is
mouse-only, and every non-typing action is mouse-reachable — directly if it has a
click target, otherwise through the actions menu opened by right-click.

---

## 1. Gesture vocabulary

The same gesture means the same thing everywhere.

| Gesture | Meaning | Never |
|---|---|---|
| Left click | Focus and select what is under the pointer | Never destructive, never navigates away |
| Left click on the already-selected row | Activate — identical to `Enter` | |
| Double click | Activate — identical to `Enter`; optional, since click-on-selected already covers it without a timer | |
| Right click | Open the actions menu for what is under the pointer, and select it | |
| Wheel up / down | Move the pane under the pointer by one scroll unit: three lines in a content pane, one page in a paginated list | Never moves the cursor within the visible rows |
| Drag on a scrollbar | Scroll the pane to the dragged position | |
| Drag on a column edge | Resize the column | |
| Hover | Highlight the element under the pointer | Never triggers an action |

Scrolling and selecting are separate concepts. A wheel gesture moves the
viewport, not the cursor. Lists here are paginated to exactly what fits, so
their scroll unit is a page: the wheel moves through the data without dragging
the cursor down the rows one at a time, which is what it used to do.

There is no horizontal scrolling anywhere. Tables fit their pane by flexing
their columns and content panes wrap, so no pane is ever wider than the space it
is given — which is what keeps `←`/`→` free to mean "one page" everywhere.

---

## 2. Click targets

Every element in this table responds to the mouse. Elements are listed with the
keyboard equivalent that must also exist.

### Chrome

| Element | Left click | Right click | Keyboard equivalent |
|---|---|---|---|
| Sidebar entry | Navigate to that section | — | Command palette, or the section's own binding |
| Sidebar collapse control | Show/hide the sidebar | — | `Ctrl+B` |
| Breadcrumb segment | Navigate to that ancestor screen | — | `Esc`, repeatedly |
| Tab in a tab strip | Activate that tab | Actions for that tab's content | `1`…`9`, `Tab` |
| Hint in the hint bar | Run that action | — | The key the hint displays |
| Help overlay entry | Run that action and close the overlay | — | The key the entry displays |
| Notification / toast | Dismiss it | — | `Esc` |

### Lists and tables

| Element | Left click | Right click | Keyboard equivalent |
|---|---|---|---|
| Row | Select it | Select it and open its actions menu | `↑`/`↓`, then `a` |
| Row, when already selected | Activate it | | `Enter` |
| Row selection marker | Toggle that row's selection | | `Space` |
| Column header | Sort by that column; click again to reverse | Choose sort column and direction | `s`, `S` |
| Pagination control | Go to that page | — | `←`/`→`, `Shift+←`/`Shift+→` |
| Empty area below the rows | Clear the selection | Actions for the screen itself | `Esc` |

### Content panes

| Element | Left click | Right click | Keyboard equivalent |
|---|---|---|---|
| Pane body | Focus the pane | Actions for the pane's content | `Tab` |
| Search match | Make it the current match | — | `n` / `N` |
| Fold or expand control | Expand or collapse that node | — | `Enter` |

### Overlays

| Element | Left click | Right click | Keyboard equivalent |
|---|---|---|---|
| Dialog button | Activate it | — | `Tab`, then `Enter` |
| Dialog backdrop | Dismiss the dialog without acting | — | `Esc` |
| Menu entry | Run it | — | `↑`/`↓` then `Enter`, or its mnemonic letter |
| Disabled menu entry | Nothing; the reason stays visible | — | — |
| Form field | Focus it and place the caret | — | `Tab` |
| Form checkbox or radio | Toggle it | — | `Space` |

---

## 3. Requirements

### Requirement: Uniform Gesture Semantics

Every click target SHALL follow the gesture vocabulary in section 1. A screen
SHALL NOT give a gesture a meaning that differs from the table.

#### Scenario: Consistent single click across screens
- **WHEN** the user single-clicks a row on any list screen in the application
- **THEN** the row is selected and nothing is opened, on every such screen alike

#### Scenario: Activation requires a deliberate gesture
- **WHEN** the user single-clicks an unselected row
- **THEN** no navigation occurs, so a stray click cannot leave the current screen

### Requirement: Wheel Scrolls, Never Selects

Wheel gestures SHALL scroll the viewport of the pane under the pointer and SHALL
leave the selection unchanged.

#### Scenario: Scrolling past the selection
- **WHEN** the user scrolls a long list until the selected row is off screen
- **THEN** the selection is still the same row, and moving the selection with the keyboard scrolls it back into view

#### Scenario: Pointer determines the target
- **WHEN** a screen shows two scrollable panes and the pointer is over the unfocused one
- **THEN** the wheel scrolls the pane under the pointer, without moving focus

### Requirement: Right-Click Opens the Actions Menu

Right-click anywhere SHALL open the actions menu for the most specific element
under the pointer, selecting it first where it is selectable.

#### Scenario: Right-click on an unselected row
- **WHEN** the user right-clicks a row that is not selected
- **THEN** that row becomes the selection and its actions menu opens

#### Scenario: Right-click on empty space
- **WHEN** the user right-clicks where there is no row or control
- **THEN** the actions menu for the screen itself opens

### Requirement: Hover Discoverability

The element under the pointer SHALL be visually distinguished whenever it is
clickable, and SHALL NOT be distinguished when it is not.

#### Scenario: Discovering click targets
- **WHEN** the user moves the pointer across the interface without clicking
- **THEN** clickable elements highlight as the pointer crosses them, and inert regions do not

### Requirement: Column Sorting by Click

Sortable column headers SHALL sort on click, reverse on a second click of the
same header, and display the active sort column and direction.

#### Scenario: Reversing a sort
- **WHEN** the user clicks the same column header twice
- **THEN** the first click sorts ascending, the second sorts descending, and the header indicates the current direction

#### Scenario: Unsortable column
- **WHEN** the user clicks a header that is not sortable
- **THEN** nothing changes and the header never showed a hover highlight

### Requirement: Mouse Reporting Toggle

The application SHALL provide a way to release mouse reporting so the terminal's
own selection and copy work, SHALL make it available both at start-up and at
runtime from the command palette, and SHALL indicate the current state.

#### Scenario: Copying with the terminal
- **WHEN** the user turns mouse reporting off and drags across the output
- **THEN** the terminal performs its native text selection and the application receives no mouse events

#### Scenario: Indicating the state
- **WHEN** mouse reporting is off
- **THEN** the interface shows that it is off and how to turn it back on

### Requirement: No Mouse-Only Actions

Every action reachable by a click SHALL also be reachable without a mouse.

#### Scenario: Keyboard-only session
- **WHEN** a user completes any workflow using only the keyboard
- **THEN** no step requires a click, and every step's binding is discoverable through the hint bar, the help overlay, the actions menu, or the command palette

### Requirement: Mouse Events Respect Modes

Mouse events SHALL obey the same precedence as key events: while an overlay is
open, clicks outside it SHALL either dismiss it (backdrop) or be ignored, and
SHALL NOT reach the screen behind it.

#### Scenario: Click behind an open dialog
- **WHEN** a modal dialog is open and the user clicks a row behind it
- **THEN** the selection behind the dialog does not change

#### Scenario: Backdrop dismissal
- **WHEN** the user clicks the backdrop of a non-destructive overlay
- **THEN** the overlay closes without acting

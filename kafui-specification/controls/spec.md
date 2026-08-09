# Controls — Keyboard and Mouse Interaction Model

This specification defines how the application is driven. It is normative for
every screen: any feature spec that introduces a user action SHALL express that
action in the terms defined here rather than inventing its own binding.

Unlike the other feature specs, this one is terminal-UI specific — it constrains
input handling, not product features. It is written black-box: it describes what
the user can press, click, and see, not how the event loop is built.

Companion documents, normative as part of this spec:

| Document | Contents |
|---|---|
| [keymap.md](keymap.md) | The complete key table: every reserved key, every context |
| [mouse.md](mouse.md) | The complete mouse interaction table |
| [audit-and-migration.md](audit-and-migration.md) | What the implementation does today, every defect found, and the per-binding migration map |

---

## Design Principles

These are the rationale behind the requirements below. Where a requirement and a
principle appear to conflict, the requirement wins.

1. **A key means one thing.** Across the whole application, a given key is bound
   to a single concept. If a screen has no use for that concept, the key is
   unbound there — it is never repurposed.
2. **Tiers by modifier.** Unmodified keys act on what the user is looking at.
   `Ctrl` keys act on the application. `Shift`+letter is always the "reverse,
   alternate, or wider" variant of the same unmodified letter.
3. **Discoverability over memorisation.** Heritage from k9s gives the app a
   dense single-letter vocabulary. That is kept for fluent users, but no action
   may depend on it: every action is reachable through a visible menu.
4. **Two palettes, not fifty chords.** Anything that would otherwise need an
   obscure chord goes into either the command palette (application-wide) or the
   actions menu (what can I do to this thing). The chord, if any, is displayed
   inside the menu, which is how users learn it.
5. **Mouse and keyboard are peers.** Neither is a degraded mode of the other.
   The actions menu is what makes this cheap: right-click and `a` open the same
   menu, so every keyboard action is automatically mouse-reachable.
6. **Destruction is never one keystroke.** Mutating actions are confirmed, and
   irreversible ones are reached deliberately.
7. **Everything on the arrow keys.** All navigation is `↑`/`↓`, `←`/`→`, and
   `Shift`+`←`/`→` — nothing requires `Fn`, a numeric keypad, or a full-size
   keyboard. `PgUp`/`PgDn`, `Home`/`End` and the vim keys `j/k/h/l`, `g/G`
   remain as unadvertised aliases, so neither a vim user nor a laptop user is
   asked to adapt.

---

## ADDED Requirements

### Requirement: Single Binding Registry

The application SHALL declare every key binding in one registry that names, for
each binding, its action, its context, its help label, and whether it is
destructive. No screen SHALL act on a raw key value that is not declared in the
registry.

#### Scenario: Screen reacts to a key
- **WHEN** any screen handles a key press
- **THEN** the handling is dispatched through the registry entry for that action, and the key that triggered it is the one the registry declares

#### Scenario: Undeclared key
- **WHEN** the user presses a key for which the active context has no registry entry
- **THEN** the application takes no action and produces no error

#### Scenario: Registry is the source of the help text
- **WHEN** the help overlay, the footer hint bar, or an actions menu displays a binding
- **THEN** the displayed key and label are read from the registry entry, so they cannot disagree with the behavior

### Requirement: No Conflicting Bindings Within a Context

Within any single input context, a key SHALL resolve to exactly one action. The
application SHALL detect duplicate bindings and SHALL fail its own start-up
validation rather than shipping an ambiguous binding.

#### Scenario: Duplicate binding introduced
- **WHEN** two actions reachable in the same context declare the same key
- **THEN** binding validation reports the conflicting pair by name and the application refuses to start

#### Scenario: Same key in disjoint contexts
- **WHEN** two actions declare the same key but can never be reachable at the same time (for example, an action inside a modal dialog and an action on a list screen)
- **THEN** validation accepts the pair

### Requirement: Reserved Global Keys

The keys listed as global in [keymap.md](keymap.md) SHALL retain their meaning on
every screen, in every overlay, and in every pane. A screen SHALL NOT rebind,
shadow, or suppress a reserved global key, with the sole exception of text-entry
contexts as defined below.

#### Scenario: Global key on a nested screen
- **WHEN** the user presses a reserved global key on any screen, however deeply nested
- **THEN** the global action runs, and no screen-local action runs for that key

#### Scenario: Screen attempts to reuse a reserved key
- **WHEN** a screen declares an action bound to a reserved global key
- **THEN** binding validation rejects it as a conflict

### Requirement: Modifier Tiers

Bindings SHALL be assigned by tier: unmodified keys act on the focused pane or
its selection; `Ctrl`-modified keys act on the application as a whole and are
never screen-local; `Shift`+letter is reserved for the reverse, alternate, or
broader form of the action bound to the corresponding unmodified letter.

#### Scenario: Shift variant relates to its base key
- **WHEN** an unmodified letter is bound to an action on some screen and the `Shift` variant of that letter is also bound there
- **THEN** the `Shift` variant performs the reverse, alternate, or broader form of the same action rather than an unrelated one

#### Scenario: Ctrl binding is application-wide
- **WHEN** a `Ctrl`-modified key is bound
- **THEN** it performs the same action on every screen

### Requirement: Terminal-Safe Key Selection

The application SHALL NOT bind key combinations that terminals cannot report
distinctly. Specifically, `Ctrl+I`, `Ctrl+M`, `Ctrl+J`, `Ctrl+H` and `Ctrl+[`
SHALL NOT be bound as distinct actions, because terminals deliver them as `Tab`,
`Enter`, `Enter`, `Backspace` and `Esc` respectively. `Ctrl`+`Shift`+letter
SHALL NOT be used, as it is not reportable by many terminal emulators.

#### Scenario: Ambiguous chord declared
- **WHEN** a binding declares one of the ambiguous combinations
- **THEN** binding validation rejects it and names the key it collides with

### Requirement: Input Modes and Key Routing Precedence

The application SHALL be in exactly one input mode at a time, and SHALL route
every key press according to a fixed precedence: **text entry** beats **overlay**
beats **screen** beats **global**.

The modes are:

- **Normal** — a list, table, or content pane has focus; the full key vocabulary applies.
- **Text entry** — a search field, filter field, palette prompt, or form field has focus.
- **Overlay** — a modal dialog, actions menu, or the help overlay is open.

#### Scenario: Typing in a search field
- **WHEN** a text-entry context has focus and the user types printable characters, including characters that are bound to actions in Normal mode
- **THEN** every character is inserted into the field and no bound action runs

#### Scenario: Emergency exit during text entry
- **WHEN** the user presses `Ctrl+C` during text entry
- **THEN** the application exits, because `Ctrl+C` is the single exception to text-entry capture

#### Scenario: Leaving text entry
- **WHEN** the user presses `Esc` during text entry
- **THEN** the field closes, any uncommitted input is discarded, and the previous mode is restored

#### Scenario: Committing text entry
- **WHEN** the user presses `Enter` during text entry
- **THEN** the input is committed, the field closes, and the previous mode is restored with the result applied

#### Scenario: Overlay captures screen keys
- **WHEN** a modal dialog or actions menu is open and the user presses a key bound to a screen action
- **THEN** the overlay handles the key if it has a binding for it, otherwise the key is ignored — it does not reach the screen behind the overlay

#### Scenario: Mode is visible
- **WHEN** the application is in text-entry or overlay mode
- **THEN** the interface indicates it, so the user can tell why ordinary keys are not acting

### Requirement: Escape Unwinds One Level

`Esc` SHALL cancel exactly one level of context per press, in this order:
close the open overlay; else leave text entry; else clear the active
search/filter; else navigate to the parent screen. `Esc` SHALL NOT exit the
application.

#### Scenario: Nested unwinding
- **WHEN** the user is on a nested screen with an active filter and an open dialog, and presses `Esc` three times
- **THEN** the first press closes the dialog, the second clears the filter, and the third returns to the parent screen

#### Scenario: Escape at the top level
- **WHEN** the user presses `Esc` on the top-level screen with nothing open and no active filter
- **THEN** nothing happens and the application does not exit

### Requirement: Quit Is Explicit and Guarded

`q` SHALL quit the application from Normal mode and SHALL NOT mean "back" on any
screen. `Ctrl+C` SHALL quit from any mode. When work would be lost — an unsaved
form, an in-flight mutating operation — quitting SHALL require confirmation.

#### Scenario: Quit from a nested screen
- **WHEN** the user presses `q` in Normal mode on a nested screen
- **THEN** the application quits rather than navigating back

#### Scenario: Quit with unsaved work
- **WHEN** the user presses `q` while a form has unsaved changes
- **THEN** a confirmation dialog appears and the application quits only on confirmation

### Requirement: Universal Navigation Cluster

Every scrollable or selectable pane SHALL respond to the same navigation keys,
and those keys SHALL be reachable without a `Fn` modifier on a laptop or 60%
keyboard. Navigation SHALL therefore be expressed on the arrow keys alone:
`↑`/`↓` move one row or scroll one line, `←`/`→` move one page — the next page
of a paginated list, or one screenful of a content pane — and `Shift`+`←`/`→`
jump to the first or last item. `PgUp`/`PgDn`, `Home`/`End`, `k`/`j`, `h`/`l`
and `g`/`G` SHALL be accepted as aliases.

No action SHALL be reachable only through `PgUp`, `PgDn`, `Home` or `End`.

#### Scenario: Same keys on every pane
- **WHEN** the user moves between screens and panes
- **THEN** the navigation keys behave identically in each, with no screen introducing its own scrolling keys

#### Scenario: One meaning for the big step
- **WHEN** the user presses `→` on a paginated list, and again on a long content pane
- **THEN** the list advances one page and the pane scrolls one screenful, so the user need not know which kind of pane has focus

#### Scenario: Laptop keyboard is sufficient
- **WHEN** a user works on a keyboard with no dedicated `PgUp`, `PgDn`, `Home` or `End` keys
- **THEN** every navigation action is reachable with the arrow keys and `Shift`, without pressing `Fn`

#### Scenario: Aliases are not advertised
- **WHEN** the footer or help displays navigation bindings
- **THEN** the arrow keys are shown, with the `PgUp`/`PgDn`/`Home`/`End` and vim aliases shown only in the expanded help

#### Scenario: Long lines wrap rather than scroll sideways
- **WHEN** a content pane holds lines wider than the pane
- **THEN** the user reads them by toggling soft wrap, and `←`/`→` continue to page rather than scrolling horizontally

### Requirement: Focus Cycling with Tab

`Tab` and `Shift+Tab` SHALL move focus forward and backward between the focusable
regions of the current screen — panes, tab strips, and form fields — and SHALL
NOT navigate between screens, switch pages of a list, or change a data mode.

#### Scenario: Cycling panes
- **WHEN** a screen has two content panes and the user presses `Tab`
- **THEN** focus moves to the next pane, the focused pane is visually distinguished, and subsequent keys act on it

#### Scenario: Backward cycling
- **WHEN** the user presses `Shift+Tab`
- **THEN** focus moves to the previous focusable region — the exact inverse of `Tab`

#### Scenario: Wrapping
- **WHEN** focus is on the last focusable region and the user presses `Tab`
- **THEN** focus wraps to the first

### Requirement: Numeric Tab Selection

On any screen presenting a tab strip, `1` through `9` SHALL select the tab at
that ordinal position, and SHALL do nothing when no tab exists at that position.

#### Scenario: Jumping to a tab
- **WHEN** a screen shows three tabs and the user presses `2`
- **THEN** the second tab becomes active

### Requirement: Activate and Drill Down

`Enter` SHALL activate the focused item: open the detail view of a selected row,
confirm the focused dialog button, or apply the focused control. `Enter` SHALL
NOT perform a destructive or mutating action directly.

#### Scenario: Opening a row
- **WHEN** a row is selected in a list and the user presses `Enter`
- **THEN** the corresponding detail screen opens

#### Scenario: Row with no detail view
- **WHEN** the selected row has no detail screen
- **THEN** `Enter` opens the actions menu for that row instead of doing nothing

### Requirement: Command Palette

The application SHALL provide a command palette, opened with `:` or `Ctrl+P`,
that offers fuzzy search over every navigable destination (resource lists,
dashboards, configuration screens) and every application-level command, executes
the chosen entry, and shows each entry's direct key binding where one exists.

#### Scenario: Navigating by name
- **WHEN** the user opens the palette and types a partial resource name
- **THEN** matching destinations are listed, and choosing one navigates there

#### Scenario: Learning a binding
- **WHEN** the palette lists an entry that also has a direct key binding
- **THEN** the binding is shown alongside the entry

#### Scenario: Dismissing
- **WHEN** the user presses `Esc` with the palette open
- **THEN** the palette closes and nothing is executed

### Requirement: Contextual Actions Menu

The application SHALL provide an actions menu, opened with `a` or by right-click,
listing every action available for the currently focused item — the selected row,
the focused pane, or the current screen when nothing is selected. Each entry
SHALL display its direct key binding where one exists, and entries the user may
not perform SHALL be shown disabled with the reason.

#### Scenario: Discovering actions
- **WHEN** the user selects a row and presses `a`
- **THEN** a menu lists every action applicable to that row, each with its key binding

#### Scenario: Permission-gated entry
- **WHEN** an action is unavailable because the active permission profile denies it, or because the cluster is read-only
- **THEN** the entry is listed but disabled, and the reason is stated

#### Scenario: Mouse parity
- **WHEN** the user right-clicks a row
- **THEN** the same menu opens for that row, and the row also becomes the selection

#### Scenario: Every action is listed
- **WHEN** any action is available on a screen
- **THEN** it appears in the actions menu, the footer hint bar, or the help overlay — no action is reachable only by an undocumented key

### Requirement: Search and Filter

`/` SHALL open an incremental search over the focused pane's content. On a list,
it filters rows; on a content pane, it highlights and counts matches, with `n`
and `N` moving to the next and previous match. The active query SHALL remain
visible while it is in effect, and `Esc` SHALL clear it.

#### Scenario: Filtering a list
- **WHEN** the user presses `/` on a list and types
- **THEN** rows are filtered as each character is typed, and the pane shows the active query

#### Scenario: Searching content
- **WHEN** the user searches inside a content pane
- **THEN** matches are highlighted, the match count and current position are shown, and `n`/`N` move between matches

#### Scenario: Query survives redraw
- **WHEN** the pane re-renders while a search is active — for example because new data arrived
- **THEN** the query, its matches, and the scroll position are preserved

### Requirement: Selection and Multi-Selection

`Space` SHALL toggle selection of the focused row on lists that support acting on
several rows at once, `Ctrl+A` SHALL select all rows currently visible after
filtering, and `Esc` SHALL clear the selection. When a selection exists, actions
SHALL apply to the selection rather than to the cursor row, and the interface
SHALL state how many rows are selected.

#### Scenario: Selecting rows
- **WHEN** the user presses `Space` on several rows
- **THEN** each is marked selected and the count of selected rows is displayed

#### Scenario: Acting on a selection
- **WHEN** the user runs an action while rows are selected
- **THEN** the action applies to all selected rows and the confirmation names the count

#### Scenario: Space is never page-down
- **WHEN** the user presses `Space` on any scrollable pane
- **THEN** the view does not scroll — paging is `→` only

### Requirement: Destructive Actions Are Confirmed

Every action that deletes, resets, purges, recreates, or otherwise irreversibly
changes cluster state SHALL require confirmation in a dialog that names the
target and, for multi-row operations, the count. Such actions SHALL NOT be
reachable by a single unmodified keystroke without that dialog.

#### Scenario: Deleting
- **WHEN** the user triggers a delete
- **THEN** a confirmation dialog names what will be deleted, defaults to the cancelling choice, and performs the deletion only on explicit confirmation

#### Scenario: Cancelling
- **WHEN** the user dismisses the confirmation with `Esc` or by choosing cancel
- **THEN** nothing is changed

#### Scenario: Denied action
- **WHEN** the user triggers a mutating action they are not permitted to perform
- **THEN** the action is refused with a message naming the reason, and no confirmation dialog is shown

### Requirement: Always-Visible Contextual Hint Bar

The application SHALL display a hint bar showing the bindings most relevant to
the current context, updated as focus, mode, and selection change. Bindings that
are unavailable in the current context SHALL NOT be shown as available.

#### Scenario: Hints follow focus
- **WHEN** focus moves to a different pane, or an overlay opens
- **THEN** the hint bar updates to the bindings that now apply

#### Scenario: Hints reflect permissions
- **WHEN** an action is denied by permissions or read-only mode
- **THEN** its hint is either omitted or shown in a disabled style, never as an available action

#### Scenario: Hints are clickable
- **WHEN** the user clicks a hint in the hint bar
- **THEN** the corresponding action runs, exactly as if its key had been pressed

### Requirement: Full Help Overlay

`?` SHALL open a help overlay listing all bindings available in the current
context, grouped by category, searchable with `/`, and dismissible with `?` or
`Esc`. The overlay SHALL distinguish global bindings from screen-local ones and
SHALL show the vim aliases.

#### Scenario: Opening help
- **WHEN** the user presses `?` on any screen
- **THEN** the overlay lists the global bindings and the bindings specific to that screen, grouped and labelled

#### Scenario: Searching help
- **WHEN** the user types a search term in the help overlay
- **THEN** only matching entries remain listed

#### Scenario: Help is context-aware
- **WHEN** the user opens help from two different screens
- **THEN** the screen-local section differs while the global section is identical

### Requirement: Mouse Support Is Complete and Consistent

Every interactive element SHALL respond to the mouse as defined in
[mouse.md](mouse.md). Left-click selects or focuses and is never destructive;
double-click activates; right-click opens the actions menu; the wheel scrolls the
pane under the pointer without changing the selection.

#### Scenario: Click selects, does not activate
- **WHEN** the user single-clicks a row
- **THEN** the row becomes the selection and no navigation occurs

#### Scenario: Double-click activates
- **WHEN** the user double-clicks a row, or clicks a row that is already selected
- **THEN** the row is activated, exactly as `Enter` would

#### Scenario: Wheel does not move the selection
- **WHEN** the user scrolls the wheel over a list
- **THEN** the view scrolls and the selected row is unchanged

#### Scenario: Clicking a column header sorts
- **WHEN** the user clicks a sortable column header
- **THEN** the list sorts by that column; clicking the same header again reverses the direction, and the header shows the active sort

#### Scenario: All chrome is clickable
- **WHEN** the user clicks a tab, a sidebar entry, a breadcrumb segment, a pagination control, a dialog button, or a hint in the hint bar
- **THEN** the corresponding action runs

#### Scenario: Hover feedback
- **WHEN** the pointer moves over a clickable element
- **THEN** that element is visually distinguished, so what is clickable is discoverable without clicking

### Requirement: Mouse Reporting Can Be Disabled

Because capturing the mouse prevents the terminal's own text selection and copy,
the application SHALL provide a way to release the mouse — a toggle available
from the command palette and a start-up option — and SHALL indicate when mouse
reporting is off. With mouse reporting off, the application SHALL remain fully
usable by keyboard.

#### Scenario: Releasing the mouse to copy text
- **WHEN** the user disables mouse reporting
- **THEN** the terminal's native selection works over the application's output, and the interface indicates that mouse input is off

#### Scenario: Keyboard completeness
- **WHEN** mouse reporting is off
- **THEN** every action remains reachable by keyboard, through direct bindings, the actions menu, or the command palette

### Requirement: Keyboard Reachability of Every Action

No action SHALL be reachable only by mouse. Every action offered by a click
target SHALL also be reachable by a direct binding, the actions menu, or the
command palette.

#### Scenario: Auditing a click target
- **WHEN** any element responds to a click
- **THEN** the action it performs is also reachable without a mouse

### Requirement: User-Defined Key Bindings

The application SHOULD let users override bindings through configuration,
validating overrides against the same conflict and terminal-safety rules as the
defaults, reporting invalid overrides without preventing start-up, and displaying
the effective bindings in the help overlay and hint bar.

#### Scenario: Valid override
- **WHEN** the user rebinds an action in configuration and restarts
- **THEN** the new key performs the action, the default key no longer does, and the help and hints show the new key

#### Scenario: Conflicting override
- **WHEN** an override collides with another binding in the same context or with a reserved global key
- **THEN** the application starts with the override rejected and reports which override was rejected and why

#### Scenario: Resetting
- **WHEN** the user removes their overrides
- **THEN** the default bindings apply again

---

## Cross-Cutting Constraints

- **Permission awareness**: bindings for actions the user may not perform are hidden or disabled in the hint bar, the help overlay, and the actions menu, and the underlying operation remains refused regardless of how it was triggered (see feature 10).
- **Read-only mode**: when the active cluster is read-only, every mutating binding is presented as disabled with that reason.
- **Feature-conditional bindings**: bindings belonging to an optional integration exist only when that integration is available for the active cluster; they are absent, not inert, when it is not.
- **No silent no-ops**: a binding that is present but cannot act states why rather than doing nothing.

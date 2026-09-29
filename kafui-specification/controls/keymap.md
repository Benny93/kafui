# Controls — Normative Key Map

This is the complete, normative key table referenced by
[spec.md](spec.md). Any binding not listed here does not exist.

Notation: `Ctrl+X` is control-modified, `X` alone is the plain key, and an
uppercase letter means the shifted key (`S` is `Shift`+`s`). "Alias" columns list
keys accepted but not advertised in the hint bar.

---

## 1. Reserved global keys

These work on every screen, in every pane, and inside every overlay. No screen
may rebind them.

| Key         | Alias    | Action                                                                               |
| ----------- | -------- | ------------------------------------------------------------------------------------ |
| `?`         | `F1`     | Open/close the help overlay                                                          |
| `:`         | `Ctrl+P` | Open the command palette                                                             |
| `a`         |          | Open the actions menu for the focused item                                           |
| `/`         |          | Search or filter the focused pane                                                    |
| `Esc`       |          | Cancel one level: overlay → text entry → active filter → parent screen               |
| `Enter`     |          | Activate the focused item                                                            |
| `q`         |          | Quit (guarded when work would be lost)                                               |
| `Ctrl+C`    |          | Quit immediately, from any mode                                                      |
| `Tab`       |          | Focus the next pane, tab, or field                                                   |
| `Shift+Tab` |          | Focus the previous pane, tab, or field                                               |
| `1`…`9`     |          | Select the tab at that position                                                      |
| `Space`     |          | Toggle selection of the focused row                                                  |
| `Ctrl+A`    |          | Select all rows visible after filtering                                              |
| `r`         | `F5`     | Refresh — refetch the current view's data, or run the current statement in an editor |
| `c`         | `y`      | Copy the focused row, cell, or pane to the clipboard                                 |
| `Ctrl+E`    |          | Export the current list as CSV                                                       |
| `Ctrl+B`    |          | Show/hide the sidebar                                                                |
| `Ctrl+N`    |          | Create a new object of the kind the current screen lists                             |

### Navigation cluster

Identical on every scrollable or selectable pane.

All navigation is on the arrow keys. `PgUp`, `PgDn`, `Home` and `End` are
accepted but never advertised: laptop and 60% keyboards reach them only through
`Fn`, so no binding may depend on them.

| Key                   | Alias                      | Action                                                                                |
| --------------------- | -------------------------- | ------------------------------------------------------------------------------------- |
| `↑` / `↓`             | `k` / `j`, wheel           | Move one row, or scroll one line                                                      |
| `←` / `→`             | `h` / `l`, `PgUp` / `PgDn` | Move one page — the next page of a paginated list, or one screenful of a content pane |
| `Shift+←` / `Shift+→` | `g` / `G`, `Home` / `End`  | Jump to the first / last item or page                                                 |
| `n` / `N`             |                            | Next / previous search match                                                          |

The three pairs form one scale: unmodified arrows are the small step, left/right
is the big step, and `Shift` is all the way to the end. `←`/`→` mean the same
thing on a paginated list and on a long content pane, so the user does not have
to know which kind of pane has focus.

Long lines in content panes are handled by soft wrap (`w`), and tables fit their
pane by flexing their columns. Nothing scrolls horizontally, which is what keeps
`←`/`→` free to mean "one page" everywhere.

`Shift`+arrow is reportable by current terminal emulators but not by all older
ones. It is used only for first/last — the least frequent navigation action —
and `Home`/`End` and `g`/`G` remain as aliases, so a terminal that swallows it
costs the user nothing.

### Debug builds only

| Key        | Action                     |
| ---------- | -------------------------- |
| `F3`       | Save a screenshot          |
| `Shift+F3` | Save a redacted screenshot |
| `F12`      | Toggle the debug overlay   |

---

## 2. Context: list and table panes

Applies to every resource list: topics, consumer groups, schemas, ACLs, quotas,
brokers, connectors, connect clusters, contexts, message lists.

| Key      | Action                                      | Notes                                          |
| -------- | ------------------------------------------- | ---------------------------------------------- |
| `s`      | Sort by the next column                     | Cycles through sortable columns                |
| `S`      | Reverse the sort direction                  | The `Shift` variant of `s`                     |
| `d`      | Delete the selection                        | Always confirmed; applies to all selected rows |
| `e`      | Edit the selection                          | Opens the edit form for the focused row        |
| `i`      | Toggle inclusion of internal/system objects | Only where such objects exist                  |
| `Ctrl+N` | Create a new object of this kind            |                                                |
| `Ctrl+E` | Export the list as CSV                      | Respects the active filter                     |

Everything else a list can do is in the actions menu (`a`).

---

## 3. Context: content panes

Applies to every pane rendering structured or long-form text: message key and
value, schema content, broker and topic configuration, connector configuration,
query results, log output.

| Key         | Action                                                                        |
| ----------- | ----------------------------------------------------------------------------- |
| `f`         | Cycle the display format of the focused pane (for example raw → pretty → hex) |
| `w`         | Toggle soft wrapping                                                          |
| `m`         | Show/hide the metadata pane                                                   |
| `/` `n` `N` | Search within the pane and move between matches                               |
| `c`         | Copy the focused pane's content                                               |

---

## 4. Context: text entry

Active while a search field, filter, palette prompt, or form field has focus.
All keys not listed insert literal characters.

| Key         | Action                                                          |
| ----------- | --------------------------------------------------------------- |
| `Enter`     | Commit and leave text entry                                     |
| `Esc`       | Discard and leave text entry                                    |
| `Tab`       | Accept the current completion, or move to the next field        |
| `Shift+Tab` | Move to the previous field                                      |
| `↑` / `↓`   | Move through completions or history                             |
| `Ctrl+U`    | Clear the field                                                 |
| `Ctrl+W`    | Delete the previous word                                        |
| `Ctrl+C`    | Quit the application — the only binding that escapes text entry |

---

## 5. Context: overlays

Applies to modal dialogs, the actions menu, the command palette, and the help
overlay. Overlays capture all keys except `Ctrl+C`.

| Key                 | Action                                                 |
| ------------------- | ------------------------------------------------------ |
| `↑` / `↓`           | Move between entries or buttons                        |
| `Tab` / `Shift+Tab` | Move between buttons or fields                         |
| `Enter`             | Confirm the focused entry or button                    |
| `Esc`               | Dismiss without acting                                 |
| `y` / `n`           | Confirm / cancel, in confirmation dialogs only         |
| mnemonic letter     | Run the menu entry whose mnemonic letter it is         |
| `/`                 | Filter the entries of a menu, palette, or help overlay |

---

## 6. Screen-specific promoted bindings

A screen may promote at most four of its actions to a direct binding. Every other
action lives in the actions menu with a mnemonic. Promoted bindings must come
from the unassigned pool and are listed here in full.

### Message browsing (topic screen)

| Key                 | Action                                                     |
| ------------------- | ---------------------------------------------------------- |
| `p`                 | Pause / resume consumption                                 |
| `f`                 | Cycle the payload display format                           |
| `m`                 | Show/hide the message metadata pane                        |
| `x`                 | Expand the highlighted row inline (full, pretty value)     |
| `Shift+↑`/`Shift+↓` | Scroll the expanded row's content (also: wheel over it)    |
| `Ctrl+N`            | Produce a message                                          |

Actions menu: seek mode, partition selection, saved filters, projections,
reproduce message, topic settings, topic analysis, partition count, replication
factor, purge messages, recreate topic, delete topic.

### Message detail

| Key         | Action                                 |
| ----------- | -------------------------------------- |
| `f`         | Cycle the payload display format       |
| `w`         | Toggle soft wrapping                   |
| `1` `2` `3` | Content / Headers / Metadata tab       |
| `Tab`       | Switch between the key and value panes |

Actions menu: export message, copy key, copy value, copy as JSON, reproduce.

### Schema registry

| Key      | Action                                        |
| -------- | --------------------------------------------- |
| `1`…`n`  | Versions / Content / Compatibility / Diff tab |
| `Ctrl+N` | Register a new version                        |
| `d`      | Delete the focused version                    |
| `F5`     | Run a compatibility check                     |

Actions menu: run a compatibility check, diff against another version, set
compatibility level, copy schema, delete subject. (A compatibility check does
not get `F5`: that is the refresh alias, and one key may not mean two things.)

### Kafka Connect

| Key      | Action                                  |
| -------- | --------------------------------------- |
| `p`      | Pause / resume the connector            |
| `1`…`n`  | Overview / Config / Tasks / Plugins tab |
| `Ctrl+N` | Create a connector                      |
| `d`      | Delete the connector                    |

Actions menu: restart connector, restart task, stop, resume, reset offsets,
validate configuration.

### Brokers

| Key     | Action                                         |
| ------- | ---------------------------------------------- |
| `1`…`n` | Overview / Configuration / Log directories tab |
| `e`     | Edit the focused configuration entry           |

Actions menu: view partition distribution, view skew, per-broker metrics.

### Consumer groups

| Key | Action           |
| --- | ---------------- |
| `d` | Delete the group |

Actions menu: reset offsets (earliest / latest / timestamp / specific), delete
offsets, view member assignments.

### ACLs and quotas

| Key      | Action                     |
| -------- | -------------------------- |
| `Ctrl+N` | Create a binding           |
| `d`      | Delete the focused binding |
| `Ctrl+E` | Export as CSV              |

Actions menu: consumer / producer / stream-app convenience flows, sync from CSV,
upsert quota, delete quota.

### Streaming SQL (ksqlDB)

| Key   | Action                                       |
| ----- | -------------------------------------------- |
| `F5`  | Execute the statement                        |
| `Tab` | Move between the editor and the results pane |

Actions menu: list streams, list tables, clear results, cancel a running query.

### Cluster management and configuration

These screens are reached from the sidebar and the command palette; they have no
direct jump keys. Their in-screen actions follow the list and content-pane
tables above, with `F2` saving a multi-field form.

---

## 7. Explicitly unbound

Keys that must remain unbound, with the reason.

| Key                                   | Reason                                                                           |
| ------------------------------------- | -------------------------------------------------------------------------------- |
| `Ctrl+I`                              | Terminals deliver it as `Tab`                                                    |
| `Ctrl+M`, `Ctrl+J`                    | Terminals deliver them as `Enter`                                                |
| `Ctrl+H`                              | Terminals deliver it as `Backspace`                                              |
| `Ctrl+[`                              | Terminals deliver it as `Esc`                                                    |
| `Ctrl+S`, `Ctrl+Q`                    | Software flow control; can freeze terminals that have not disabled XON/XOFF      |
| `Ctrl+Z`                              | Job control; the shell suspends the process                                      |
| `Ctrl`+`Shift`+letter                 | Not reportable by many terminal emulators                                        |
| `Shift`+letter as a standalone action | Reserved for the variant of the corresponding lowercase key                      |
| `Ctrl+D`                              | Reads as end-of-input; too close to "delete" to be safe, and `d` covers deletion |

---

## 8. Free pool

Unassigned and available for future promoted bindings, in preference order:

`b` `o` `t` `u` `v` `z` `B` `D` `E` `F` `I` `L` `M` `O` `P` `R` `T` `U` `V` `W` `X` `Y` `Z` `F2` `F4` `F6`…`F11`

Uppercase letters should be taken only as the `Shift` variant of an already-bound
lowercase key, as required by the modifier-tier rule.

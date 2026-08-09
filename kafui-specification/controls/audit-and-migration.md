# Controls — Current-State Audit and Migration Map

This document records what the implementation does today, the defects that
motivated [spec.md](spec.md), and how each existing binding maps onto the
specified model. It is descriptive, not normative — but it is the checklist for
bringing the implementation into conformance.

Audit performed against the tree at the time of writing. File references point at
the code that establishes each finding.

---

## 1. Where bindings live today

Bindings are declared in three places at once, and the three disagree.

| Source | What it holds | Status |
|---|---|---|
| `pkg/ui/keys/keys.go` | Six key maps (Global, Main, Topic, Detail, ResourceDetail, Search) with help text | Partially authoritative. Some entries are never matched at runtime. |
| The shell, `pkg/ui/template/ui/reusable_app.go` | A literal `switch msg.String()` over `ctrl+c`/`q`, `ctrl+s`/`t`, `ctrl+r`, `ctrl+d`, `?` | Authoritative, and takes precedence over everything below it |
| Each page | Literal `case "…"` switches | Authoritative in practice; roughly 120 distinct literals across the pages |

Consequence: the help text rendered from `keys.go` describes bindings that the
shell intercepts first, so the help can be read as wrong without any page being
individually wrong.

---

## 2. Defects found

### D1 — The shell handles a key and then forwards it (high)

`reusable_app.go` matches `ctrl+d`, `ctrl+r`, `ctrl+s`/`t`, `q` and `?`, appends
its command, and then falls through to `a.content.Update(msg)`. The page sees the
same key.

Observable results:

- `Ctrl+D` toggles the debug overlay **and** deletes the selected topic, group, ACL or quota (`pkg/ui/pages/main/providers.go`).
- `Ctrl+R` refreshes the sidebar **and** recreates the selected topic — a destructive operation (`main/providers.go`), and separately means "recreate topic" on the topic screen (`pkg/ui/pages/topic/keys.go`).
- `t` toggles the sidebar **and** is declared as "switch resource" in `MainKeyMap.SwitchResource` (`":", "t"`), **and** opens topic analysis on the topic screen, **and** does something else again on the connector screen.

Spec: single binding registry, no conflicting bindings within a context, one
handler per key.

### D2 — `q` means two different things (high)

The shell quits on `q`. `DetailKeyMap.Back` and `ResourceDetailKeyMap.Back`
declare `esc, q` as "back". The shell wins, so `q` quits from a detail screen
while the help says it goes back.

Spec: `q` always quits; `Esc` always goes back.

### D3 — `Tab` has four incompatible meanings (high)

| Where | Meaning |
|---|---|
| `GlobalKeyMap.NextPage` | "next page" |
| `TopicKeyMap.SwitchMode` | cycle Newest / Oldest / Live |
| Message detail | switch focus between key and value panes — and `Shift+Tab` moves to the **next** tab, not the previous |
| Broker, connector screens | next tab |
| Forms, search | next field / accept completion |

Spec: `Tab` and `Shift+Tab` cycle focus forward and backward, only.

### D4 — `Space` has three meanings (medium)

Page-down on the topic and detail screens (`TopicKeyMap.PageDown`,
`DetailKeyMap.PageDown`), toggle topic selection on the main screen, and
pause/resume consumption in the recorded demos.

Spec: `Space` toggles row selection; paging is `→`.

### D4b — Navigation depends on keys laptops do not have (medium)

`PgUp`, `PgDn`, `Home` and `End` carry the paging and jump-to-end actions across
every key map. Mac laptop keyboards and 60% keyboards have none of them; they
require `Fn`+arrow, which is a two-handed gesture for the most frequent action in
a list-driven application.

Spec: navigation lives on the arrow keys — `↑`/`↓` one row, `←`/`→` one page,
`Shift`+`←`/`→` first/last — with the dedicated keys demoted to unadvertised
aliases.

### D5 — `Ctrl+I` is bound (high)

`main/providers.go` binds `ctrl+i` to "open the ACL sync form". Terminals deliver
`Ctrl+I` as `Tab`, so this fires on every `Tab` press while the ACL list is
shown, and can never be triggered deliberately.

Spec: terminal-safe key selection; `Ctrl+I` explicitly unbound.

### D6 — `Ctrl+S` used for save (medium)

Used on the connector screen and the schema register form. `Ctrl+S` is XOFF; on
terminals that have not disabled software flow control it freezes output rather
than saving.

Spec: `Ctrl+S` explicitly unbound; `F5` runs, `F2` saves, `Enter` confirms.

### D7 — Destructive actions on bare keys, undiscoverable (high)

The topic screen binds `Ctrl+P` purge-all-messages, `Ctrl+R` recreate topic and
`Ctrl+D` delete topic, alongside fourteen further single-letter actions —
`C o s E t + F S # P Y L X` — none of which appear in `TopicKeyMap`, so none
appear in the help overlay or the hint bar. The same pattern holds on the schema
detail screen (`y v d r c D x [ ]`) and the connector screen
(`r p u s R z t T f 1 2 3 4`).

Spec: no action reachable only by an undocumented key; destructive actions are
confirmed and reached through the actions menu.

### D8 — Uppercase letters used as unrelated jump keys (medium)

`T` toggles the theme, `C` opens the cluster dashboard, `K` opens ksqlDB — none
of which is the "reverse or alternate" form of the corresponding lowercase key,
and all of which squat on letters pages might want.

Spec: `Shift`+letter is reserved for the variant of the lowercase action; screen
jumps go through the command palette.

### D9 — `r` versus `R` versus `Ctrl+R` (medium)

`TopicKeyMap.Retry` is `r` ("retry connection"), `TopicKeyMap.Refresh` is `R`
("refresh messages"), the shell's `Ctrl+R` refreshes the sidebar, the connector
screen uses `r` for one refresh concept and `R` for another. Four keys, one idea.

Spec: `r` (alias `F5`) refreshes the current view; nothing else.

### D10 — Search state destroyed on redraw (fixed, retain as a regression rule)

The message detail screen re-set its viewers' content on every frame, which reset
the active search and the scroll position. The same failure mode exists anywhere
a pane is rebuilt from source data each render.

Spec: an active query, its matches, and the scroll position survive a redraw.

---

## 3. Mouse coverage today

Mouse reporting is enabled once, in `pkg/ui/kafui.go` (`tea.WithMouseCellMotion`).
Only six regions are click-mapped, via `zone.Mark`:

| Marked region | Behavior |
|---|---|
| Sidebar items | Click switches resource |
| Main resource table | Click selects a row. Wheel **moves the selection**, not the viewport |
| Topic message table | Click selects **and opens** the message |
| Message detail tab strip | Click switches tab |
| Schema content pane | Marked, no click behavior |
| Confirmation dialog | Receives mouse events; no click targets |

Gaps against [mouse.md](mouse.md):

- **Inconsistent click semantics.** The main table selects on click; the message table selects *and activates*. One of the two is wrong on every screen.
- **Wheel moves the selection** on the main table and the topic table, instead of scrolling the viewport.
- **No right-click anywhere** — so there is no mouse route to any action.
- **No double-click**, and no click-on-selected-row activation.
- **No hover feedback** at all.
- **Not clickable:** column headers (no click-to-sort exists anywhere), pagination controls, breadcrumbs, the hint bar, help overlay entries, dialog buttons, form fields and checkboxes, notifications, scrollbars, and the tab strips of the broker, connector and schema screens.
- **No screens are click-mapped at all:** clusters, ksqlDB, metrics, broker, connector, resource detail, application config, cluster wizard, error pages.
- **No way to release the mouse**, so the terminal's own text selection is unavailable for the whole session.

---

## 4. Migration map

Each row states the current binding, the target, and the disposition. "Menu"
means the action loses its direct key and is reached through the actions menu
(`a`) with a mnemonic.

### Global

| Today | Target | Disposition |
|---|---|---|
| `?` help | `?` / `F1` | Keep |
| `q`, `Ctrl+C` quit | `q`, `Ctrl+C` | Keep; remove `q` = "back" from the detail key maps |
| `Esc` back | `Esc` | Keep; formalise as one-level unwind |
| `Tab` / `Shift+Tab` next/prev page | `Tab` / `Shift+Tab` focus cycling | **Change** |
| `/` search | `/` | Keep |
| `:` resource picker | `:` / `Ctrl+P` command palette | Widen to all destinations and commands |
| `t`, `Ctrl+S` sidebar | `Ctrl+B` | **Change** — frees `t`, removes the `Ctrl+S` flow-control hazard |
| `Ctrl+R` refresh sidebar | `r` / `F5` | **Change** — merge with the two other refresh keys |
| `Ctrl+D` debug overlay | `F12` | **Change** — frees `Ctrl+D`, which currently double-fires with delete |
| `T` toggle theme | Palette (`:theme`) | **Remove key** |
| `C` clusters | Palette / sidebar | **Remove key** |
| `K` ksqlDB | Palette / sidebar | **Remove key** |
| `Ctrl+T` metrics | Palette / sidebar | **Remove key** |
| `Ctrl+G` app config | Palette / sidebar | **Remove key** |
| `Ctrl+W` cluster wizard | Palette | **Remove key** — `Ctrl+W` is "delete word" in text entry |
| `F3`, `Shift+F3` screenshot | Unchanged | Keep, debug builds only |

### Resource lists

| Today | Target | Disposition |
|---|---|---|
| `Enter` select | `Enter` | Keep |
| `↑↓` `k j` | Unchanged | Keep |
| `←→` `h l` previous/next page | `←`/`→`, widened to mean "one page" on content panes too | **Change** |
| `PgUp` `PgDn` screenful | `←`/`→`; kept as unadvertised aliases | **Change** — requires `Fn` on laptop and 60% keyboards |
| `Home` `End`, `g` `G` first/last | `Shift+←`/`Shift+→`; kept as unadvertised aliases | **Change** — same `Fn` problem |
| `s` sort column, `S` sort direction | Unchanged | Keep — this is already the specified pattern |
| `c` copy row | `c` (alias `y`) | Keep |
| `Space` toggle selection | Unchanged | Keep; remove `Space` = page-down elsewhere |
| `Ctrl+A` select all | Unchanged | Keep |
| `Ctrl+E` export CSV | Unchanged | Keep |
| `n` new topic / ACL / quota / connector | `Ctrl+N` | **Change** — frees `n` for next-search-match |
| `Ctrl+N` clone topic | Menu | **Change** |
| `Ctrl+D` delete | `d` | **Change** |
| `Ctrl+R` recreate topic | Menu | **Change** — destructive, must not sit on a shell-shadowed chord |
| `Ctrl+P` purge messages | Menu | **Change** — destructive |
| `Ctrl+I` ACL sync | Menu | **Change** — currently unreachable and misfires on `Tab` |
| `e` edit quota | `e` | Keep, generalise to "edit the selection" |
| `i` hide internal topics | `i` | Keep |
| `f` consumer-group state filter | Menu, or a facet of `/` | **Change** — `f` is the display-format key |
| `p` ACL pattern filter | Menu, or a facet of `/` | **Change** — `p` is pause/resume |

### Message browsing (topic screen)

| Today | Target | Disposition |
|---|---|---|
| `p` pause/resume (`TopicKeyMap.Pause`) | `p` | Keep |
| `Space` pause/resume | `p` only | **Change** |
| `r` retry, `R` refresh | `r` / `F5` | **Merge** |
| `Tab` switch seek mode | Menu | **Change** — `Tab` is focus cycling |
| `f` format | `f` | Keep |
| `h` headers, `m` metadata | `m` metadata pane | **Merge**; `h` is a navigation alias |
| `c` copy key, `v` copy value | `c` copies the focused pane | **Change** — `Tab` selects which pane |
| `C o s E t + F S # P Y L X` | Menu | **Change** — all twelve |
| `Ctrl+P` clear all, `Ctrl+R` recreate, `Ctrl+D` delete | Menu, confirmed | **Change** |
| `P` produce | `Ctrl+N` | **Change** |

### Message detail

| Today | Target | Disposition |
|---|---|---|
| `Shift+Tab` next tab | `1` `2` `3`, `Tab` for panes | **Change** — currently inverted |
| `tab` switch key/value focus | `Tab` | Keep |
| `f` format, `w` wrap, `/` `n` `N` search | Unchanged | Keep |
| `c` copy | `c` | Keep |
| `e` export | Menu | **Change** |
| `r` reload schema info | `r` | Keep, as the general refresh |

### Schema, connect, broker, consumer-group screens

| Today | Target | Disposition |
|---|---|---|
| `1`…`4` tab selection | Unchanged | Keep — promote to a universal rule |
| `tab` next tab | `Tab` focus cycling, `1`…`9` for tabs | **Change** |
| `Ctrl+S` save | `F2`, or `Enter` on the submit button | **Change** |
| `Ctrl+K` compatibility / register | Menu | **Change** |
| `y v d r c D x [ ]` (schema) | `d` delete, `c` copy, `r` refresh; rest to menu | **Change** |
| `r p u s R z t T f` (connect) | `p` pause/resume, `r` refresh, `d` delete; rest to menu | **Change** |
| `e` edit config (broker) | `e` | Keep |
| `a` add ACL flows (consumer group) | Menu — `a` opens the menu itself | **Change** |

---

## 5. Conformance checklist

An implementation conforms when all of the following hold. Each is mechanically
checkable. Status is against the tree at the time of writing.

| # | Requirement | Status |
|---|---|---|
| 1 | No file outside the binding registry compares a raw key string | Done, and guarded by `TestNoPageComparesRawKeyStrings`, which walks `pkg/ui` and fails on any key-shaped `case` literal |
| 2 | Binding validation runs in a test and fails on duplicates, reserved-global shadowing, and forbidden keys | `pkg/ui/keys/registry_test.go` |
| 3 | The shell handles a key or forwards it, never both | Done |
| 4 | Every registry entry carries a help label, a category, and a destructive flag | Enforced by `Validate` |
| 5 | Every destructive entry routes through the confirmation dialog | Done via the existing guard + dialog |
| 6 | Every screen exposes an actions menu covering everything it can do | Done for main, topic, message detail, connector, broker, schema, consumer group, ksqlDB, cluster wizard |
| 7 | Every action in the registry is reachable without a mouse | Done |
| 8 | Every click target in `mouse.md` exists, with no gesture outside the vocabulary | Rows, headers, tabs, sidebar, breadcrumbs, hint bar, menu rows |
| 9 | Wheel events change no selection anywhere | Done |
| 9b | No action is reachable only through `PgUp`, `PgDn`, `Home` or `End` | `TestNoActionIsReachableOnlyByAFnKey` |
| 10 | Mouse reporting can be toggled off at runtime, with the state visible | Palette entry + notification |
| 11 | An active search survives a redraw on every pane that supports search | Done |
| 12 | The hint bar and help overlay are generated from the registry | Done, including the sidebar shortcut panel |
| 13 | User overrides are validated by the same rules and reported, not fatal | `keybindings:` in the app config |

### Debug aid: the keycast strip

Debug builds (`make build-debug`) render a strip of the last six key presses
below the hint bar, each showing the key and the action the registry resolved it
to — or `unbound` in red when the scope has no binding for it. It exists because
an unbound key and a bound key whose action did nothing look identical on
screen, and because a demo recording otherwise shows results with no visible
cause. It resolves against whichever scope is actually active (overlay, text
entry, or the current page), so it reports what the application really did
rather than what the global scope would have done. Release builds get a no-op
stub.

Remaining known divergences, all deliberate and recorded here rather than left
implicit:

- The consumer-group and ksqlDB tables cycle their sort key rather than exposing
  per-column sorts, so a header click there advances the sort instead of
  selecting a column. The main resource table resolves the clicked column
  properly, mirroring bubble-table's flex arithmetic.
- The schema diff view maps the version steppers onto the overlay's up/down
  rather than giving them keys of their own; `[` and `]` were undiscoverable.
- Forms and text editors still switch on editing keys (enter, esc, tab,
  backspace, arrows) directly. That is the text-entry context doing its job, and
  the conformance test allows exactly those literals.

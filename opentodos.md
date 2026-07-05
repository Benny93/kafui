# Open TODOs — remaining findings

Status snapshot: all 263 tasks in `kafui-plan/` are implemented and committed;
`go build ./...` and `go vet ./...` are clean; `go test ./...` is green except
three pre-existing, environment-dependent failures (below). This file tracks the
defects and gaps found while implementing and while recording the VHS demos.

---

## 1. Tables overflow their pane and get wrapped by the enclosing box (HIGH)

**Symptom.** In several feature GIFs the resource table's rows break onto a
second line and the columns no longer line up (e.g. the ACLs view splits
`User:CN=service-account` into `User:CN=service-` + `account`).

**Root cause.** It is *not* in-cell wrapping — the offending cell content is
narrower than its column. The problem is the **sum of the fixed column widths
exceeds the content pane**, so the full row string is wider than the bordered
content box, and the surrounding lipgloss layout wraps that long line (splitting
mid-cell). `github.com/evertras/bubble-table` is configured with
`WithTargetWidth(tableWidth)`, but for **fixed** columns (`table.NewColumn(key,
title, width)`) that call only distributes *surplus* width to flex columns — it
never shrinks fixed columns below their declared width. Column-width sums today:

| Resource   | Σ fixed column widths | Fits a ~90-col pane? |
|------------|-----------------------|----------------------|
| Topics     | 93                    | borderline           |
| Brokers    | 82                    | yes                  |
| ACLs       | 100                   | no                   |
| Consumer groups | 93               | borderline           |
| Connectors | **132**               | no (worst offender)  |
| Connect clusters | 84              | yes                  |
| Quotas     | 96                    | borderline           |

Defined in `createResourceTableColumns` (`pkg/ui/pages/main/providers.go`). The
per-page detail tables (broker/connector/consumer_group/ksql/metrics/clusters)
and `pkg/ui/components/datatable` use the same library and have the same latent
issue on narrow terminals.

**Proper fix (the real product fix).** Make tables always fit their pane at any
width instead of relying on a wide-enough terminal. Options, cheapest first:
1. Make the wide **text** columns flex (`table.NewFlexColumn(key, title,
   flexFactor)`) and keep numeric columns fixed — `WithTargetWidth` then shrinks
   the flex columns to fit and evertras truncates their cell content to a single
   line. This is the smallest change with the biggest payoff.
2. Or compute actual widths from `tableWidth`: scale the declared widths down
   proportionally (with a per-column minimum) whenever `Σwidths > tableWidth`,
   and re-apply via `WithColumns(...)` on resize and on resource switch.
   `createResourceTableColumns` would take a target width and return scaled
   columns.
3. Belt-and-braces: also call `.WithMultiline(false)` on every table (it is the
   library default today, so it is only defensive) so an individual over-long
   cell truncates rather than wraps.

Every table should be verified at a narrow terminal (e.g. 100 cols) after the
fix. The `Connectors` view (8 columns, Σ132) is the acid test — consider
dropping/merging a column there (e.g. fold "Consumer group" into the detail
page) since 8 columns rarely fit a terminal cleanly.

**Video workaround applied now.** The VHS `_config.tape` was widened (larger
`Width`, smaller `FontSize`) so the content pane is wide enough for every
table's fixed columns, and all GIFs were re-rendered with `vhs/render-all.sh`.
This makes the demos correct but does **not** fix the underlying product bug —
the table still overflows on a normal-width terminal. Item #1 above is the fix.

**Related, fixed separately.** The ksqlDB query results table (a different
widget, `charmbracelet/bubbles/table`, not evertras/bubble-table) had the same
symptom for a different reason — confirmed against a real cluster as bug #9 in
`BUG_REPORT_2026-07-05.md`: `rebuildResults()` floored each column to a minimum
of 8 chars with no cap on the total, so wide result sets (many columns)
overflowed the pane regardless of how narrow it was. Fixed by budgeting column
width against the pane width and bubbles/table's per-cell padding instead of an
unconditional floor (`pkg/ui/pages/ksql/query_page.go`). This item (#1) — the
main resource tables' fixed-column-sum overflow — is still open.

---

## 2. FIXED — typing a hotkey letter (e.g. `q`) in the `:` resource picker quit the app

The main-page `MainPageModel` did not implement `IsInputMode()`, so the root
model's hotkey guard (`pkg/ui/ui.go`) couldn't tell that the `:` picker (or the
search bar / a create-edit form) was capturing text. Typing a letter that is
also a global hotkey — `q` (quit), `C`, `K`, `T`, `?` — was handled by the root
instead of the picker. `q` quit the app mid-type; this is also why `:quotas`
appeared "broken" while `:acls`/`:brokers` worked (those names contain no hotkey
letters). Fixed by adding `MainPageModel.IsInputMode()` delegating to the
content provider, so the root suppresses single-key hotkeys while typing.

---

## 2c. FIXED — detail-page tables looked "broken" (stray `│` down an empty pane)

The detail pages (broker/clusters/connector/consumer_group/ksql/metrics) render
`charmbracelet/bubbles/table`, which draws no border of its own. Sitting inside
the template's full-height rounded content box, a short borderless table left
the box's left border `│` running down beside blank rows — reading as a
rendering glitch (user report: "broker view seems broken"). Fixed by wrapping
each table's `View()` in `stylesPkg.FrameTable` (`pkg/ui/styles/frame.go`) — a
rounded border matching the main page's evertras look — and sizing the table to
`contentWidth-2` inside each page's `render(width,height)` (the true content
width, not `SetDimensions`' page width). Also `content.go` now reserves one
column for the optional vertical scrollbar so a full-width framed table doesn't
overflow and wrap its border when the scrollbar appears. All affected GIFs
re-rendered.

## 2b. `Ctrl+D` is bound to two actions at once (LOW)

On the topic resource/detail, `Ctrl+D` deletes, but the template
(`ReusableApp.Update`, `pkg/ui/template/ui/reusable_app.go`) also treats
`Ctrl+D` as "toggle debug overlay". Pressing it deletes *and* pops the
`[App: WxH, mode]` debug overlay. Move the debug toggle to a non-conflicting
combo (e.g. `Ctrl+Alt+D`) or gate it behind the debug build tag.

## 3. FIXED — Sidebar "Resources" active-resource highlight didn't follow `--resource`

`ResourcesSection.RenderItems` (`pkg/ui/pages/main/sidebar_sections.go`) already
lists all nine resource types (Topics/Consumer Groups/Schemas/Contexts/ACLs/
Brokers/Quotas/Connect Clusters/Connectors, capability-gated) — the "missing
resources" half of this item was stale. The active-resource highlight staying
on "Topics" after a `--resource` deep-link, confirmed against a real cluster as
bug #7 in `BUG_REPORT_2026-07-05.md`, was real: the deep-link was applied via an
async message (`SwitchResourceByNameMsg`) fired from `ui.Model.Init()`, racing
the main page's own default-resource `Init()` and usually losing. Fixed by
applying `common.InitialResource` synchronously in
`mainpage.NewModelWithCommon` before `Init()` ever runs (`pkg/ui/pages/main/main_page.go`),
so the content provider and the sidebar's `ResourcesSection.currentResource`
both start correct — no race, no async message needed.

---

## 4. FIXED — `ClusterOverview.TopicCount` was always 0 on the dashboard

Confirmed against a real cluster as bug #6 in `BUG_REPORT_2026-07-05.md`. The
collector never populated `TopicCount`. Fixed by wiring it from a
`GetTopicNames()` count in `Collector.collectOne` (`pkg/cluster/collector.go`).

---

## 5. Deferred-by-design features (documented `ponytail:` notes in code)

These are implemented to their adapted spec but intentionally not fully wired,
because they need external infrastructure or a library that doesn't exist for
Go. Each has a `ponytail:` note at its site.

- **Real Sarama KRaft/ZooKeeper quorum detection** — sarama v1.45.1 has no
  quorum API, so `ClusterStatistics.CoordinationType` is best-effort `"unknown"`
  (`pkg/datasource/kafds/cluster_stats.go`). Everything else in cluster stats is
  real.
- **Replica log-dir reassignment on real clusters** — sarama v1.45.1 lacks
  `AlterReplicaLogDirsRequest`; `AlterReplicaLogDir` returns a typed
  not-supported error on kafds (the mock implements the full flow).
- **Byte-rate throughput** — not derivable from Sarama admin APIs; comes from
  the metrics feature. Dashboard byte-rate columns render `–` unless a metrics
  endpoint (Prometheus/Jolokia) is configured.
- **JMX collection** — native JMX (Java RMI) is not feasible from Go; provided
  via the Jolokia HTTP bridge (MM-17). `Type: "JMX"` without a bridge degrades
  to a logged warning + empty broker metrics, as the spec allows.
- **PromQL range graphs (MM-14/15)** — implemented, but only active when a
  Prometheus-compatible `TimeSeriesURLs` endpoint is configured; otherwise the
  graph list is hidden and the in-process sparklines remain.
- **OAuth2 device-code (AA-13)** — implemented; needs a real OIDC provider with
  a device-authorization endpoint to exercise end-to-end.

---

## 6. Pre-existing test failures (NOT introduced by this work; do not "fix" blind)

- `TestKafkaDataSourceKaf_GetTopics_Integration` and
  `TestKafkaDataSourceKaf_ConsumeTopic_Integration`
  (`pkg/datasource/kafds`) — expect a connection error but get `nil`; they pass
  only when no broker is reachable, so they fail whenever a local Kafka is up.
  They should be gated behind `testing.Short()`/a build tag or given an isolated
  no-broker address.
- `TestKafkaDataSourceMock_ConsumeTopic_RealisticOffsets`
  (`pkg/datasource/mock`) — flaky (~1 in 6): asserts monotonically increasing
  offsets across messages whose partitions are assigned randomly in
  `generateMessage`. Make the assertion partition-aware or seed the RNG.
- `TestCappedContentWidth` (`pkg/ui/template/ui/components`) — asserts a content
  width cap of 120 but the code caps differently; either the test or the cap
  constant is stale. Reconcile them.

---

## 7. Lint debt (LOW, non-blocking)

`go vet` is clean, but `staticcheck`/modernize surface a long tail across the
tree: `interface{}` → `any`, `for i := 0; i < n` → `for range n`, a few
`sort.Slice` → `slices.Sort`, deprecated `io/ioutil.ReadFile`,
`tls.BuildNameToCertificate`, `strings.Title`, and `lipgloss Style.Copy`. A
`gofmt`/modernize sweep would clear most of it. None affects behavior.

---

## VHS demos

Tapes live in `vhs/*.tape` (shared settings in `vhs/_config.tape`); rendered
GIFs in `vhs/gifs/*.gif`; embedded in the README "Features" section. Re-render
everything with `vhs/render-all.sh` (builds the binary, then runs every tape).
The ACLs GIF shows ACLs + the create/convenience form rather than switching to
quotas, because of finding #2.

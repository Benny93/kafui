# Bug Report — real-cluster exercise of `vehub-dev-aks`

**Date:** 2026-07-05
**Build:** `0.1.32-282-ge64f042-dirty` (commit `e64f042`, go1.25.6, darwin/arm64)
**Cluster:** `vehub-dev-aks` (current context in `~/.kaf/config`; Strimzi on Azure AKS,
TLS mutual-auth, bootstrap `kafka-bootstrap.vehub-dev...:443`, schema registry configured)
**Mode:** every run used `--read-only` — **no destructive/mutating action was performed
against the cluster.** Only listing/reading/consuming.

## Method

- Non-interactive: `kafui health`, `kafui get brokers`, CLI edge cases (bad cluster,
  bad format).
- Interactive TUI driven headlessly with VHS (real `$HOME`, no `--mock`), one feature
  per tape, deep-linked via `--resource <type>` and global hotkeys (`C`, `ctrl+t`,
  `ctrl+g`, `K`). Frames extracted with ffmpeg and inspected; `~/.kafui/kafui.log`
  scanned for backend errors/panics after each run.
- No `level=ERROR` or panic was logged during any *interactive* TUI run — the TUI
  degrades gracefully on this cluster's API/authorization restrictions. All crashes
  found were on the *non-interactive* CLI path.

---

## Bugs

### 1. Unknown `--cluster` / `-c` silently connects to `localhost:9092` (HIGH)

An unrecognised cluster/context name is **not validated**; kafui silently falls back to
the `localhost:9092` default and then reports a misleading "can't reach localhost"
error. A user who typos a context name has no idea their name was ignored.

**Repro**
```
$ kafui health -c no-such-cluster
broker  FAIL  Unable to get client: ... dial tcp [::1]:9092: connect: connection refused
$ kafui get brokers -c no-such-cluster
Error: listing brokers: unable to connect to Kafka cluster at [localhost:9092]: ...
```

**Root cause** — `config.ActiveCluster()` (kaf dep) returns `nil` when the
`ClusterOverride` name matches no cluster. Both `onInit()` and `InitFromConfig()` in
`pkg/datasource/kafds/datasource_kaf.go` treat `nil` as "not configured" and substitute
`&config.Cluster{Brokers: []string{"localhost:9092"}}`. The override name is never
checked against the configured clusters.

**Fix** — when `clusterFlag`/`ClusterOverride` is non-empty but `ActiveCluster()` returns
`nil`, fail fast with `cluster %q not found in config (available: …)` instead of falling
back to localhost.

---

### 2. Every non-interactive CLI error panics with a Go stack trace (MEDIUM)

`get` (and any future command) that returns an error prints the cobra error + usage and
then **panics**, dumping a goroutine stack trace to the user and exiting with code 2.

**Repro**
```
$ kafui get brokers --format json ; echo $?
Error: unsupported format "json" (only csv)
Usage: ...
panic: unsupported format "json" (only csv)
goroutine 1 [running]:
github.com/Benny93/kafui/cmd/kafui.DoExecute() .../cmd/kafui/root.go:82
2
```
(also reproduces with `-c <bad-cluster>`, `--format bogus`, or any connection failure).

**Root cause** — `cmd/kafui/root.go:82`:
```go
func DoExecute() {
    rootCmd := CreateRootCommand(defaultKafuiInit)
    if err := rootCmd.Execute(); err != nil {
        panic(err)          // <- dumps a stack trace on every user error
    }
}
```
Note `health` does *not* panic (it prints `FAIL` and `os.Exit(1)` itself), so behaviour
is inconsistent across subcommands.

**Fix** — print the error to stderr (cobra already does) and `os.Exit(1)`; don't
`panic`. Also set `SilenceErrors`/`SilenceUsage` on the leaf commands so the error isn't
printed twice (once by cobra, once here) and usage isn't dumped on a runtime failure.

---

### 3. Contexts view reuses the Topics column headers (MEDIUM, cosmetic/correctness)

Opening Contexts (`--resource contexts`) renders the table with headers
**`Name | Partitions | Replication | Messages`**, so the broker bootstrap address sits
under "Partitions" and the active/inactive status under "Replication".

**Root cause** — `createResourceTableColumns` (`pkg/ui/pages/main/providers.go`) has no
`case ContextResourceType`, so contexts fall through to the generic `default:` (Topics)
columns.

**Fix** — add a `ContextResourceType` case with context-appropriate headers, e.g.
`Name | Brokers | Status` (and drop the numeric "Messages" column that is always `–`).

---

### 4. Topic list `OSR` and `Size` columns never resolve — perpetual loading spinner (MEDIUM)

On this cluster the topic list's `Name / Partitions / Replication / Messages` all
populate (Messages after ~a few seconds), but **`OSR` and `Size` stay on the loading
placeholder (`…`) indefinitely** (still spinning after 26 s).

`Size` depends on `DescribeLogDirs`, which this cluster does not expose (see §7), but the
UI never falls back to a terminal state — it shows an infinite spinner that reads as
"hung". `OSR` appears to be resolved in the same detail pass and is stuck alongside it,
even though ISR/replica data *is* available (the Brokers view shows `ISR 1759/1759`).

**Fix** — when the per-topic detail/log-dir fetch fails or is unsupported, render `Size`
(and `OSR` if it can't be derived) as `N/A`/`–` instead of an unbounded spinner.
Consider deriving `OSR` from topic metadata (ISR vs replicas) independently of the
log-dir/size fetch so a size failure doesn't block it.

---

### 5. Sidebar "Broker Summary → Brokers" shows the bootstrap count, not the real broker count (LOW)

The main-page sidebar shows **Brokers (1)** for `vehub-dev-aks`, but the cluster has
**3** brokers (the Brokers list and the Clusters dashboard both correctly show 3, and
`kafui get brokers` returns 3).

**Root cause** — the sidebar cluster summary uses `len(details.Brokers)`
(`sidebar_sections.go:310`), and `GetClusterDetails` (`datasource_kaf.go:163`) returns
`Brokers: cluster.Brokers` — i.e. the **configured bootstrap endpoints** from
`~/.kaf/config` (one entry), not the brokers discovered from cluster metadata.

**Fix** — populate the summary broker count from cluster metadata (as the Clusters
dashboard already does) rather than the bootstrap list length.

---

### 6. Clusters dashboard "Topics" is always 0 (LOW)

The Clusters dashboard (`C`) shows **Topics: 0** for both `vehub-dev-aks` and
`vehub-preprod-aks`, despite thousands of topics. (This matches the pre-existing note
`opentodos.md` #4 — `ClusterOverview.TopicCount` is never populated — now confirmed
against a real cluster.)

**Fix** — wire `TopicCount` from a `GetTopicNames()` count (or the metadata already
fetched for partition stats) in `pkg/cluster/collector.go`.

---

### 7. Active-resource indicator doesn't follow a `--resource` deep-link (LOW, cosmetic)

Launching `--resource consumer-groups` (or schemas/brokers/…) shows the correct data,
but the sidebar "Resources" highlight (green dot) and the footer breadcrumb both stay on
**"Topics"**. (Matches `opentodos.md` #3, confirmed live.)

**Fix** — set the active resource type from the `--resource` flag on startup and on
picker switch so the sidebar/footer reflect it.

### 8. Cluter overview opened mutliple times grows the breadcrumb bar indefnetly

### 9. ksqldb running a query content does not fit borders and newlines wraps appear breaking the view

---

## Graceful degradations / minor observations (not crashes)

These are handled without crashing and are mostly cluster capability limits, but the
messaging or fallback could be improved:

- **ACLs** → `Error: failed to list ACLs: EOF`. Surfaced cleanly, but "EOF" is cryptic;
  likely means the cluster has no ACL authorizer. Suggest: "ACLs unavailable (authorizer
  may be disabled)".
- **Quotas** → `Error: failed to describe client quotas: kafka server: The version of
  API is not supported`. Broker doesn't support the `DescribeClientQuotas` version
  sarama requests. Clean message; consider version negotiation or a friendlier hint.
- **Broker → Configs tab** → `Cluster authorization failed` (client lacks describe-config
  perms). Clean, clear, with retry hint. 👍
- **Broker → Log Dirs tab** → `Log dir data not available`; broker `Disk Usage: N/A`.
  Clean. This cluster doesn't expose `DescribeLogDirs` (also the source of §4's `Size`).
- **Connectors** → `No connectors found` (no Kafka Connect REST URL configured). Clean
  empty state; could hint how to configure a Connect endpoint.
- **Metrics** (`ctrl+t`) → stuck on `Collecting metrics… (rates appear after the second
  cycle)` for ~15 s (refresh interval 30 s). Expected, but a first snapshot before the
  second cycle would feel less empty.
- **ksqlDB** (`K`) → silent no-op when no ksqlDB endpoint is configured (capability-gated,
  intended). No feedback though — a toast like "ksqlDB not configured for this cluster"
  would help.
- **Clusters dashboard** → `Version: Unknown` (Kafka version not detected) and
  `Msgs/s`/`Bytes` = `–` (no metrics endpoint). Both documented as best-effort. Also both
  clusters show identical Brokers=3 / Partitions=2571 — likely genuinely mirrored envs,
  but worth confirming the collector actually polls preprod separately (low confidence).
- **App config** shows per-cluster `Read Only: false` even when launched with
  `--read-only` (the header badge does show `read-only`). The page reflects the *config*
  value, not the runtime flag; a note or an "effective read-only" line would avoid
  confusion.
- **`__transaction_state`** message keys render with a leading `?` (a non-printable byte
  in the binary transactional-id key shown as `?`). Cosmetic.

## Works well against the real cluster ✅

Topics list + topic message consumption (72 msgs, ~3.8 s), Consumer Groups list +
detail, Schemas list + schema detail (Avro JSON with highlighting + info sidebar),
Brokers list, Contexts switching, Clusters dashboard, App config page (no secrets
leaked, whoami correctly hidden since authz is disabled), `health` (broker + SR OK),
`get brokers`. Read-only mode enforced throughout. No panics or `ERROR` logs in the
interactive TUI.

## Scope note

`kafui get` only implements `get brokers`; topics/consumer-groups/schemas have no
machine-readable listing. Not a bug, but a gap if scripting parity with the TUI is a goal.

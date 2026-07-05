# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Kafui is a k9s-inspired terminal UI for Apache Kafka, built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea). It reads `$HOME/.kaf/config` (shared with the [kaf](https://github.com/birdayz/kaf) CLI), so any configured kaf cluster is browsable.

## Commands

```bash
make build              # Release binary (-ldflags "-w -s")
make build-debug        # Debug binary (-tags debug); enables F3 screenshot capture
make run-mock           # Run against mock data — no Kafka broker needed
make run                # Run against the real cluster from ~/.kaf/config
make test               # Full suite + HTML coverage report
make test-short         # go test -short (skips slow/integration)
make check-coverage     # Enforce thresholds from .testcoverage.yml
make run-kafka / stop-kafka   # Local Kafka via example/dockercompose

# Single test
go test ./pkg/ui/pages/topic/... -v -run TestName
```

`make run-mock` is the fastest inner loop — the mock datasource (`pkg/datasource/mock`) implements the full `KafkaDataSource` interface, so most UI work needs no broker.

## Architecture

Bubble Tea's Elm architecture (Init → Update → View) over a layered design. Data flows one way: the `KafkaDataSource` interface is the only path to Kafka, and every UI component receives its dependencies through `*core.Common` — **never use globals for data access.**

Boot path: `main.go` → `cmd/kafui` (Cobra) → `ui.Init` picks the real or mock datasource → `ui.OpenUI` starts the Bubble Tea program → `Router` (`pkg/ui/router`) owns page navigation.

- **`pkg/api/`** — `KafkaDataSource` interface + domain types (`Message`, `Topic`, `Schema`, `ACLEntry`, …). Typed domain errors live in `errors.go`. Several methods are deliberately split for laziness on large clusters: `GetTopicNames` (fast, names only) vs `GetTopics` (full partition detail); `GetSchemas` (subjects) vs `GetSchemaDetails`/`GetSchemaContent`. `DecodeMessage` decodes Avro raw bytes lazily, on demand.
- **`pkg/datasource/kafds/`** — Real implementation on IBM Sarama. Uses factory interfaces (`KafkaClientFactory`, `ConsumerInterface`) for DI/testing. `InitTUIWriters()` redirects stdout/stderr/sarama logging to a file so it can't corrupt the TUI.
- **`pkg/datasource/mock/`** — Full mock datasource for `--mock` and tests.
- **`pkg/ui/core/`** — `Page`/`StatefulPage`/`Component` interfaces, `BaseComponent`, and the `Common` DI struct (holds DataSource, Styles, Layout, Config).
- **`pkg/ui/router/`** — Page registry, history/breadcrumbs, and message-driven navigation. Supports dynamic IDs like `topic:<name>`.
- **`pkg/ui/pages/`** — Page implementations: `main`, `topic`, `message_detail`, `resource_detail`, `schema_detail`.
- **`pkg/ui/template/ui/`** — Template system giving pages a consistent header/sidebar/content/footer via provider interfaces.
- **`pkg/ui/components/`, `pkg/ui/styles/`, `pkg/ui/keys/`** — Reusable components (embed `core.BaseComponent`), semantic light/dark style palette (reference styles by role, not hex), centralized key bindings.

### Page conventions

Each page is a package under `pkg/ui/pages/<name>/` implementing `core.Page` (`Init`, `Update`, `View`, `SetDimensions`, `GetID`, `GetTitle`, `GetHelp`, `HandleNavigation`, `OnFocus`, `OnBlur`). File layout varies by page complexity — `<name>_page.go` is the entry Model and `<name>_providers.go` holds template providers, but larger pages split logic across topical files (e.g. `topic/` has `handlers.go`, `consumption.go`, `pagination.go`, `keys.go`, `types.go`). Follow the existing split in the page you're editing rather than a fixed template.

### Authorization & audit seam (auth-rbac-audit)

Local permission profiles + audit + read-only enforcement live behind one seam. Two contracts every feature must honor:

1. **Every state-changing `KafkaDataSource` method is enforced in `pkg/datasource/guard.go`.** `Guard` EMBEDS `api.KafkaDataSource`, so reads/analytics pass through automatically. When you add a *mutating* method to the interface you MUST override it on `Guard` (use the `do`/inline-closure helpers) declaring its authz resource + action: the override gate-checks *before any effect* (returning `api.AccessDeniedError` on profile denial, `api.ClusterReadOnlyError` under read-only) and emits one audit record. Action vocabulary + altering classification are in `pkg/authz/model.go`; the Gate is `pkg/authz/gate.go`. Authz is disabled (allow-all) when no profiles are configured; read-only is an independent switch (`--read-only` flag or per-cluster `readOnly`).
2. **Every mutating UI action checks `core.Common.Can(action, resource, name)`** to hide/disable its key, and routes the guard's typed error to the status bar as a `UIError`/notification (the guard is the backstop). See `pkg/ui/pages/topic/produce.go` (`canProduce`) for the pattern. Identity + active profile + read-only badge render in the header (`pkg/ui/pages/main/providers.go`); the effective-permissions "whoami" view is a section of the appconfig page (`pkg/ui/pages/appconfig_view`). Config schema + audit sections are in `pkg/appconfig` (`AuthzSettings`, `AuditSettings`); see `example/kafui-config.yaml`.

### Conventions

- **Errors**: domain errors are typed structs in `pkg/api/errors.go` with `Unwrap()`; UI errors are `UIError` (`pkg/ui/shared/types.go`) sent as `tea.Cmd` messages and shown in the status bar. Wrap with `fmt.Errorf("...: %w", err)`.
- **Receivers**: pointer receivers for any method that mutates state (ADR-011).
- **Tests**: `testify/assert` + `testify/mock`; table-driven with `t.Run`; inject the mock `KafkaDataSource` via constructors.

## Repo notes

- `DEVELOPMENT_GUIDE.md` has copy-paste code snippets for adding pages, components, key bindings, and using Common/Layout/Styles — a live how-to, not a status snapshot.
- `.github/copilot-instructions.md` carries the same conventions in more detail — keep the two in sync when either changes.
- Architecture decisions are recorded in `ARCHITECTURE_DECISIONS.md` (ADR format).
- The many `*.md` files at the repo root (BUBBLE_TEA_*, PHASE_*, *_PLAN, *_STATUS, etc.) are historical planning/status snapshots from past refactors — **not authoritative**. Trust the code, ADRs, and this file over them.

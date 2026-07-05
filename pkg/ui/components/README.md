# UI Components

Reusable, page-agnostic UI building blocks. Page layout/chrome (header, sidebar,
footer, content area, breadcrumb) is owned by the template system under
`pkg/ui/template/ui`, **not** this package — the older `layout.go`, `sidebar.go`,
`footer.go`, `main_content.go`, `status_bar.go`, and `modal.go` were removed
(UI-18) once the template system and the `pkg/ui/dialog` / `pkg/ui/notify`
packages superseded them.

## Live components

- **`search_bar.go`** — `SearchBarModel`, a text-input search/filter bar with
  fuzzy suggestions and history.
- **`fuzzy.go`** — `FuzzyMatcher` used for suggestion ranking and filtering.
- **`fetch_progress_bar.go`** — `FetchProgressBar`, a determinate animated
  progress bar for counted batch fetches (channel-driven `ProgressMsg` stream).
- **`loading.go`** — `Spinner` wrapper over `bubbles/spinner` plus
  `CenteredLoading`, the shared indeterminate loading indicator (UI-12).
- **`json_content_view.go`** — read-only JSON content view (also see the richer
  `editor/` subpackage).
- **`sparkline.go`** — inline sparkline renderer for the metrics page.
- **`styles.go`** — small shared lipgloss style helpers for the above.

## Subpackages

- **`form/`** — shared create/edit form framework (`Form`, typed fields).
- **`datatable/`** — shared sortable/paginated table pattern.
- **`editor/`** — read-only viewer, editable textarea, and diff views with JSON
  highlighting.

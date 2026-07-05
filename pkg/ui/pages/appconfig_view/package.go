// Package appconfig_view contains the read-only "Application Config" page.
//
// It renders the effective merged kafui configuration (build info, app/UI
// settings, and per-cluster sections) as a scrollable, structured document.
// All secret values are masked through common.Redactor before display.
//
// The intended router page ID is "appconfig" (registration lives in the router,
// not here). The page implements core.Page via the template UI system and uses a
// bubbles viewport for scrolling.
//
// Architecture:
//   - appconfig_view_page.go: page model, document builder, and Page interface
//   - appconfig_view_providers.go: viewport-backed content provider
package appconfig_view

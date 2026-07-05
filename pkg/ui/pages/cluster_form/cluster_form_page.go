// Package cluster_form implements the cluster setup-wizard page (AC-12/AC-13):
// add, edit or delete a cluster in the kafui-owned config, validate connectivity
// without saving, and apply changes with an in-place datasource reload.
//
// The page is gated on appconfig.DynamicConfigEnabled: when the toggle is off it
// renders an explanation and refuses to edit (entry points should also be hidden
// by their hosts).
package cluster_form

import (
	"context"
	"fmt"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
	"github.com/Benny93/kafui/pkg/ui/core"
	stylesPkg "github.com/Benny93/kafui/pkg/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PageID is the router base ID for the wizard. Instances use a dynamic ID of the
// form "cluster_form:<name>" (empty name ⇒ add mode).
const PageID = "cluster_form"

var _ core.Page = (*Model)(nil)

// candidateValidator is the seam onto the AC-11 connectivity-validation service.
// The real datasource (*kafds.KafkaDataSourceKaf) implements it.
type candidateValidator interface {
	ValidateCandidate(ctx context.Context, candidate appconfig.Config) api.ValidationReport
}

// reloader is the seam onto the AC-13 in-place reload. Implemented by
// *kafds.KafkaDataSourceKaf; absent on the mock (apply then only updates Common).
type reloader interface {
	Reload(effective appconfig.Config) error
}

// Model is the cluster setup-wizard page.
type Model struct {
	common       *core.Common
	dims         core.Dimensions
	form         *formpkg.Form
	originalName string // "" ⇒ add mode; else the cluster being edited
	disabled     bool
	results      *api.ValidationReport
	notice       string

	// validate runs the AC-11 probe on a candidate. Overridable in tests.
	validate func(ctx context.Context, candidate appconfig.Config) api.ValidationReport
	// savePath is the kafui config file to persist to. Overridable in tests.
	savePath string
}

// NewModelWithCommon builds the wizard for adding (empty clusterName) or editing
// a cluster. When dynamic config is disabled the page is inert (see disabled).
func NewModelWithCommon(common *core.Common, clusterName string) *Model {
	m := &Model{
		common:       common,
		originalName: clusterName,
		savePath:     appconfig.DefaultPath(),
	}
	if common.AppConfig == nil || !common.AppConfig.DynamicConfigEnabled {
		m.disabled = true
		return m
	}
	m.form = formpkg.New(buildFields(clusterName, m.prefill(clusterName)))
	m.validate = func(ctx context.Context, candidate appconfig.Config) api.ValidationReport {
		if v, ok := common.DataSource.(candidateValidator); ok {
			return v.ValidateCandidate(ctx, candidate)
		}
		return api.ValidationReport{}
	}
	return m
}

// prefill resolves the current extension for a cluster being edited, filling
// broker/registry basics from the datasource when the cluster lives in the
// read-only kaf file rather than the kafui file.
func (m *Model) prefill(name string) appconfig.ClusterExtension {
	if name == "" || m.common.AppConfig == nil {
		return appconfig.ClusterExtension{}
	}
	ext := m.common.AppConfig.Clusters[name]
	if !ext.IsFullyDefined() && m.common.DataSource != nil {
		if details, err := m.common.DataSource.GetClusterDetails(name); err == nil {
			ext.Brokers = details.Brokers
			if ext.SchemaRegistryURL == "" {
				ext.SchemaRegistryURL = details.SchemaRegistryURL
			}
			ext.ReadOnly = ext.ReadOnly || details.ReadOnly
		}
	}
	return ext
}

// disabledCmd surfaces the toggle explanation as an error notification.
func disabledCmd() tea.Cmd {
	return func() tea.Msg {
		return core.NotificationMsg{
			Severity: core.StatusError,
			Title:    "Cluster editing disabled",
			Message:  "Set dynamicConfigEnabled: true in the kafui config to add, edit or delete clusters.",
			Sticky:   true,
		}
	}
}

// Init implements core.Page.
func (m *Model) Init() tea.Cmd {
	if m.disabled {
		return disabledCmd()
	}
	return m.form.Init()
}

// Update implements core.Page.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil
	case formpkg.FormSubmitMsg:
		return m, m.apply(msg.Values)
	case formpkg.FormCancelMsg:
		return m, func() tea.Msg { return core.BackMsg{} }
	case tea.KeyMsg:
		if m.disabled {
			return m, func() tea.Msg { return core.BackMsg{} }
		}
		switch msg.String() {
		case "ctrl+v":
			return m, m.runValidate()
		case "ctrl+d":
			if m.originalName != "" {
				return m, m.confirmDelete()
			}
		}
	}
	if m.disabled || m.form == nil {
		return m, nil
	}
	cmd, _ := m.form.Update(msg)
	return m, cmd
}

// runValidate builds a candidate from the current field values and probes it via
// the AC-11 service, storing the per-service report for rendering (no save).
func (m *Model) runValidate() tea.Cmd {
	name, ext, err := candidateFromValues(m.form.Values())
	if err != nil {
		m.notice = "cannot validate: " + err.Error()
		return nil
	}
	candidate := appconfig.Config{Clusters: map[string]appconfig.ClusterExtension{name: ext}}
	report := m.validate(context.Background(), candidate)
	m.results = &report
	m.notice = ""
	return nil
}

// apply maps the submitted form to a candidate, merges + validates + persists it
// to the kafui file only, reloads the datasource in place, and navigates back.
func (m *Model) apply(values map[string]string) tea.Cmd {
	name, ext, err := candidateFromValues(values)
	if err != nil {
		return core.NotifyError("Invalid cluster", err)
	}
	merged, err := appconfig.ApplyCluster(m.savePath, *m.common.AppConfig, m.originalName, name, ext)
	if err != nil {
		return core.NotifyError("Apply failed", err)
	}
	m.applyEffective(merged)
	return tea.Batch(
		core.NewNotification(core.StatusSuccess, "Cluster saved", name),
		func() tea.Msg { return core.BackMsg{} },
	)
}

// confirmDelete asks for confirmation before removing the edited cluster.
func (m *Model) confirmDelete() tea.Cmd {
	name := m.originalName
	return func() tea.Msg {
		return core.ShowConfirmMsg{
			Title:        "Delete cluster",
			Message:      fmt.Sprintf("Remove cluster %q from the kafui config?", name),
			Danger:       true,
			ConfirmLabel: "Delete",
			OnConfirm:    m.doDelete(name),
		}
	}
}

func (m *Model) doDelete(name string) tea.Cmd {
	return func() tea.Msg {
		merged, err := appconfig.DeleteCluster(m.savePath, *m.common.AppConfig, name)
		if err != nil {
			return core.NotificationMsg{Severity: core.StatusError, Title: "Delete failed", Message: err.Error()}
		}
		m.applyEffective(merged)
		return core.BackMsg{}
	}
}

// applyEffective installs the new effective config on Common and reloads the
// datasource in place when it supports it (real kafds; the mock does not).
func (m *Model) applyEffective(merged appconfig.Config) {
	m.common.ApplyAppConfig(merged)
	if r, ok := m.common.DataSource.(reloader); ok {
		_ = r.Reload(merged)
	}
}

// View implements core.Page.
func (m *Model) View() string {
	s := m.common.Styles
	if m.disabled {
		return s.Error.Render("Cluster editing is disabled.") + "\n\n" +
			s.Muted.Render("Set dynamicConfigEnabled: true in the kafui config to add, edit or delete clusters.\nPress esc to go back.")
	}

	title := "Add Cluster"
	if m.originalName != "" {
		title = "Edit Cluster: " + m.originalName
	}

	var b strings.Builder
	b.WriteString(s.Header.Render(title))
	b.WriteString("\n")
	b.WriteString(s.Muted.Render("ctrl+v validate • ctrl+d delete • enter submit • esc cancel"))
	b.WriteString("\n\n")
	b.WriteString(m.form.View())
	if m.notice != "" {
		b.WriteString("\n" + s.Error.Render(m.notice))
	}
	b.WriteString(m.renderResults())
	return b.String()
}

func (m *Model) renderResults() string {
	if m.results == nil {
		return ""
	}
	s := m.common.Styles
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(s.Header.Render("Validation Results"))
	b.WriteString("\n")
	if len(m.results.Clusters) == 0 {
		b.WriteString(s.Muted.Render("  no results"))
		return b.String()
	}
	okStyle := lipgloss.NewStyle().Foreground(stylesPkg.Success)
	for _, cv := range m.results.Clusters {
		for _, r := range cv.Results {
			if r.OK {
				b.WriteString(okStyle.Render(fmt.Sprintf("  ✓ %s", r.Component)))
			} else {
				b.WriteString(s.Error.Render(fmt.Sprintf("  ✗ %s: %s", r.Component, r.Err)))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// SetDimensions implements core.Page.
func (m *Model) SetDimensions(width, height int) {
	m.dims = core.Dimensions{Width: width, Height: height}
	if m.form != nil {
		m.form.SetDimensions(width, height)
	}
}

// GetID implements core.Page.
func (m *Model) GetID() string {
	if m.originalName != "" {
		return PageID + ":" + m.originalName
	}
	return PageID
}

// GetTitle implements core.Page.
func (m *Model) GetTitle() string { return "Cluster" }

// GetHelp implements core.Page.
func (m *Model) GetHelp() []key.Binding { return nil }

// HandleNavigation implements core.Page.
func (m *Model) HandleNavigation(msg tea.Msg) (core.Page, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" && m.disabled {
		return m, func() tea.Msg { return core.BackMsg{} }
	}
	return m, nil
}

// OnFocus implements core.Page.
func (m *Model) OnFocus() tea.Cmd { return nil }

// OnBlur implements core.Page.
func (m *Model) OnBlur() tea.Cmd { return nil }

package ui

import (
	"fmt"
	"log"
	"os"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/audit"
	"github.com/Benny93/kafui/pkg/authz"
	"github.com/Benny93/kafui/pkg/cluster"
	"github.com/Benny93/kafui/pkg/datasource"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/Benny93/kafui/pkg/datasource/mock"
	"github.com/Benny93/kafui/pkg/metrics"
	"github.com/Benny93/kafui/pkg/ui/router"
	"github.com/Benny93/kafui/pkg/ui/shared"
	zone "github.com/lrstanley/bubblezone"
	tea "github.com/charmbracelet/bubbletea"
)

// openUIFunc is a variable that holds the OpenUI function, allowing it to be mocked in tests
var openUIFunc = OpenUI

// InitOptions carries CLI-level configuration into the app.
type InitOptions struct {
	ConfigFile     string
	Mock           bool
	Brokers        []string
	SchemaRegistry string
	Cluster        string
	Verbose        bool
	// ReadOnly is the global --read-only flag: when true every cluster is
	// treated read-only and all altering operations are denied (AA-4).
	ReadOnly bool
	// Topic deep-links directly to a topic page on startup (UI-9).
	Topic string
	// Resource pre-switches the main page to a resource type on startup (UI-9).
	Resource string
	// MetricsListen is the optional --metrics-listen address (e.g. ":9090"). When
	// non-empty, kafui serves the current metrics snapshot in Prometheus
	// exposition format for the lifetime of the program (MM-16). Default off.
	MetricsListen string
}

// Init boots kafui with the given options.
func Init(opts InitOptions) {
	shared.InitLogger()

	var dataSource api.KafkaDataSource

	dataSource = &mock.KafkaDataSourceMock{}
	if !opts.Mock {
		// Apply CLI overrides before the datasource reads config.
		kafds.SetOverrides(opts.Brokers, opts.SchemaRegistry, opts.Cluster, opts.Verbose)
		ds := kafds.NewKafkaDataSourceKaf()
		// Run the interactive OAuth2 device-code grant (if configured) while
		// stdout is still the terminal — before InitTUIWriters redirects it (AA-13).
		if err := kafds.PrepareOAuthDeviceFlow(opts.ConfigFile, os.Stdout); err != nil {
			log.Fatalf("OAuth device authentication failed: %v", err)
		}
		kafds.InitTUIWriters() // redirect stdout/stderr/sarama to log file before TUI starts
		dataSource = ds
	}
	dataSource.Init(opts.ConfigFile)
	if !opts.Mock {
		if err := api.ValidateClusterOverride(dataSource, opts.Cluster); err != nil {
			log.Fatalf("%v", err)
		}
	}

	// Load the kafui-owned config (missing file tolerated -> defaults).
	appCfg, err := appconfig.Load(appconfig.DefaultPath())
	if err != nil {
		log.Printf("kafui config: %v (using defaults)", err)
		appCfg = appconfig.Default()
	}
	validateClusters(dataSource, &appCfg)

	// Build the authorization gate + audit service and wrap the datasource with
	// the enforcement guard before it reaches the UI (AA-8). A bad authz config
	// is fatal (fail fast before the TUI starts).
	readOnlyForCluster := func(cluster string) bool {
		ext, ok := appCfg.Clusters[cluster]
		return ok && ext.ReadOnly
	}
	gate, err := authz.NewGate(appCfg.Authz, readOnlyForCluster, opts.ReadOnly)
	if err != nil {
		log.Fatalf("invalid authz configuration: %v", err)
	}
	if !gate.Enabled() && !opts.ReadOnly {
		shared.Log.Warn("authorization is disabled: access is unrestricted (no profiles configured)")
	}
	auditSvc := buildAuditService(appCfg.Audit)
	dataSource = datasource.NewGuard(dataSource, gate, auditSvc)

	openUIFunc(dataSource, appCfg, gate, audit.ResolveUser(), opts.Topic, opts.Resource, opts.MetricsListen)
}

// buildAuditService constructs the audit service from config, returning a
// disabled no-op service when auditing is off or the log file cannot be opened.
func buildAuditService(cfg appconfig.AuditSettings) *audit.Service {
	if !cfg.Enabled {
		return audit.NewService(false, "", nil, shared.Log)
	}
	path := cfg.Path
	if path == "" {
		path = audit.DefaultPath()
	}
	w, err := audit.NewFileWriter(path)
	if err != nil {
		shared.Log.Error("audit disabled: cannot open log file", "path", path, "err", err)
		return audit.NewService(false, "", nil, shared.Log)
	}
	return audit.NewService(true, audit.Level(cfg.Level), w, shared.Log)
}

// validateClusters runs startup validation over the merged cluster list. Problems
// in the shared ~/.kaf/config are logged but tolerated (that file is not ours to
// reject); this surfaces duplicate names and missing brokers as warnings.
func validateClusters(ds api.KafkaDataSource, _ *appconfig.Config) {
	names, err := ds.GetContexts()
	if err != nil {
		return
	}
	clusters := make([]appconfig.ClusterConfig, 0, len(names))
	for _, n := range names {
		info, derr := ds.GetClusterDetails(n)
		if derr != nil {
			continue
		}
		clusters = append(clusters, appconfig.ClusterConfig{Name: info.Name, Brokers: info.Brokers})
	}
	if verr := appconfig.Validate(clusters); verr != nil {
		log.Printf("cluster config warning: %v", verr)
	}
}

func OpenUI(dataSource api.KafkaDataSource, appCfg appconfig.Config, gate *authz.Gate, identity, initialTopic, initialResource, metricsListen string) {
	zone.NewGlobal()
	model := initialModelWithRouter(dataSource)
	common := model.GetCommon()
	common.Gate = gate
	common.Identity = identity
	common.ApplyAppConfig(appCfg)
	// Resolve and apply the theme to BOTH style systems (UI-3): "auto" uses
	// terminal-background detection; the template chrome follows the selection.
	model.applyThemeMode(appCfg.UI.Theme)
	common.Collector = cluster.New(dataSource, appCfg.RefreshInterval, func(name string) bool {
		ext, ok := appCfg.Clusters[name]
		return ok && ext.ReadOnly
	})
	// Metrics collector: offset-delta message-in rates (always available) plus a
	// stubbed byte-rate endpoint path. Poll cadence comes from the active
	// cluster's metrics config (falls back to the collector default).
	metricsInterval := appconfig.DefaultMetricsPollInterval
	if ext, ok := appCfg.Clusters[dataSource.GetContext()]; ok {
		metricsInterval = ext.MetricsSettings().PollInterval
	}
	common.MetricsCollector = metrics.New(dataSource, metricsInterval, func(name string) string {
		ext, ok := appCfg.Clusters[name]
		if !ok {
			return ""
		}
		return ext.MetricsSettings().Endpoint
	})
	// Resolve full per-cluster metrics settings so the collector can honor
	// Type=JMX via the Jolokia bridge (MM-17).
	common.MetricsCollector.SetSettingsResolver(func(name string) appconfig.MetricsSettings {
		ext, ok := appCfg.Clusters[name]
		if !ok {
			return appconfig.MetricsSettings{}
		}
		return ext.MetricsSettings()
	})
	// Optional Prometheus exposition endpoint (MM-16): flag-gated, default off.
	stopExposition := startExpositionServer(metricsListen, common.MetricsCollector, appCfg)
	// CLI deep-linking (UI-9): open a topic directly, or pre-switch the resource.
	if initialTopic != "" {
		model.Router.SetInitialRoute("topic:"+initialTopic, &router.NavigationData{TopicName: initialTopic})
	}
	common.InitialResource = initialResource
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	// Config auto-reload (AC-16): poll the kafui config file for changes and
	// hot-apply reloadable settings. Off by default; never reconnects clusters.
	var watchDone chan struct{}
	if appCfg.AutoReload.Enabled {
		watchDone = make(chan struct{})
		go watchConfigFile(p, appconfig.DefaultPath(), appCfg.AutoReload.Interval, watchDone)
	}
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
	if watchDone != nil {
		close(watchDone)
	}
	if stopExposition != nil {
		stopExposition()
	}
}

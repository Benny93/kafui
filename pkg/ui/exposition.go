package ui

import (
	"context"
	"net/http"
	"time"

	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/metrics"
	"github.com/Benny93/kafui/pkg/ui/shared"
)

// startExpositionServer starts the flag-gated Prometheus exposition endpoint
// (MM-16) when addr is non-empty, serving the collector's current snapshots. It
// returns a stop function (nil when disabled) to be called after the program
// exits. A cluster exports only when its metrics config has ExpositionEnabled.
func startExpositionServer(addr string, collector *metrics.Collector, appCfg appconfig.Config) func() {
	if addr == "" || collector == nil {
		return nil
	}
	handler := metrics.NewExpositionHandler(collector, func(cluster string) bool {
		ext, ok := appCfg.Clusters[cluster]
		if !ok {
			return true // unconfigured clusters export by default
		}
		return ext.MetricsSettings().ExpositionEnabled
	})
	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			shared.Log.Error("metrics exposition server stopped", "addr", addr, "err", err)
		}
	}()
	shared.Log.Info("metrics exposition endpoint listening", "addr", addr)
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
}

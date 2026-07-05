package kafds

import (
	"sync"

	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/serde"
)

// loadSerdeConfigs returns the per-cluster serde bindings for a context. It is a
// var so tests can stub it without touching disk.
var loadSerdeConfigs = func(context string) []serde.SerdeConfig {
	cfg, err := appconfig.Load(appconfig.DefaultPath())
	if err != nil {
		return nil
	}
	if ext, ok := cfg.Clusters[context]; ok {
		return ext.Serdes
	}
	return nil
}

var (
	serdeRegistryMu    sync.Mutex
	cachedSerdeRegistry *serde.Registry
)

// invalidateSerdeRegistry drops the cached registry so it is rebuilt against the
// new cluster's schema cache / serde config. Called on context switch/reload.
func invalidateSerdeRegistry() {
	serdeRegistryMu.Lock()
	cachedSerdeRegistry = nil
	serdeRegistryMu.Unlock()
}

// getSerdeRegistry returns the process-cached serde registry for the active
// cluster, building it on first use. The schema-registry serde reuses the
// existing Avro cache; Avro decode therefore flows through the same path as
// before. On a build error it falls back to a built-in-only registry so
// decoding still works.
func getSerdeRegistry() *serde.Registry {
	serdeRegistryMu.Lock()
	defer serdeRegistryMu.Unlock()
	if cachedSerdeRegistry != nil {
		return cachedSerdeRegistry
	}

	decode := func(data []byte) ([]byte, error) {
		cache, err := getOrInitSchemaCache()
		if err != nil {
			return nil, err
		}
		return avroDecodeWithCache(data, cache)
	}

	context := ""
	if currentCluster != nil {
		context = currentCluster.Name
	}
	reg, err := serde.BuildRegistry(decode, loadSerdeConfigs(context))
	if err != nil {
		// Bad config (e.g. a missing descriptor file) must not break decoding.
		reg, _ = serde.BuildRegistry(decode, nil)
	}
	cachedSerdeRegistry = reg
	return reg
}

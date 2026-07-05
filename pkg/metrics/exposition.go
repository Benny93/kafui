package metrics

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/Benny93/kafui/pkg/api"
)

// ExpositionContentType is the Prometheus text exposition format content type.
const ExpositionContentType = "text/plain; version=0.0.4; charset=utf-8"

// SnapshotLister provides the cached per-cluster snapshots the exposition
// endpoint renders. *Collector satisfies it.
type SnapshotLister interface {
	List() []api.ClusterMetrics
}

// NewExpositionHandler builds the flag-gated Prometheus exposition handler
// (MM-16). enabledFor reports whether a cluster opts in to exposition; clusters
// for which it returns false are 404 on their own path and excluded from the
// merged global endpoint.
//
//   - GET /metrics           merged families across all opted-in clusters
//   - GET /metrics/{cluster} a single cluster (404 unknown or opted-out)
func NewExpositionHandler(src SnapshotLister, enabledFor func(cluster string) bool) http.Handler {
	if enabledFor == nil {
		enabledFor = func(string) bool { return true }
	}
	h := func(w http.ResponseWriter, r *http.Request) {
		// A trailing path segment selects a single cluster; bare /metrics merges all.
		name := strings.TrimPrefix(r.URL.Path, "/metrics")
		name = strings.TrimPrefix(name, "/")
		all := src.List()
		var selected []api.ClusterMetrics
		if name == "" {
			for _, cm := range all {
				if enabledFor(cm.Cluster) {
					selected = append(selected, cm)
				}
			}
		} else {
			var found *api.ClusterMetrics
			for i := range all {
				if all[i].Cluster == name {
					found = &all[i]
					break
				}
			}
			if found == nil || !enabledFor(name) {
				http.NotFound(w, r)
				return
			}
			selected = []api.ClusterMetrics{*found}
		}
		w.Header().Set("Content-Type", ExpositionContentType)
		writeExposition(w, selected)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", h)
	mux.HandleFunc("/metrics/", h)
	return mux
}

// expoSample is one labeled data point within a family.
type expoSample struct {
	labels []label
	value  float64
}

type label struct{ k, v string }

// family accumulates same-named samples across clusters (families are merged).
type family struct {
	name, help string
	samples    []expoSample
}

// writeExposition renders the given clusters as merged Prometheus text families.
// Every sample is labeled cluster_name; broker samples additionally carry
// broker_id.
func writeExposition(w io.Writer, clusters []api.ClusterMetrics) {
	var order []string
	fam := map[string]*family{}
	add := func(name, help string, value float64, labels ...label) {
		f, ok := fam[name]
		if !ok {
			f = &family{name: name, help: help}
			fam[name] = f
			order = append(order, name)
		}
		f.samples = append(f.samples, expoSample{labels: labels, value: value})
	}

	for _, cm := range clusters {
		cl := label{"cluster_name", cm.Cluster}
		add("kafui_cluster_brokers", "Number of brokers in the cluster.", float64(cm.BrokerCount), cl)
		add("kafui_cluster_topics", "Number of topics in the cluster.", float64(cm.TopicCount), cl)
		add("kafui_cluster_partitions", "Number of partitions in the cluster.", float64(cm.PartitionCount), cl)
		add("kafui_cluster_messages", "Approximate total message count.", float64(cm.MessageCount), cl)
		if cm.MessagesInPerSec >= 0 {
			add("kafui_messages_in_per_second", "Cluster message-in rate (msgs/s).", cm.MessagesInPerSec, cl)
		}
		if cm.BytesInPerSec >= 0 {
			add("kafui_bytes_in_per_second", "Cluster bytes-in rate (bytes/s).", cm.BytesInPerSec, cl)
		}
		if cm.BytesOutPerSec >= 0 {
			add("kafui_bytes_out_per_second", "Cluster bytes-out rate (bytes/s).", cm.BytesOutPerSec, cl)
		}
		for _, tm := range cm.Topics {
			tl := label{"topic", tm.Name}
			add("kafui_topic_messages", "Approximate topic message count.", float64(tm.MessageCount), cl, tl)
			if tm.MessagesInPerSec >= 0 {
				add("kafui_topic_messages_in_per_second", "Topic message-in rate (msgs/s).", tm.MessagesInPerSec, cl, tl)
			}
		}
		for _, bm := range cm.Brokers {
			bl := label{"broker_id", fmt.Sprintf("%d", bm.ID)}
			add("kafui_broker_leaders", "Leader partition count on the broker.", float64(bm.LeaderCount), cl, bl)
			add("kafui_broker_replicas", "Replica count on the broker.", float64(bm.ReplicaCount), cl, bl)
			add("kafui_broker_segment_bytes", "Total log segment bytes on the broker.", float64(bm.SegmentSize), cl, bl)
		}
	}

	for _, name := range order {
		f := fam[name]
		fmt.Fprintf(w, "# HELP %s %s\n", f.name, f.help)
		fmt.Fprintf(w, "# TYPE %s gauge\n", f.name)
		for _, s := range f.samples {
			fmt.Fprintf(w, "%s%s %s\n", f.name, formatLabels(s.labels), formatValue(s.value))
		}
	}
}

func formatLabels(labels []label) string {
	if len(labels) == 0 {
		return ""
	}
	// Sort for deterministic output.
	sorted := make([]label, len(labels))
	copy(sorted, labels)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].k < sorted[j].k })
	parts := make([]string, len(sorted))
	for i, l := range sorted {
		parts[i] = fmt.Sprintf("%s=%q", l.k, l.v)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func formatValue(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", v), "0"), ".")
}

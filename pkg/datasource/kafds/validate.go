package kafds

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/config"
	"golang.org/x/oauth2/clientcredentials"
)

// probeTimeout bounds every network probe (broker metadata listing and HTTP
// service checks). Kept short so a broken candidate fails fast in the wizard.
const probeTimeout = 5 * time.Second

// loadClusterExtension resolves the kafui overlay entry for a cluster name. It is
// a package variable so tests can substitute an in-memory config without disk.
// The kaf config file (~/.kaf/config) is never read or written here.
var loadClusterExtension = func(name string) appconfig.ClusterExtension {
	cfg, err := appconfig.Load(appconfig.DefaultPath())
	if err != nil {
		return appconfig.ClusterExtension{}
	}
	return cfg.Clusters[name]
}

// ValidateCandidate probes every cluster in a candidate configuration without
// persisting anything. For each cluster it independently checks the broker
// connection, the schema registry (when configured) and each configured
// connect/ksql/metrics endpoint. TLS material is opened and parsed first; a load
// failure is reported as that cluster's error without attempting a connection.
// An empty candidate returns an empty report.
func (kp KafkaDataSourceKaf) ValidateCandidate(ctx context.Context, candidate appconfig.Config) api.ValidationReport {
	report := api.ValidationReport{}
	names := make([]string, 0, len(candidate.Clusters))
	for name := range candidate.Clusters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		report.Clusters = append(report.Clusters, api.ClusterValidation{
			Cluster: name,
			Results: kp.validateCluster(ctx, candidate.Clusters[name]),
		})
	}
	return report
}

// validateCluster probes a single cluster's components and returns one result
// per component. TLS is loaded first and short-circuits the whole cluster on
// failure (no connection is attempted).
func (kp KafkaDataSourceKaf) validateCluster(ctx context.Context, ext appconfig.ClusterExtension) []api.ValidationResult {
	// 1. TLS material FIRST — a load/parse failure short-circuits before connect.
	tlsConf, err := buildProbeTLS(ext.TLS)
	if err != nil {
		return []api.ValidationResult{{Component: "tls", OK: false, Err: err.Error()}}
	}

	var results []api.ValidationResult

	// 2. Broker probe: short-lived admin client + metadata listing.
	results = append(results, kp.probeBroker(ext, tlsConf))

	// 3. Schema registry (when configured).
	if url := firstURL(ext.SchemaRegistryURL); url != "" {
		results = append(results, httpProbe(ctx, "schema-registry", url+"/subjects",
			ext.SchemaRegistryUsername, ext.SchemaRegistryPassword, tlsInsecure(ext.TLS)))
	}

	// 4. Connect services (when configured).
	for _, c := range ext.Connect {
		if addr := strings.TrimRight(strings.TrimSpace(c.Address), "/"); addr != "" {
			results = append(results, httpProbe(ctx, "connect:"+c.Name, addr+"/connectors",
				c.Username, c.Password, false))
		}
	}

	// 5. ksqlDB (when configured).
	if ext.Ksql != nil {
		if url := firstURL(ext.Ksql.URL); url != "" {
			results = append(results, httpProbe(ctx, "ksql", url+"/info",
				ext.Ksql.Username, ext.Ksql.Password, false))
		}
	}

	// 6. Metrics store (when a URL is configured).
	if url := metricsURL(ext.Metrics); url != "" {
		results = append(results, httpProbe(ctx, "metrics", url, "", "", false))
	}

	return results
}

// probeBroker builds a reduced-retry sarama config and lists metadata via a
// short-lived admin client created through the injectable client factory.
func (kp KafkaDataSourceKaf) probeBroker(ext appconfig.ClusterExtension, tlsConf *tls.Config) api.ValidationResult {
	const component = "broker"
	if len(ext.Brokers) == 0 {
		return api.ValidationResult{Component: component, OK: false, Err: "no brokers configured"}
	}
	sc, err := probeSaramaConfig(ext, tlsConf)
	if err != nil {
		return api.ValidationResult{Component: component, OK: false, Err: err.Error()}
	}
	admin, err := kp.clientFactory.CreateClusterAdmin(ext.Brokers, sc)
	if err != nil {
		return api.ValidationResult{Component: component, OK: false, Err: err.Error()}
	}
	defer admin.Close()
	if _, err := admin.ListTopics(); err != nil {
		return api.ValidationResult{Component: component, OK: false, Err: err.Error()}
	}
	return api.ValidationResult{Component: component, OK: true}
}

// probeSaramaConfig builds a sarama config with reduced retry/timeout for a fast
// one-shot metadata probe, applying the candidate's SASL and (already-parsed) TLS.
func probeSaramaConfig(ext appconfig.ClusterExtension, tlsConf *tls.Config) (*sarama.Config, error) {
	sc := sarama.NewConfig()
	sc.Version = sarama.V1_1_0_0
	if ext.KafkaVersion != "" {
		v, err := sarama.ParseKafkaVersion(ext.KafkaVersion)
		if err != nil {
			return nil, fmt.Errorf("parse kafka version: %w", err)
		}
		sc.Version = v
	}
	// Reduced retry/timeout: fail fast rather than block the wizard.
	sc.Metadata.Retry.Max = 1
	sc.Metadata.Retry.Backoff = 200 * time.Millisecond
	sc.Metadata.Full = false
	sc.Net.DialTimeout = probeTimeout
	sc.Net.ReadTimeout = probeTimeout
	sc.Net.WriteTimeout = probeTimeout
	sc.Admin.Timeout = probeTimeout

	sslEnabled := ext.SecurityProtocol == "SSL" || ext.SecurityProtocol == "SASL_SSL" || (tlsConf != nil && ext.SecurityProtocol == "")
	if tlsConf != nil && sslEnabled {
		sc.Net.TLS.Enable = true
		sc.Net.TLS.Config = tlsConf
	}
	if err := applyProbeSASL(sc, ext.SASL); err != nil {
		return nil, err
	}
	return sc, nil
}

// applyProbeSASL configures broker SASL from the candidate's mechanism.
func applyProbeSASL(sc *sarama.Config, s *appconfig.SASLConfig) error {
	if s == nil || s.Mechanism == "" {
		return nil
	}
	sc.Net.SASL.Enable = true
	switch s.Mechanism {
	case "SCRAM-SHA-512":
		sc.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
		sc.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512)
		sc.Net.SASL.User = s.Username
		sc.Net.SASL.Password = s.Password
	case "SCRAM-SHA-256":
		sc.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
		sc.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA256)
		sc.Net.SASL.User = s.Username
		sc.Net.SASL.Password = s.Password
	case "OAUTHBEARER":
		sc.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeOAuth)
		sc.Net.SASL.TokenProvider = &probeTokenProvider{cfg: &clientcredentials.Config{
			ClientID:     s.ClientID,
			ClientSecret: s.ClientSecret,
			TokenURL:     s.TokenURL,
		}}
	default: // PLAIN
		sc.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypePlaintext)
		sc.Net.SASL.User = s.Username
		sc.Net.SASL.Password = s.Password
	}
	return nil
}

// probeTokenProvider is a non-singleton OAUTHBEARER token source used only for
// candidate probing (the production path uses newTokenProvider).
type probeTokenProvider struct {
	cfg *clientcredentials.Config
}

func (p *probeTokenProvider) Token() (*sarama.AccessToken, error) {
	t, err := p.cfg.Token(context.Background())
	if err != nil {
		return nil, err
	}
	return &sarama.AccessToken{Token: t.AccessToken}, nil
}

// buildProbeTLS opens and parses the candidate's CA/cert/key files. A missing or
// malformed file is a hard error (returned before any connection is attempted).
// Returns (nil, nil) when no TLS material is configured.
func buildProbeTLS(t *appconfig.TLSConfig) (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}
	if t.CAPath == "" && t.CertPath == "" && t.KeyPath == "" {
		if t.Insecure {
			return &tls.Config{InsecureSkipVerify: true}, nil
		}
		return nil, nil
	}
	conf := &tls.Config{InsecureSkipVerify: t.Insecure}
	if t.CAPath != "" {
		caCert, err := os.ReadFile(t.CAPath)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("parse CA file %s: no certificates found", t.CAPath)
		}
		conf.RootCAs = pool
	}
	if t.CertPath != "" || t.KeyPath != "" {
		if t.CertPath == "" || t.KeyPath == "" {
			return nil, fmt.Errorf("both client cert and key paths are required")
		}
		cert, err := tls.LoadX509KeyPair(t.CertPath, t.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client key pair: %w", err)
		}
		conf.Certificates = []tls.Certificate{cert}
	}
	return conf, nil
}

func tlsInsecure(t *appconfig.TLSConfig) bool {
	return t != nil && t.Insecure
}

// httpProbe performs a GET against url and reports OK on a 2xx status.
func httpProbe(ctx context.Context, component, url, user, pass string, insecure bool) api.ValidationResult {
	res := api.ValidationResult{Component: component}
	client := &http.Client{Timeout: probeTimeout}
	if insecure {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	resp, err := client.Do(req)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		res.Err = resp.Status
		return res
	}
	res.OK = true
	return res
}

// firstURL returns the first non-empty, trailing-slash-trimmed entry of a
// possibly comma-separated URL list.
func firstURL(list string) string {
	for _, u := range strings.Split(list, ",") {
		if u = strings.TrimRight(strings.TrimSpace(u), "/"); u != "" {
			return u
		}
	}
	return ""
}

func metricsURL(m map[string]string) string {
	if m == nil {
		return ""
	}
	for _, k := range []string{"url", "endpoint", "address"} {
		if v := strings.TrimSpace(m[k]); v != "" {
			return v
		}
	}
	return ""
}

// clusterExtensionFor builds a probe-ready extension for a named cluster by
// overlaying the kafui entry on the live kaf cluster (when the kafui entry is
// not fully self-defined). It never writes to disk.
func (kp KafkaDataSourceKaf) clusterExtensionFor(name string) (appconfig.ClusterExtension, error) {
	ext := loadClusterExtension(name)
	if ext.IsFullyDefined() {
		return ext, nil
	}
	for _, c := range cfg.Clusters {
		if c.Name != name {
			continue
		}
		ext.Brokers = c.Brokers
		ext.SecurityProtocol = c.SecurityProtocol
		ext.KafkaVersion = c.Version
		if c.SASL != nil {
			ext.SASL = &appconfig.SASLConfig{
				Mechanism:    c.SASL.Mechanism,
				Username:     c.SASL.Username,
				Password:     c.SASL.Password,
				ClientID:     c.SASL.ClientID,
				ClientSecret: c.SASL.ClientSecret,
				TokenURL:     c.SASL.TokenURL,
			}
		}
		if c.TLS != nil {
			ext.TLS = &appconfig.TLSConfig{
				CAPath:   c.TLS.Cafile,
				CertPath: c.TLS.Clientfile,
				KeyPath:  c.TLS.Clientkeyfile,
				Insecure: c.TLS.Insecure,
			}
		}
		ext.SchemaRegistryURL = c.SchemaRegistryURL
		if cr := c.SchemaRegistryCredentials; cr != nil {
			ext.SchemaRegistryUsername = cr.Username
			ext.SchemaRegistryPassword = cr.Password
		}
		return ext, nil
	}
	return ext, fmt.Errorf("cluster with name '%s' not found", name)
}

// kafClusterFromExtension converts a fully-kafui-defined cluster to the kaf
// in-memory type so it can be merged into the running cluster list on Reload.
func kafClusterFromExtension(name string, ext appconfig.ClusterExtension) *config.Cluster {
	c := &config.Cluster{
		Name:              name,
		Version:           ext.KafkaVersion,
		Brokers:           ext.Brokers,
		SecurityProtocol:  ext.SecurityProtocol,
		SchemaRegistryURL: ext.SchemaRegistryURL,
	}
	if ext.SASL != nil {
		c.SASL = &config.SASL{
			Mechanism:    ext.SASL.Mechanism,
			Username:     ext.SASL.Username,
			Password:     ext.SASL.Password,
			ClientID:     ext.SASL.ClientID,
			ClientSecret: ext.SASL.ClientSecret,
			TokenURL:     ext.SASL.TokenURL,
		}
	}
	if ext.TLS != nil {
		c.TLS = &config.TLS{
			Cafile:        ext.TLS.CAPath,
			Clientfile:    ext.TLS.CertPath,
			Clientkeyfile: ext.TLS.KeyPath,
			Insecure:      ext.TLS.Insecure,
		}
	}
	if ext.SchemaRegistryUsername != "" || ext.SchemaRegistryPassword != "" {
		c.SchemaRegistryCredentials = &config.SchemaRegistryCredentials{
			Username: ext.SchemaRegistryUsername,
			Password: ext.SchemaRegistryPassword,
		}
	}
	return c
}

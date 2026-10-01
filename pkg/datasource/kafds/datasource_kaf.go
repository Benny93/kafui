package kafds

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"syscall"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/serde"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/avro"
	"github.com/birdayz/kaf/pkg/config"
	//"github.com/birdayz/kaf/pkg/proto"
)

type KafkaDataSourceKaf struct {
	clientFactory KafkaClientFactory
	configManager ConfigManager
}

// NewKafkaDataSourceKaf creates a new instance with default dependencies
func NewKafkaDataSourceKaf() *KafkaDataSourceKaf {
	return &KafkaDataSourceKaf{
		clientFactory: kafkaClientFactory,
		configManager: configManager,
	}
}

// NewKafkaDataSourceKafWithDeps creates a new instance with custom dependencies for testing
func NewKafkaDataSourceKafWithDeps(clientFactory KafkaClientFactory, configManager ConfigManager) *KafkaDataSourceKaf {
	return &KafkaDataSourceKaf{
		clientFactory: clientFactory,
		configManager: configManager,
	}
}

var cfgFile string

func (kp *KafkaDataSourceKaf) Init(cfgOption string) {
	if cfgOption != "" {
		cfgFile = cfgOption
	}
	onInit()
}

// GetTopicNames returns only the topic names using a lightweight Sarama client
// metadata request. This is faster than GetTopics() because it skips the
// per-topic DescribeConfigs that ListTopics() adds.
func (kp KafkaDataSourceKaf) GetTopicNames() ([]string, error) {
	client, err := getSharedClient()
	if err != nil {
		return nil, err
	}
	if err := client.RefreshMetadata(); err != nil {
		return nil, fmt.Errorf("failed to refresh metadata: %w", err)
	}
	return client.Topics()
}

// GetTopics retrieves a list of Kafka topics. It uses ListTopics, which also
// sends one DescribeConfigs for every topic: the UI reads Topic.ConfigEntries
// (the "N configs" column of the topic list, and the topic page's config view
// and cleanup.policy check), so the configs cannot be dropped here. Callers
// that only need names should use GetTopicNames.
func (kp KafkaDataSourceKaf) GetTopics() (map[string]api.Topic, error) {

	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}
	topicDetails, err := admin.ListTopics()
	if err != nil {
		return nil, err
	}

	//client := getClient()

	topics := make(map[string]api.Topic)

	for key, value := range topicDetails {
		/*
			var messageCount int64 = 0
			// Iterate over all partitions last offset to get the overall message count
			for i := 0; i < int(value.NumPartitions); i++ {
				offsets, err := getOffsets(client, key, int32(i))
				msgCount := offsets.newest - offsets.oldest
				if err == nil {
					messageCount += msgCount
				}
			}*/

		topics[key] = api.Topic{
			NumPartitions:     value.NumPartitions,
			ReplicationFactor: value.ReplicationFactor,
			ReplicaAssignment: value.ReplicaAssignment,
			ConfigEntries:     value.ConfigEntries,
			MessageCount:      -1,
		}
	}

	return topics, err
}

// GetTopicMessageCounts fetches the approximate message count for each topic by summing
// (newestOffset - oldestOffset) across all partitions. The offsets of all topics are
// fetched with one request per leader broker (see fetchOffsetsBatched) on the shared
// client. Partitions that fail individually are skipped so a partial result is always
// returned.
func (kp KafkaDataSourceKaf) GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error) {
	client, err := getSharedClient()
	if err != nil {
		return nil, err
	}

	parts := make(map[string][]int32, len(topics))
	for name, numPartitions := range topics {
		var ids []int32
		for i := int32(0); i < numPartitions; i++ {
			ids = append(ids, i)
		}
		parts[name] = ids
	}
	offs, _ := fetchOffsetsBatched(client, parts)

	counts := make(map[string]int64, len(topics))
	for name := range topics {
		var total int64
		for _, o := range offs[name] {
			total += o.newest - o.oldest
		}
		counts[name] = total
	}
	return counts, nil
}

func (kp KafkaDataSourceKaf) GetContext() string {
	// Check if cfg is properly initialized
	if cfg.Clusters == nil {
		return "default localhost:9092 (config not loaded)"
	}

	activeCluster := kp.configManager.GetActiveCluster(cfg)
	if activeCluster == nil {
		return "default localhost:9092"
	}
	return activeCluster.Name
}

// GetContexts retrieves a list of Kafka contexts
func (kp KafkaDataSourceKaf) GetContexts() ([]string, error) {
	// Logic to fetch the list of contexts from Kafka
	var contexts []string
	for _, cluster := range cfg.Clusters {

		contexts = append(contexts, cluster.Name)
	}
	return contexts, nil
}

// GetClusterDetails returns configuration details for the named cluster.
func (kp KafkaDataSourceKaf) GetClusterDetails(clusterName string) (api.ClusterInfo, error) {
	currentCtx := kp.GetContext()
	for _, cluster := range cfg.Clusters {
		if cluster.Name == clusterName {
			return api.ClusterInfo{
				Name:              cluster.Name,
				Brokers:           cluster.Brokers,
				SchemaRegistryURL: cluster.SchemaRegistryURL,
				IsCurrent:         cluster.Name == currentCtx,
			}, nil
		}
	}
	return api.ClusterInfo{}, fmt.Errorf("cluster with name '%s' not found", clusterName)
}

// Reload rebuilds the in-memory kaf config and active cluster from the effective
// kafui configuration, merging fully-kafui-defined clusters into the loaded
// cluster list (replacing by name or appending), then invalidates caches
// (mirroring SetContext). It NEVER reads or writes ~/.kaf/config — the merge is
// entirely in memory. Called after an in-UI config apply to take effect without
// restarting the process.
func (kp *KafkaDataSourceKaf) Reload(effective appconfig.Config) error {
	for name, ext := range effective.Clusters {
		if !ext.IsFullyDefined() {
			continue // overlay-only entry; it decorates an existing kaf cluster
		}
		kc := kafClusterFromExtension(name, ext)
		replaced := false
		for i, c := range cfg.Clusters {
			if c.Name == name {
				cfg.Clusters[i] = kc
				replaced = true
				break
			}
		}
		if !replaced {
			cfg.Clusters = append(cfg.Clusters, kc)
		}
	}

	// Re-resolve the active cluster: keep the current one if it still exists,
	// otherwise fall back to the first configured cluster.
	target := cfg.CurrentCluster
	if currentCluster != nil && currentCluster.Name != "" {
		target = currentCluster.Name
	}
	found := false
	for _, c := range cfg.Clusters {
		if c.Name == target {
			cc := *c
			currentCluster = &cc
			cfg.CurrentCluster = cc.Name
			found = true
			break
		}
	}
	if !found && len(cfg.Clusters) > 0 {
		cc := *cfg.Clusters[0]
		currentCluster = &cc
		cfg.CurrentCluster = cc.Name
	}
	// The active cluster is now cfg.CurrentCluster. A --cluster override
	// would otherwise keep winning in GetContext (see SetContext).
	cfg.ClusterOverride = ""

	// Invalidate caches (mirror SetContext).
	cachedSchemaCache = nil
	invalidateSerdeRegistry()
	resetSharedClients()
	return nil
}

func (kp KafkaDataSourceKaf) SetContext(contextName string) error {
	// Only update the in-memory currentCluster pointer — never write to disk.
	// Calling cfg.SetCurrentCluster() would truncate ~/.kaf/config and re-serialize
	// it, which strips TLS cert paths due to missing YAML tags in the kaf library.
	for _, cluster := range cfg.Clusters {
		if cluster.Name == contextName {
			currentCluster = cluster
			cfg.CurrentCluster = contextName
			// kaf's ActiveCluster (used by GetContext) prefers ClusterOverride,
			// so a --cluster override must be dropped on a switch, or GetContext
			// (audit records, caches, the header) keeps naming the old cluster.
			cfg.ClusterOverride = ""
			cachedSchemaCache = nil // invalidate schema cache on cluster switch
			invalidateSerdeRegistry()
			resetSharedClients()
			return nil
		}
	}
	return fmt.Errorf("cluster with name '%s' not found", contextName)
}

func (kp KafkaDataSourceKaf) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}

	// ListConsumerGroups is a single fast broker round-trip that returns every
	// group name and its protocol type.  DescribeConsumerGroups is intentionally
	// skipped here: on large clusters it can take tens of seconds (or hang
	// indefinitely) because it fan-outs to every partition coordinator.
	groups, err := admin.ListConsumerGroups()
	if err != nil {
		return nil, err
	}

	shared.Log.Info("GetConsumerGroups: raw list", "count", len(groups))

	finalGroups := make([]api.ConsumerGroup, 0, len(groups))
	for name, protocol := range groups {
		state := protocol
		if state == "" {
			state = "consumer"
		}
		finalGroups = append(finalGroups, api.ConsumerGroup{
			Name:      name,
			State:     state,
			Consumers: 0,
		})
	}

	sort.Slice(finalGroups, func(i, j int) bool {
		return finalGroups[i].Name < finalGroups[j].Name
	})

	return finalGroups, nil
}

func (kp KafkaDataSourceKaf) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {
	DoConsume(ctx, topicName, flags, handleMessage, onError)
	return nil
}

// GetACLs implements api.KafkaDataSource (the match-any case of GetACLsFiltered).
func (kp KafkaDataSourceKaf) GetACLs() ([]api.ACLEntry, error) {
	return kp.GetACLsFiltered(api.ACLFilter{})
}

// GetMessageSchemaInfo implements api.KafkaDataSource
func (kp KafkaDataSourceKaf) GetMessageSchemaInfo(keySchemaID, valueSchemaID string) (*api.MessageSchemaInfo, error) {
	schemaInfo := &api.MessageSchemaInfo{}

	if keySchemaID != "" {
		if keySchema, err := kp.fetchSchemaInfo(keySchemaID); err == nil && keySchema != nil {
			schemaInfo.KeySchema = keySchema
		}
	}

	if valueSchemaID != "" {
		if valueSchema, err := kp.fetchSchemaInfo(valueSchemaID); err == nil && valueSchema != nil {
			schemaInfo.ValueSchema = valueSchema
		}
	}

	if schemaInfo.KeySchema == nil && schemaInfo.ValueSchema == nil {
		return nil, nil
	}
	return schemaInfo, nil
}

// DecodeMessage decodes Avro-encoded raw bytes stored in msg.RawKey / msg.RawValue
// into human-readable strings. Messages without raw bytes are returned unchanged.
// The schema registry client is shared across calls (see cachedSchemaCache).
func (kp KafkaDataSourceKaf) DecodeMessage(_ context.Context, msg api.Message) (api.Message, error) {
	if len(msg.RawKey) == 0 && len(msg.RawValue) == 0 {
		return msg, nil
	}
	reg := getSerdeRegistry()
	if len(msg.RawKey) > 0 {
		text, name, _ := serde.Decode(reg, "", msg.RawKey)
		msg.Key, msg.KeySerde = text, name
	}
	if len(msg.RawValue) > 0 {
		text, name, _ := serde.Decode(reg, "", msg.RawValue)
		msg.Value, msg.ValueSerde = text, name
	}
	return msg, nil
}

// ListSerdes returns the names of serdes available for decoding, driven by the
// active cluster's registry (built-ins + configured). (MSG-18)
func (kp KafkaDataSourceKaf) ListSerdes() []string {
	return getSerdeRegistry().Names()
}

func getConfig() (saramaConfig *sarama.Config, e error) {
	saramaConfig = sarama.NewConfig()
	saramaConfig.Version = sarama.V1_1_0_0
	saramaConfig.Producer.Return.Successes = true

	if currentCluster == nil {
		return nil, fmt.Errorf("no Kafka cluster configured")
	}

	cluster := currentCluster
	if cluster.Version != "" {
		parsedVersion, err := sarama.ParseKafkaVersion(cluster.Version)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse Kafka version: %v\n", err)
		}
		saramaConfig.Version = parsedVersion
	}
	if cluster.SASL != nil {
		saramaConfig.Net.SASL.Enable = true
		// Token-based mechanisms carry no username/password pair.
		if cluster.SASL.Mechanism != "OAUTHBEARER" && cluster.SASL.Mechanism != "AWS_MSK_IAM" {
			saramaConfig.Net.SASL.User = cluster.SASL.Username
			saramaConfig.Net.SASL.Password = cluster.SASL.Password
		}
		saramaConfig.Net.SASL.Version = cluster.SASL.Version
	}
	if cluster.TLS != nil && cluster.SecurityProtocol != "SASL_SSL" {
		saramaConfig.Net.TLS.Enable = true
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cluster.TLS.Insecure,
		}

		if cluster.TLS.Cafile != "" {
			caCert, err := ioutil.ReadFile(cluster.TLS.Cafile)
			if err != nil {
				return nil, fmt.Errorf("Unable to read Cafile :%v\n", err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caCertPool
		}

		if cluster.TLS.Clientfile != "" && cluster.TLS.Clientkeyfile != "" {
			clientCert, err := ioutil.ReadFile(cluster.TLS.Clientfile)
			if err != nil {
				return nil, fmt.Errorf("Unable to read Clientfile :%v\n", err)
			}
			clientKey, err := ioutil.ReadFile(cluster.TLS.Clientkeyfile)
			if err != nil {
				return nil, fmt.Errorf("Unable to read Clientkeyfile :%v\n", err)
			}

			cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
			if err != nil {
				return nil, fmt.Errorf("Unable to create KeyPair: %v\n", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}

			// nolint
			tlsConfig.BuildNameToCertificate()
		}
		saramaConfig.Net.TLS.Config = tlsConfig
	}
	if cluster.SecurityProtocol == "SASL_SSL" {
		saramaConfig.Net.TLS.Enable = true
		if cluster.TLS != nil {
			tlsConfig := &tls.Config{
				InsecureSkipVerify: cluster.TLS.Insecure,
			}
			if cluster.TLS.Cafile != "" {
				caCert, err := ioutil.ReadFile(cluster.TLS.Cafile)
				if err != nil {
					// Never os.Exit here: this runs inside a tea.Cmd, and exiting
					// would leave the terminal in alt-screen/raw mode.
					return nil, fmt.Errorf("unable to read TLS CA file %q: %w", cluster.TLS.Cafile, err)
				}
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				tlsConfig.RootCAs = caCertPool
			}
			saramaConfig.Net.TLS.Config = tlsConfig

		} else {
			saramaConfig.Net.TLS.Config = &tls.Config{InsecureSkipVerify: false}
		}
	}
	if cluster.SecurityProtocol == "SASL_SSL" || cluster.SecurityProtocol == "SASL_PLAINTEXT" {
		if cluster.SASL == nil {
			return nil, fmt.Errorf("cluster %q uses %s but has no SASL configuration", cluster.Name, cluster.SecurityProtocol)
		}
		if cluster.SASL.Mechanism == "SCRAM-SHA-512" {
			saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
			saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512)
		} else if cluster.SASL.Mechanism == "SCRAM-SHA-256" {
			saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
			saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA256)
		} else if cluster.SASL.Mechanism == "OAUTHBEARER" {
			//Here setup get token function
			saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeOAuth)
			saramaConfig.Net.SASL.TokenProvider = newTokenProvider()

		} else if cluster.SASL.Mechanism == "AWS_MSK_IAM" {
			// MSK IAM is an OAUTHBEARER-flavored mechanism: the SigV4-signed
			// presigned URL travels in the token field (issue #3).
			saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeOAuth)
			provider, err := newMSKTokenProvider()
			if err != nil {
				return nil, err
			}
			saramaConfig.Net.SASL.TokenProvider = provider
		}
	}
	return saramaConfig, nil
}

// InitTUIWriters routes the sarama Kafka client logger to the structured
// logger so nothing corrupts the TUI. Call this once before starting
// tea.NewProgram.
func InitTUIWriters() {
	w := shared.NewSlogWriter(shared.Log)
	// Always route sarama logs to file — it is very chatty on reconnects.
	sarama.Logger = log.New(w, "[sarama] ", 0)
}

var cfg config.Config
var currentCluster *config.Cluster

var (
	brokersFlag       []string
	schemaRegistryURL string
	verbose           bool
	clusterOverride   string
)

// SetOverrides applies CLI overrides before Init/onInit runs. Empty/nil values
// leave the corresponding config value untouched. Kafui's own CLI calls this.
func SetOverrides(brokers []string, schemaRegistry, cluster string, verboseLogging bool) {
	if len(brokers) > 0 {
		brokersFlag = brokers
	}
	if schemaRegistry != "" {
		schemaRegistryURL = schemaRegistry
	}
	if cluster != "" {
		clusterOverride = cluster
	}
	verbose = verboseLogging
}

// protectConfigFile is intentionally not called at runtime. The primary
// protection against config corruption is that SetContext() never calls
// cfg.Write() or cfg.SetCurrentCluster(). This function is kept for
// reference but should not be invoked automatically.
func protectConfigFile(cfgPath string) error {
	path := cfgPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = home + "/.kaf/config"
	}
	return os.Chmod(path, 0444)
}

// InitFromConfig reads the kaf config file at cfgPath (pass "" to use the
// default ~/.kaf/config) and sets the active cluster, like Init does but
// without the CLI overrides; use it in examples and standalone programs.
func InitFromConfig(cfgPath string) error {
	var err error
	cfg, err = config.ReadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("reading kaf config: %w", err)
	}
	cluster := cfg.ActiveCluster()
	if cluster != nil {
		currentCluster = cluster
	} else {
		currentCluster = &config.Cluster{
			Brokers: []string{"localhost:9092"},
		}
	}
	return nil
}

func onInit() {
	var err error
	cfg, err = config.ReadConfig(cfgFile)
	if err != nil {
		// Instead of panicking, create a default config
		shared.Log.Warn("could not read config file, using defaults", "err", err)
		cfg = config.Config{
			Clusters: []*config.Cluster{},
		}
	}

	cfg.ClusterOverride = clusterOverride

	cluster := cfg.ActiveCluster()
	if cluster != nil {
		// Use active cluster from config
		currentCluster = cluster
	} else {
		// Create sane default if not configured
		currentCluster = &config.Cluster{
			Brokers: []string{"localhost:9092"},
		}
	}

	// Any set flags override the configuration
	if schemaRegistryURL != "" {
		currentCluster.SchemaRegistryURL = schemaRegistryURL
		currentCluster.SchemaRegistryCredentials = nil
	}

	if brokersFlag != nil {
		currentCluster.Brokers = brokersFlag
	}
	// sarama.Logger is set by InitTUIWriters() before the TUI starts.
}

// sharedClients holds the datasource-owned sarama ClusterAdmin and Client for
// the active cluster. Both are created lazily on first use and then reused by
// every call. Creating one per call used to dial the brokers and fetch
// metadata each time, and most callers never closed it, so the UI's 5s polling
// leaked a client (sockets plus a metadata-updater goroutine) per call.
// resetSharedClients drops both on SetContext and Reload.
//
// Each handle remembers the factory and *config.Cluster it was built for. A
// change of either (a cluster switch, or a test swapping the factory or the
// cluster) rebuilds it, so a stale handle is never handed out.
//
// mu is never held across network I/O: a dial runs unlocked and concurrent
// callers wait on the one in flight (see acquireShared), so SetContext and
// Reload, which run on the Bubble Tea Update goroutine, never wait for a dial.
var sharedClients struct {
	mu sync.Mutex

	// epoch is bumped by resetSharedClients. A dial started in an older epoch
	// is discarded instead of installed.
	epoch uint64
	// gen numbers installed handles, so a failure can drop exactly the handle
	// it happened on (see sharedClusterAdmin.check).
	gen uint64

	admin  sharedSlot[ClusterAdminInterface]
	client sharedSlot[sarama.Client]
}

// sharedSlot is one cached handle plus the dial currently building it.
type sharedSlot[T io.Closer] struct {
	handle  T
	live    bool
	gen     uint64
	factory KafkaClientFactory
	cluster *config.Cluster
	dial    *sharedDial[T]
}

// sharedDial is a dial in flight. Callers that want the same cluster and
// factory wait on done instead of dialing again.
type sharedDial[T io.Closer] struct {
	factory KafkaClientFactory
	cluster *config.Cluster
	epoch   uint64
	done    chan struct{}
	handle  T
	gen     uint64
	err     error
}

// errClusterChangedWhileConnecting is returned to callers whose dial finished
// after a cluster switch or reload: the handle it built is closed, not used.
var errClusterChangedWhileConnecting = errors.New("the active Kafka cluster changed while connecting")

// take empties the slot (and forgets any dial in flight), returning the handle
// it held, if any. The caller holds sharedClients.mu.
func (slot *sharedSlot[T]) take() (T, bool) {
	h, live := slot.handle, slot.live
	*slot = sharedSlot[T]{}
	return h, live
}

// closeShared closes a handle off the caller's goroutine: sarama's
// client.Close waits for the background metadata updater, and the caller may
// be the Bubble Tea Update goroutine.
func closeShared(c io.Closer) {
	if c == nil {
		return
	}
	go func() { _ = c.Close() }()
}

// acquireShared returns the slot's handle for the active cluster, dialing it
// when there is none. The lock is released for the dial; only one dial per
// cluster and factory runs at a time, and the result is installed only if the
// active cluster, the factory and the reset epoch are unchanged.
func acquireShared[T io.Closer](slot *sharedSlot[T], noCluster error, dial func(KafkaClientFactory, *config.Cluster) (T, error)) (T, uint64, error) {
	s := &sharedClients
	var zero T

	s.mu.Lock()
	cluster, factory := currentCluster, kafkaClientFactory
	if cluster == nil {
		s.mu.Unlock()
		return zero, 0, noCluster
	}
	if slot.live && slot.factory == factory && slot.cluster == cluster {
		h, gen := slot.handle, slot.gen
		s.mu.Unlock()
		return h, gen, nil
	}
	if d := slot.dial; d != nil && d.factory == factory && d.cluster == cluster && d.epoch == s.epoch {
		s.mu.Unlock()
		<-d.done
		return d.handle, d.gen, d.err
	}
	d := &sharedDial[T]{factory: factory, cluster: cluster, epoch: s.epoch, done: make(chan struct{})}
	slot.dial = d
	s.mu.Unlock()

	h, err := dial(factory, cluster)

	var stale io.Closer
	s.mu.Lock()
	if slot.dial == d {
		slot.dial = nil
	}
	switch {
	case err != nil:
		d.err = err
	case s.epoch != d.epoch || currentCluster != cluster || kafkaClientFactory != factory:
		stale, d.err = h, errClusterChangedWhileConnecting
	default:
		if slot.live {
			stale = slot.handle // built for another factory or cluster
		}
		s.gen++
		slot.handle, slot.live, slot.gen = h, true, s.gen
		slot.factory, slot.cluster = factory, cluster
		d.handle, d.gen = h, s.gen
	}
	s.mu.Unlock()
	close(d.done)

	if stale != nil {
		closeShared(stale)
	}
	return d.handle, d.gen, d.err
}

// dropSharedAdmin forgets and closes the cached admin, but only if gen is
// still the current one: a failure on an admin that was already replaced must
// not throw away its healthy successor.
func dropSharedAdmin(gen uint64) {
	s := &sharedClients
	s.mu.Lock()
	if !s.admin.live || s.admin.gen != gen {
		s.mu.Unlock()
		return
	}
	dial := s.admin.dial
	h, _ := s.admin.take()
	s.admin.dial = dial // a dial in flight is for a fresh handle; keep it
	s.mu.Unlock()
	closeShared(h)
}

// isConnectionError reports whether err means the connection to a broker is
// gone (as opposed to a Kafka error answer). sarama does not reopen a dead
// broker connection on the ListTopics/ListConsumerGroups paths, so the cached
// admin has to be rebuilt after such an error.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, sarama.ErrNotConnected) ||
		errors.Is(err, sarama.ErrClosedClient) ||
		errors.Is(err, sarama.ErrOutOfBrokers) ||
		errors.As(err, &netErr)
}

// sharedClusterAdmin is what getClusterAdmin hands out. Close is a no-op: the
// datasource owns the connection, and a caller that still defers Close must
// not tear it down while other tea.Cmds are using it. A connection error from
// any call drops the cached admin (if it is still this one), so the next
// getClusterAdmin reconnects instead of failing until a cluster switch.
type sharedClusterAdmin struct {
	ClusterAdminInterface
	gen uint64
}

func (sharedClusterAdmin) Close() error { return nil }

func (a sharedClusterAdmin) check(err error) error {
	if isConnectionError(err) {
		dropSharedAdmin(a.gen)
	}
	return err
}

func (a sharedClusterAdmin) ListTopics() (map[string]sarama.TopicDetail, error) {
	r, err := a.ClusterAdminInterface.ListTopics()
	return r, a.check(err)
}

func (a sharedClusterAdmin) ListConsumerGroups() (map[string]string, error) {
	r, err := a.ClusterAdminInterface.ListConsumerGroups()
	return r, a.check(err)
}

func (a sharedClusterAdmin) DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error) {
	r, err := a.ClusterAdminInterface.DescribeConsumerGroups(groups)
	return r, a.check(err)
}

func (a sharedClusterAdmin) ListAcls(filter sarama.AclFilter) ([]sarama.ResourceAcls, error) {
	r, err := a.ClusterAdminInterface.ListAcls(filter)
	return r, a.check(err)
}

func (a sharedClusterAdmin) CreateACLs(resourceACLs []*sarama.ResourceAcls) error {
	return a.check(a.ClusterAdminInterface.CreateACLs(resourceACLs))
}

func (a sharedClusterAdmin) DeleteACL(filter sarama.AclFilter, validateOnly bool) ([]sarama.MatchingAcl, error) {
	r, err := a.ClusterAdminInterface.DeleteACL(filter, validateOnly)
	return r, a.check(err)
}

func (a sharedClusterAdmin) DescribeClientQuotas(components []sarama.QuotaFilterComponent, strict bool) ([]sarama.DescribeClientQuotasEntry, error) {
	r, err := a.ClusterAdminInterface.DescribeClientQuotas(components, strict)
	return r, a.check(err)
}

func (a sharedClusterAdmin) AlterClientQuotas(entity []sarama.QuotaEntityComponent, op sarama.ClientQuotasOp, validateOnly bool) error {
	return a.check(a.ClusterAdminInterface.AlterClientQuotas(entity, op, validateOnly))
}

func (a sharedClusterAdmin) DescribeCluster() ([]*sarama.Broker, int32, error) {
	brokers, controllerID, err := a.ClusterAdminInterface.DescribeCluster()
	return brokers, controllerID, a.check(err)
}

func (a sharedClusterAdmin) DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
	r, err := a.ClusterAdminInterface.DescribeConfig(resource)
	return r, a.check(err)
}

func (a sharedClusterAdmin) IncrementalAlterConfig(resourceType sarama.ConfigResourceType, name string, entries map[string]sarama.IncrementalAlterConfigsEntry, validateOnly bool) error {
	return a.check(a.ClusterAdminInterface.IncrementalAlterConfig(resourceType, name, entries, validateOnly))
}

func (a sharedClusterAdmin) DescribeLogDirs(brokers []int32) (map[int32][]sarama.DescribeLogDirsResponseDirMetadata, error) {
	r, err := a.ClusterAdminInterface.DescribeLogDirs(brokers)
	return r, a.check(err)
}

func (a sharedClusterAdmin) DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error) {
	r, err := a.ClusterAdminInterface.DescribeTopics(topics)
	return r, a.check(err)
}

func (a sharedClusterAdmin) ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	r, err := a.ClusterAdminInterface.ListConsumerGroupOffsets(group, topicPartitions)
	return r, a.check(err)
}

func (a sharedClusterAdmin) DeleteConsumerGroup(group string) error {
	return a.check(a.ClusterAdminInterface.DeleteConsumerGroup(group))
}

func (a sharedClusterAdmin) DeleteConsumerGroupOffset(group string, topic string, partition int32) error {
	return a.check(a.ClusterAdminInterface.DeleteConsumerGroupOffset(group, topic, partition))
}

func (a sharedClusterAdmin) CreateTopic(topic string, detail *sarama.TopicDetail, validateOnly bool) error {
	return a.check(a.ClusterAdminInterface.CreateTopic(topic, detail, validateOnly))
}

func (a sharedClusterAdmin) DeleteTopic(topic string) error {
	return a.check(a.ClusterAdminInterface.DeleteTopic(topic))
}

func (a sharedClusterAdmin) CreatePartitions(topic string, count int32, assignment [][]int32, validateOnly bool) error {
	return a.check(a.ClusterAdminInterface.CreatePartitions(topic, count, assignment, validateOnly))
}

func (a sharedClusterAdmin) DeleteRecords(topic string, partitionOffsets map[int32]int64) error {
	return a.check(a.ClusterAdminInterface.DeleteRecords(topic, partitionOffsets))
}

func (a sharedClusterAdmin) AlterPartitionReassignments(topic string, assignment [][]int32) error {
	return a.check(a.ClusterAdminInterface.AlterPartitionReassignments(topic, assignment))
}

// sharedClient is what getSharedClient hands out. Close is a no-op for the
// same reason as sharedClusterAdmin.
type sharedClient struct{ sarama.Client }

func (sharedClient) Close() error { return nil }

// getClusterAdmin returns the cached ClusterAdmin for the active cluster,
// creating it on first use. It is safe for concurrent use. Callers must not
// Close it (Close is a no-op): the datasource closes it on a cluster switch or
// reload, or after a connection error.
func getClusterAdmin() (ClusterAdminInterface, error) {
	admin, gen, err := acquireShared(&sharedClients.admin,
		fmt.Errorf("no Kafka cluster configured. Please check your configuration or ensure Kafka is running"),
		func(factory KafkaClientFactory, cluster *config.Cluster) (ClusterAdminInterface, error) {
			cfg, err := getConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to get Kafka config: %v", err)
			}
			admin, err := factory.CreateClusterAdmin(cluster.Brokers, cfg)
			if err != nil {
				return nil, fmt.Errorf("unable to connect to Kafka cluster at %v: %v\nPlease ensure Kafka is running and accessible", cluster.Brokers, err)
			}
			return admin, nil
		})
	if err != nil {
		return nil, err
	}
	return sharedClusterAdmin{ClusterAdminInterface: admin, gen: gen}, nil
}

// getSharedClient returns the cached sarama.Client for the active cluster, for
// metadata and offset lookups. Like getClusterAdmin it is created on first use,
// safe for concurrent use, and must not be closed by callers. Consumers and
// producers that need their own config keep creating their own client.
//
// Metadata.Full is off: the client only tracks the topics it is asked about,
// so creating it does not fetch metadata for every topic in the cluster. The
// admin keeps Full on, because sarama's Controller lookup refreshes metadata
// and fails with ErrNoTopicsToUpdateMetadata when no topics are known.
func getSharedClient() (sarama.Client, error) {
	client, _, err := acquireShared(&sharedClients.client,
		fmt.Errorf("no Kafka cluster configured"),
		func(factory KafkaClientFactory, cluster *config.Cluster) (sarama.Client, error) {
			cfg, err := getConfig()
			if err != nil {
				return nil, err
			}
			cfg.Metadata.Full = false
			client, err := factory.CreateClient(cluster.Brokers, cfg)
			if err != nil {
				return nil, fmt.Errorf("unable to get client: %w", err)
			}
			if client == nil {
				return nil, fmt.Errorf("unable to get client: factory returned no client")
			}
			return client, nil
		})
	if err != nil {
		return nil, err
	}
	return sharedClient{client}, nil
}

// resetSharedClients forgets the cached admin and client (and any dial in
// flight), so the next call connects to the (new) active cluster. SetContext
// and Reload call it on the Bubble Tea Update goroutine, so it only swaps
// pointers under the lock and closes the old handles in the background.
func resetSharedClients() {
	s := &sharedClients
	s.mu.Lock()
	s.epoch++
	admin, adminLive := s.admin.take()
	client, clientLive := s.client.take()
	s.mu.Unlock()

	if adminLive {
		closeShared(admin)
	}
	if clientLive {
		closeShared(client)
	}
}

func getClient() (client sarama.Client, e error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}
	client, err = sarama.NewClient(currentCluster.Brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("Unable to get client: %v\n", err)
	}
	return client, nil
}

func getClientFromConfig(config *sarama.Config) (sarama.Client, error) {
	client, err := sarama.NewClient(currentCluster.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("Unable to get client: %v\n", err)
	}
	return client, nil
}

func getSchemaCache() (cache *avro.SchemaCache, er error) {
	if currentCluster == nil || currentCluster.SchemaRegistryURL == "" {
		return nil, nil
	}
	var username, password string
	if creds := currentCluster.SchemaRegistryCredentials; creds != nil {
		username = creds.Username
		password = creds.Password
	}
	cache, err := avro.NewSchemaCache(currentCluster.SchemaRegistryURL, username, password)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

// cachedSchemaCache is a process-lifetime cache of the schema registry client.
// It is invalidated when SetContext switches the active cluster.
var cachedSchemaCache *avro.SchemaCache

// getOrInitSchemaCache returns the cached SchemaCache, initialising it on first
// call. Returns nil (not an error) when no schema registry is configured.
func getOrInitSchemaCache() (*avro.SchemaCache, error) {
	if cachedSchemaCache != nil {
		return cachedSchemaCache, nil
	}
	sc, err := getSchemaCache()
	if err != nil {
		return nil, err
	}
	cachedSchemaCache = sc
	return sc, nil
}

// extractRecordName extracts the record name from an Avro schema JSON
func extractRecordName(schemaJSON string) string {
	// Simple JSON parsing to extract the "name" field
	// This is a basic implementation - could be improved with proper JSON parsing
	var schemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaMap); err != nil {
		return "Unknown"
	}

	if name, ok := schemaMap["name"].(string); ok {
		return name
	}

	return "Unknown"
}

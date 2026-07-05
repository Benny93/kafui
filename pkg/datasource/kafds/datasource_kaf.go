package kafds

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/serde"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/ui/shared"
	"github.com/IBM/sarama"
	"github.com/birdayz/kaf/pkg/avro"
	"github.com/birdayz/kaf/pkg/config"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
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
// metadata request. This is faster than GetTopics() because it skips the full
// per-partition replica assignment that ListTopics() returns.
func (kp KafkaDataSourceKaf) GetTopicNames() ([]string, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	if err := client.RefreshMetadata(); err != nil {
		return nil, fmt.Errorf("failed to refresh metadata: %w", err)
	}
	return client.Topics()
}

// GetTopics retrieves a list of Kafka topics
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
// (newestOffset - oldestOffset) across all partitions. A single Sarama client is reused
// for all topics to avoid repeated connection overhead. Topics that fail individually are
// skipped so a partial result is always returned.
func (kp KafkaDataSourceKaf) GetTopicMessageCounts(topics map[string]int32) (map[string]int64, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	counts := make(map[string]int64, len(topics))
	for name, numPartitions := range topics {
		var total int64
		for i := int32(0); i < numPartitions; i++ {
			offs, err := getOffsets(client, name, i)
			if err != nil {
				continue
			}
			total += offs.newest - offs.oldest
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

	// Invalidate caches (mirror SetContext).
	cachedSchemaCache = nil
	invalidateSerdeRegistry()
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
			cachedSchemaCache = nil // invalidate schema cache on cluster switch
			invalidateSerdeRegistry()
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
		if cluster.SASL.Mechanism != "OAUTHBEARER" {
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
					shared.Log.Error("failed to read TLS CA file", "file", cluster.TLS.Cafile, "err", err)
					os.Exit(1)
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

		}
	}
	return saramaConfig, nil
}

var (
	// outWriter and errWriter point to a discard writer during TUI operation.
	// InitTUIWriters() must be called before tea.NewProgram to prevent any
	// datasource output from corrupting Bubble Tea's alt-screen rendering.
	outWriter    io.Writer = os.Stdout
	errWriter    io.Writer = os.Stderr
	inReader     io.Reader = os.Stdin
	colorableOut io.Writer = colorable.NewColorableStdout()
)

// InitTUIWriters redirects outWriter and errWriter to the structured logger
// and silences the sarama Kafka client logger so nothing corrupts the TUI.
// Call this once before starting tea.NewProgram.
func InitTUIWriters() {
	w := shared.NewSlogWriter(shared.Log)
	outWriter = w
	errWriter = w
	colorableOut = w
	// Always route sarama logs to file — it is very chatty on reconnects.
	sarama.Logger = log.New(w, "[sarama] ", 0)
}

// Will be replaced by GitHub action and by goreleaser
// see https://goreleaser.com/customization/build/
var commit string = "HEAD"
var version string = "latest"

var rootCmd = &cobra.Command{
	Use:     "kaf",
	Short:   "Kafka Command Line utility for cluster management",
	Version: fmt.Sprintf("%s (%s)", version, commit),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		outWriter = cmd.OutOrStdout()
		errWriter = cmd.ErrOrStderr()
		inReader = cmd.InOrStdin()

		if outWriter != os.Stdout {
			colorableOut = outWriter
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		shared.Log.Error("root command failed", "err", err)
		os.Exit(1)
	}
}

var cfg config.Config
var currentCluster *config.Cluster

var (
	brokersFlag       []string
	schemaRegistryURL string
	protoFiles        []string
	protoExclude      []string
	decodeMsgPack     bool
	verbose           bool
	clusterOverride   string
)

// SetOverrides applies CLI overrides before Init/onInit runs. Empty/nil values
// leave the corresponding config value untouched. Kafui's own CLI calls this;
// the embedded rootCmd below is never executed.
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

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kaf/config)")
	rootCmd.PersistentFlags().StringSliceVarP(&brokersFlag, "brokers", "b", nil, "Comma separated list of broker ip:port pairs")
	rootCmd.PersistentFlags().StringVar(&schemaRegistryURL, "schema-registry", "", "URL to a Confluent schema registry. Used for attempting to decode Avro-encoded messages")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Whether to turn on sarama logging")
	rootCmd.PersistentFlags().StringVarP(&clusterOverride, "cluster", "c", "", "set a temporary current cluster")
	cobra.OnInitialize(onInit)
}

/*
var setupProtoDescriptorRegistry = func(cmd *cobra.Command, args []string) {
	if protoType != "" {
		r, err := proto.NewDescriptorRegistry(protoFiles, protoExclude)
		if err != nil {
			errorExit("Failed to load protobuf files: %v\n", err)
		}
		reg = r
	}
}*/

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
// default ~/.kaf/config) and sets the active cluster. This is identical to the
// setup that the main CLI performs via cobra.OnInitialize; use it in examples
// and standalone programs that do not go through the Cobra entry point.
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

func getClusterAdmin() (admin ClusterAdminInterface, e error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kafka config: %v", err)
	}

	if currentCluster == nil {
		return nil, fmt.Errorf("no Kafka cluster configured. Please check your configuration or ensure Kafka is running")
	}

	clusterAdmin, err := kafkaClientFactory.CreateClusterAdmin(currentCluster.Brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to Kafka cluster at %v: %v\nPlease ensure Kafka is running and accessible", currentCluster.Brokers, err)
	}

	return clusterAdmin, nil
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

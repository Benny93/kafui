package kafds

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/Benny93/kafui/pkg/api"
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

func (kp KafkaDataSourceKaf) SetContext(contextName string) error {
	cfg, err := kp.configManager.ReadConfig(cfgFile)
	if err != nil {
		return err
	}

	// Iterate through clusters in the config
	for _, cluster := range cfg.Clusters {
		// Check if the cluster name matches the contextName
		if cluster.Name == contextName {
			currentCluster = cluster
			err := cfg.SetCurrentCluster(currentCluster.Name)
			if err != nil {
				return err
			}
			return nil
		}
	}

	// If no matching cluster is found, return an error
	return fmt.Errorf("cluster with name '%s' not found", contextName)

}

func (kp KafkaDataSourceKaf) GetConsumerGroups() ([]api.ConsumerGroup, error) {
	admin, err := getClusterAdmin()
	if err != nil {
		return nil, err
	}

	groups, err := admin.ListConsumerGroups()
	if err != nil {
		return nil, err
	}

	groupList := make([]string, 0, len(groups))
	for grp := range groups {
		groupList = append(groupList, grp)
	}

	sort.Slice(groupList, func(i int, j int) bool {
		return groupList[i] < groupList[j]
	})

	groupDescs, err := admin.DescribeConsumerGroups(groupList)
	if err != nil {
		return nil, fmt.Errorf("Unable to describe consumer groups: %v\n", err)
	}

	finalGroups := make([]api.ConsumerGroup, 0, len(groupDescs))
	for _, detail := range groupDescs {
		state := detail.State
		consumers := len(detail.Members)
		tmpGroup := api.ConsumerGroup{
			Name:      detail.GroupId,
			State:     state,
			Consumers: consumers,
		}
		finalGroups = append(finalGroups, tmpGroup)

	}

	return finalGroups, nil
}

func (kp KafkaDataSourceKaf) ConsumeTopic(ctx context.Context, topicName string, flags api.ConsumeFlags, handleMessage api.MessageHandlerFunc, onError func(err any)) error {

	admin, err := getClusterAdmin()
	if err != nil {
		return err
	}
	topicDetails, _ := admin.ListTopics()

	keys := make([]string, 0, len(topicDetails))
	for key := range topicDetails {
		keys = append(keys, key)
	}

	DoConsume(ctx, topicName, flags, handleMessage, onError)

	//cgs := []string{"message1", "message2", "message3"} // Example
	return nil
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
					fmt.Println(err)
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
	outWriter io.Writer = os.Stdout
	errWriter io.Writer = os.Stderr
	inReader  io.Reader = os.Stdin

	colorableOut io.Writer = colorable.NewColorableStdout()
)

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
		fmt.Println(err)
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

func onInit() {
	var err error
	cfg, err = config.ReadConfig(cfgFile)
	if err != nil {
		// Instead of panicking, create a default config
		fmt.Fprintf(errWriter, "Warning: Could not read config file (%v). Using default configuration.\n", err)
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

	if verbose {
		sarama.Logger = log.New(errWriter, "[sarama] ", log.Lshortfile|log.LstdFlags)
	}
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
	if currentCluster.SchemaRegistryURL == "" {
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

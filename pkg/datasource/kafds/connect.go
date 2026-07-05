package kafds

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/appconfig"
	"github.com/Benny93/kafui/pkg/ui/shared"
)

// loadConnectClusters resolves the Connect clusters configured for the given
// Kafka context name from the kafui overlay config. It is a package variable so
// tests can substitute an in-memory config without touching disk. The kaf config
// file (~/.kaf/config) is never read or written here.
var loadConnectClusters = func(context string) []appconfig.ConnectCluster {
	cfg, err := appconfig.Load(appconfig.DefaultPath())
	if err != nil {
		shared.Log.Warn("loading kafui config for connect", "err", err)
		return nil
	}
	return cfg.Clusters[context].Connect
}

// connectError carries a Connect REST non-2xx response. The Connect API returns
// {"error_code": ..., "message": ...} bodies; ErrorCode is 0 when unparseable.
type connectError struct {
	StatusCode int
	ErrorCode  int
	Message    string
}

func (e *connectError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("connect returned %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("connect returned %d", e.StatusCode)
}

// connectClient is a small HTTP client for one Connect cluster supporting all
// verbs, basic auth, optional TLS, and typed error mapping. Construct via
// newConnectClient.
type connectClient struct {
	cluster  appconfig.ConnectCluster
	baseURL  string
	username string
	password string
	http     *http.Client
}

// newConnectClient builds a client for a single configured Connect cluster,
// wiring basic auth and TLS from its config fields.
func newConnectClient(cc appconfig.ConnectCluster) (*connectClient, error) {
	base := strings.TrimRight(strings.TrimSpace(cc.Address), "/")
	if base == "" {
		return nil, fmt.Errorf("connect cluster %q has no address", cc.Name)
	}

	transport, err := connectTLSTransport(cc)
	if err != nil {
		return nil, err
	}

	return &connectClient{
		cluster:  cc,
		baseURL:  base,
		username: cc.Username,
		password: cc.Password,
		http: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
	}, nil
}

// connectTLSTransport builds an http.RoundTripper honoring the cluster's TLS
// settings. It returns nil (use http defaults) when no TLS fields are set.
func connectTLSTransport(cc appconfig.ConnectCluster) (http.RoundTripper, error) {
	if cc.TLSCAPath == "" && cc.TLSCertPath == "" && cc.TLSKeyPath == "" {
		return nil, nil
	}
	tlsCfg := &tls.Config{}
	if cc.TLSCAPath != "" {
		pem, err := os.ReadFile(cc.TLSCAPath)
		if err != nil {
			return nil, fmt.Errorf("reading connect CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("connect CA cert %q contains no valid certificates", cc.TLSCAPath)
		}
		tlsCfg.RootCAs = pool
	}
	if cc.TLSCertPath != "" && cc.TLSKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(cc.TLSCertPath, cc.TLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("loading connect client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return &http.Transport{TLSClientConfig: tlsCfg}, nil
}

// do executes a request against the Connect cluster. An HTTP response — even
// 4xx/5xx — is returned as a *connectError on non-2xx. On success the body is
// decoded into out when out is non-nil.
func (c *connectClient) do(method, path string, body, out interface{}) error {
	var reqBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding connect request: %w", err)
		}
		reqBody = b
	}

	var rdr io.Reader
	if reqBody != nil {
		rdr = bytes.NewReader(reqBody)
	}
	req, err := http.NewRequest(method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		ce := &connectError{StatusCode: resp.StatusCode}
		var parsed struct {
			ErrorCode int    `json:"error_code"`
			Message   string `json:"message"`
		}
		if json.Unmarshal(respBody, &parsed) == nil {
			ce.ErrorCode = parsed.ErrorCode
			ce.Message = parsed.Message
		}
		if ce.Message == "" {
			ce.Message = strings.TrimSpace(string(respBody))
		}
		return ce
	}
	if out != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

func (c *connectClient) doGet(path string, out interface{}) error {
	return c.do(http.MethodGet, path, nil, out)
}
func (c *connectClient) doPost(path string, body, out interface{}) error {
	return c.do(http.MethodPost, path, body, out)
}
func (c *connectClient) doPut(path string, body, out interface{}) error {
	return c.do(http.MethodPut, path, body, out)
}
func (c *connectClient) doDelete(path string, out interface{}) error {
	return c.do(http.MethodDelete, path, nil, out)
}

// mapConnectError converts a *connectError into a typed api error using the
// call's connect/connector context. Non-connect errors pass through unchanged.
func mapConnectError(err error, connect, connector string) error {
	if err == nil {
		return nil
	}
	var ce *connectError
	if !errors.As(err, &ce) {
		return err
	}
	switch {
	case ce.StatusCode == http.StatusNotFound:
		return api.ConnectorNotFoundError{Connector: connector, Connect: connect, Cause: ce}
	case ce.StatusCode == http.StatusConflict:
		// 409 on create means the connector already exists; elsewhere it is a
		// rebalance in progress. Surface the message verbatim for the latter.
		if strings.Contains(strings.ToLower(ce.Message), "already exists") {
			return api.ConnectorAlreadyExistsError{Connector: connector, Connect: connect, Cause: ce}
		}
		return ce
	default:
		return ce
	}
}

// connectClient resolves the named Connect cluster from the active context's
// config and returns an HTTP client for it. Unknown name yields a
// ConnectClusterNotFoundError.
func (kp KafkaDataSourceKaf) connectClient(name string) (*connectClient, error) {
	ctx := kp.GetContext()
	for _, cc := range loadConnectClusters(ctx) {
		if cc.Name == name {
			return newConnectClient(cc)
		}
	}
	return nil, api.ConnectClusterNotFoundError{Connect: name, Cluster: ctx}
}

// consumerGroupFor derives a sink connector's consumer group from the cluster's
// configured pattern (default "connect-<connector>"). A "<connector>" token in
// the pattern is substituted with the connector name.
func consumerGroupFor(cc appconfig.ConnectCluster, connector string) string {
	pattern := cc.ConsumerNamePattern
	if pattern == "" {
		return "connect-" + connector
	}
	return strings.ReplaceAll(pattern, "<connector>", connector)
}

// --- read operations (KC-5) ---

// rootInfo is the GET / response of a Connect worker.
type connectRootInfo struct {
	Version        string `json:"version"`
	Commit         string `json:"commit"`
	KafkaClusterID string `json:"kafka_cluster_id"`
}

// connectorStatusResp is the /status response for one connector.
type connectorStatusResp struct {
	Name      string `json:"name"`
	Connector struct {
		State    string `json:"state"`
		WorkerID string `json:"worker_id"`
		Trace    string `json:"trace"`
	} `json:"connector"`
	Tasks []struct {
		ID       int    `json:"id"`
		State    string `json:"state"`
		WorkerID string `json:"worker_id"`
		Trace    string `json:"trace"`
	} `json:"tasks"`
	Type string `json:"type"`
}

func (kp KafkaDataSourceKaf) GetConnectClusters(withStats bool) ([]api.ConnectCluster, error) {
	configured := loadConnectClusters(kp.GetContext())
	out := make([]api.ConnectCluster, 0, len(configured))
	for _, cc := range configured {
		cluster := api.ConnectCluster{Name: cc.Name, Address: cc.Address}
		client, err := newConnectClient(cc)
		if err != nil {
			out = append(out, cluster) // unreachable: listed, not reachable
			continue
		}
		var root connectRootInfo
		if err := client.doGet("/", &root); err != nil {
			shared.Log.Warn("connect cluster root unreachable", "connect", cc.Name, "err", err)
			out = append(out, cluster)
			continue
		}
		cluster.Reachable = true
		cluster.Version = root.Version
		cluster.Commit = root.Commit
		cluster.KafkaClusterID = root.KafkaClusterID

		if withStats {
			kp.fillConnectStats(client, &cluster)
		}
		out = append(out, cluster)
	}
	return out, nil
}

// fillConnectStats computes connector/task counts for one cluster using the
// expanded connectors endpoint. Failures leave counts at zero.
func (kp KafkaDataSourceKaf) fillConnectStats(client *connectClient, cluster *api.ConnectCluster) {
	var expanded map[string]struct {
		Status *connectorStatusResp `json:"status"`
	}
	if err := client.doGet("/connectors?expand=status", &expanded); err != nil {
		shared.Log.Warn("connect stats unavailable", "connect", cluster.Name, "err", err)
		return
	}
	for _, entry := range expanded {
		cluster.ConnectorCount++
		if entry.Status == nil {
			continue
		}
		if strings.EqualFold(entry.Status.Connector.State, api.ConnectorStateFailed) {
			cluster.FailedConnectorCount++
		}
		for _, t := range entry.Status.Tasks {
			cluster.TaskCount++
			if strings.EqualFold(t.State, api.ConnectorStateFailed) {
				cluster.FailedTaskCount++
			}
		}
	}
}

func (kp KafkaDataSourceKaf) GetConnectorNames(connect string) ([]string, error) {
	client, err := kp.connectClient(connect)
	if err != nil {
		return nil, err
	}
	var names []string
	if err := client.doGet("/connectors", &names); err != nil {
		return nil, fmt.Errorf("listing connectors on %q: %w", connect, mapConnectError(err, connect, ""))
	}
	sort.Strings(names)
	return names, nil
}

func (kp KafkaDataSourceKaf) GetConnectors() ([]api.Connector, error) {
	configured := loadConnectClusters(kp.GetContext())

	var (
		mu     sync.Mutex
		result []api.Connector
		wg     sync.WaitGroup
	)
	const workers = 20
	sem := make(chan struct{}, workers)

	for _, cc := range configured {
		cc := cc
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			conns, err := kp.connectorsForCluster(cc)
			if err != nil {
				shared.Log.Warn("connect cluster omitted from aggregation", "connect", cc.Name, "err", err)
				return
			}
			mu.Lock()
			result = append(result, conns...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].ConnectCluster < result[j].ConnectCluster
	})
	return result, nil
}

// connectorsForCluster fetches all connectors of a single cluster with their
// info and status expanded. An unreachable cluster returns an error so the
// aggregator can drop it.
func (kp KafkaDataSourceKaf) connectorsForCluster(cc appconfig.ConnectCluster) ([]api.Connector, error) {
	client, err := newConnectClient(cc)
	if err != nil {
		return nil, err
	}
	var expanded map[string]struct {
		Info struct {
			Config map[string]string `json:"config"`
			Type   string            `json:"type"`
		} `json:"info"`
		Status *connectorStatusResp `json:"status"`
	}
	if err := client.doGet("/connectors?expand=info&expand=status", &expanded); err != nil {
		return nil, mapConnectError(err, cc.Name, "")
	}

	out := make([]api.Connector, 0, len(expanded))
	for name, entry := range expanded {
		conn := api.Connector{
			ConnectCluster: cc.Name,
			Name:           name,
			Class:          entry.Info.Config["connector.class"],
			Type:           connectorTypeOf(entry.Info.Type),
		}
		if entry.Status != nil {
			conn.State = entry.Status.Connector.State
			conn.WorkerID = entry.Status.Connector.WorkerID
			conn.Trace = entry.Status.Connector.Trace
			conn.TaskCount = len(entry.Status.Tasks)
			for _, t := range entry.Status.Tasks {
				if strings.EqualFold(t.State, api.ConnectorStateFailed) {
					conn.FailedTaskCount++
				}
			}
			if conn.Type == api.ConnectorTypeUnknown {
				conn.Type = connectorTypeOf(entry.Status.Type)
			}
		}
		conn.Topics = kp.connectorTopics(client, name)
		if conn.Type == api.ConnectorTypeSink {
			conn.ConsumerGroup = consumerGroupFor(cc, name)
		}
		out = append(out, conn)
	}
	return out, nil
}

func connectorTypeOf(t string) api.ConnectorType {
	switch strings.ToLower(t) {
	case "source":
		return api.ConnectorTypeSource
	case "sink":
		return api.ConnectorTypeSink
	default:
		return api.ConnectorTypeUnknown
	}
}

// connectorTopics fetches a connector's topics, tolerating 404/older Connect
// versions with an empty list.
func (kp KafkaDataSourceKaf) connectorTopics(client *connectClient, name string) []string {
	var resp map[string]struct {
		Topics []string `json:"topics"`
	}
	if err := client.doGet("/connectors/"+name+"/topics", &resp); err != nil {
		return nil
	}
	if entry, ok := resp[name]; ok {
		sort.Strings(entry.Topics)
		return entry.Topics
	}
	return nil
}

func (kp KafkaDataSourceKaf) GetConnectorDetails(connect, name string) (api.ConnectorDetails, error) {
	client, err := kp.connectClient(connect)
	if err != nil {
		return api.ConnectorDetails{}, err
	}

	var info struct {
		Name   string            `json:"name"`
		Config map[string]string `json:"config"`
		Type   string            `json:"type"`
	}
	if err := client.doGet("/connectors/"+name, &info); err != nil {
		return api.ConnectorDetails{}, mapConnectError(err, connect, name)
	}

	details := api.ConnectorDetails{
		ConnectCluster: connect,
		Name:           name,
		Class:          info.Config["connector.class"],
		Type:           connectorTypeOf(info.Type),
		Config:         api.MaskConnectorConfig(info.Config),
		State:          api.ConnectorStateUnassigned,
	}

	var status connectorStatusResp
	if err := client.doGet("/connectors/"+name+"/status", &status); err != nil {
		// Missing status: report UNASSIGNED with an empty task list, not an error.
		shared.Log.Warn("connector status unavailable", "connect", connect, "connector", name, "err", err)
	} else {
		details.State = status.Connector.State
		details.WorkerID = status.Connector.WorkerID
		details.Trace = status.Connector.Trace
		for _, t := range status.Tasks {
			details.Tasks = append(details.Tasks, api.ConnectorTask{
				ID:       t.ID,
				WorkerID: t.WorkerID,
				State:    t.State,
				Trace:    t.Trace,
			})
		}
		if details.Type == api.ConnectorTypeUnknown {
			details.Type = connectorTypeOf(status.Type)
		}
	}
	sort.Slice(details.Tasks, func(i, j int) bool { return details.Tasks[i].ID < details.Tasks[j].ID })

	details.Topics = kp.connectorTopics(client, name)
	if details.Type == api.ConnectorTypeSink {
		if cc, ok := kp.connectClusterConfig(connect); ok {
			details.ConsumerGroup = consumerGroupFor(cc, name)
		}
	}
	return details, nil
}

// connectClusterConfig returns the raw config entry for a named Connect cluster.
func (kp KafkaDataSourceKaf) connectClusterConfig(name string) (appconfig.ConnectCluster, bool) {
	for _, cc := range loadConnectClusters(kp.GetContext()) {
		if cc.Name == name {
			return cc, true
		}
	}
	return appconfig.ConnectCluster{}, false
}

// --- write operations (KC-7) ---

func (kp KafkaDataSourceKaf) CreateConnector(connect, name string, config map[string]string) (api.Connector, error) {
	client, err := kp.connectClient(connect)
	if err != nil {
		return api.Connector{}, err
	}
	body := map[string]interface{}{"name": name, "config": config}
	var resp struct {
		Name   string            `json:"name"`
		Config map[string]string `json:"config"`
		Type   string            `json:"type"`
	}
	if err := client.doPost("/connectors", body, &resp); err != nil {
		return api.Connector{}, mapConnectError(err, connect, name)
	}
	return api.Connector{
		ConnectCluster: connect,
		Name:           name,
		Class:          config["connector.class"],
		Type:           connectorTypeOf(resp.Type),
	}, nil
}

func (kp KafkaDataSourceKaf) UpdateConnectorConfig(connect, name string, config map[string]string) (api.Connector, error) {
	client, err := kp.connectClient(connect)
	if err != nil {
		return api.Connector{}, err
	}
	var resp struct {
		Config map[string]string `json:"config"`
		Type   string            `json:"type"`
	}
	if err := client.doPut("/connectors/"+name+"/config", config, &resp); err != nil {
		return api.Connector{}, mapConnectError(err, connect, name)
	}
	return api.Connector{
		ConnectCluster: connect,
		Name:           name,
		Class:          config["connector.class"],
		Type:           connectorTypeOf(resp.Type),
	}, nil
}

func (kp KafkaDataSourceKaf) DeleteConnector(connect, name string) error {
	client, err := kp.connectClient(connect)
	if err != nil {
		return err
	}
	if err := client.doDelete("/connectors/"+name, nil); err != nil {
		return mapConnectError(err, connect, name)
	}
	return nil
}

func (kp KafkaDataSourceKaf) PauseConnector(connect, name string) error {
	return kp.lifecyclePut(connect, name, "/pause")
}

func (kp KafkaDataSourceKaf) ResumeConnector(connect, name string) error {
	return kp.lifecyclePut(connect, name, "/resume")
}

func (kp KafkaDataSourceKaf) StopConnector(connect, name string) error {
	return kp.lifecyclePut(connect, name, "/stop")
}

func (kp KafkaDataSourceKaf) lifecyclePut(connect, name, action string) error {
	client, err := kp.connectClient(connect)
	if err != nil {
		return err
	}
	if err := client.doPut("/connectors/"+name+action, nil, nil); err != nil {
		return mapConnectError(err, connect, name)
	}
	return nil
}

func (kp KafkaDataSourceKaf) RestartConnector(connect, name string) error {
	client, err := kp.connectClient(connect)
	if err != nil {
		return err
	}
	if err := client.doPost("/connectors/"+name+"/restart", nil, nil); err != nil {
		return mapConnectError(err, connect, name)
	}
	return nil
}

func (kp KafkaDataSourceKaf) RestartConnectorTask(connect, name string, taskID int) error {
	client, err := kp.connectClient(connect)
	if err != nil {
		return err
	}
	path := "/connectors/" + name + "/tasks/" + strconv.Itoa(taskID) + "/restart"
	if err := client.doPost(path, nil, nil); err != nil {
		return mapConnectError(err, connect, name)
	}
	return nil
}

func (kp KafkaDataSourceKaf) ResetConnectorOffsets(connect, name string) error {
	client, err := kp.connectClient(connect)
	if err != nil {
		return err
	}
	// Guard: the connector must be STOPPED. Check state first without mutating.
	var status connectorStatusResp
	if err := client.doGet("/connectors/"+name+"/status", &status); err != nil {
		return mapConnectError(err, connect, name)
	}
	if !strings.EqualFold(status.Connector.State, api.ConnectorStateStopped) {
		return api.ConnectorNotStoppedError{Connector: name, Connect: connect, State: status.Connector.State}
	}
	if err := client.doDelete("/connectors/"+name+"/offsets", nil); err != nil {
		return mapConnectError(err, connect, name)
	}
	return nil
}

// --- plugins & validation (KC-8) ---

func (kp KafkaDataSourceKaf) GetConnectorPlugins(connect string) ([]api.ConnectorPlugin, error) {
	client, err := kp.connectClient(connect)
	if err != nil {
		return nil, err
	}
	var plugins []struct {
		Class   string `json:"class"`
		Type    string `json:"type"`
		Version string `json:"version"`
	}
	if err := client.doGet("/connector-plugins", &plugins); err != nil {
		return nil, mapConnectError(err, connect, "")
	}
	out := make([]api.ConnectorPlugin, len(plugins))
	for i, p := range plugins {
		out[i] = api.ConnectorPlugin{Class: p.Class, Type: p.Type, Version: p.Version}
	}
	return out, nil
}

func (kp KafkaDataSourceKaf) ValidateConnectorConfig(connect, pluginClass string, config map[string]string) (api.ConnectorValidationResult, error) {
	client, err := kp.connectClient(connect)
	if err != nil {
		return api.ConnectorValidationResult{}, err
	}
	var resp struct {
		Name       string   `json:"name"`
		ErrorCount int      `json:"error_count"`
		Groups     []string `json:"groups"`
		Configs    []struct {
			Value struct {
				Name              string   `json:"name"`
				Value             string   `json:"value"`
				Errors            []string `json:"errors"`
				RecommendedValues []string `json:"recommended_values"`
				Visible           bool     `json:"visible"`
			} `json:"value"`
		} `json:"configs"`
	}
	path := "/connector-plugins/" + pluginClass + "/config/validate"
	if err := client.doPut(path, config, &resp); err != nil {
		return api.ConnectorValidationResult{}, mapConnectError(err, connect, "")
	}
	result := api.ConnectorValidationResult{
		Name:       resp.Name,
		ErrorCount: resp.ErrorCount,
		Groups:     resp.Groups,
	}
	for _, c := range resp.Configs {
		result.Configs = append(result.Configs, api.ConnectorConfigKeyValidation{
			Name:              c.Value.Name,
			Value:             c.Value.Value,
			Errors:            c.Value.Errors,
			RecommendedValues: c.Value.RecommendedValues,
			Visible:           c.Value.Visible,
		})
	}
	return result, nil
}

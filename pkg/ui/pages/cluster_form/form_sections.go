package cluster_form

import (
	"fmt"
	"os"
	"strings"

	"github.com/Benny93/kafui/pkg/appconfig"
	formpkg "github.com/Benny93/kafui/pkg/ui/components/form"
)

// Field names shared by the form builder and the candidate mapping.
const (
	fName             = "name"
	fReadOnly         = "readOnly"
	fBrokers          = "brokers"
	fKafkaVersion     = "kafkaVersion"
	fSecurityProtocol = "securityProtocol"
	fSaslMechanism    = "saslMechanism"
	fSaslUsername     = "saslUsername"
	fSaslPassword     = "saslPassword"
	fSaslClientID     = "saslClientID"
	fSaslClientSecret = "saslClientSecret"
	fSaslTokenURL     = "saslTokenURL"
	fTLSCa            = "tlsCaPath"
	fTLSCert          = "tlsCertPath"
	fTLSKey           = "tlsKeyPath"
	fTLSInsecure      = "tlsInsecure"
	fSchemaURL        = "schemaRegistryUrl"
	fSchemaUser       = "schemaRegistryUser"
	fSchemaPassword   = "schemaRegistryPassword"
	fConnectName      = "connectName"
	fConnectAddress   = "connectAddress"
	fKsqlURL          = "ksqlUrl"
	fMetricsURL       = "metricsUrl"
)

const noneOption = "(none)"

var securityProtocolOptions = []string{"PLAINTEXT", "SSL", "SASL_PLAINTEXT", "SASL_SSL"}

// saslMechanismOptions is limited to what kafds supports.
var saslMechanismOptions = []string{noneOption, "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512", "OAUTHBEARER"}

// fileExistsValidator rejects a non-empty path that does not point at a readable file.
func fileExistsValidator(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	if _, err := os.Stat(v); err != nil {
		return fmt.Errorf("file not found: %s", v)
	}
	return nil
}

// buildFields returns the wizard field set, prefilled from ext when editing.
func buildFields(name string, ext appconfig.ClusterExtension) []formpkg.Field {
	sasl := ext.SASL
	if sasl == nil {
		sasl = &appconfig.SASLConfig{}
	}
	tls := ext.TLS
	if tls == nil {
		tls = &appconfig.TLSConfig{}
	}
	secProto := ext.SecurityProtocol
	if secProto == "" {
		secProto = "PLAINTEXT"
	}
	mech := sasl.Mechanism
	if mech == "" {
		mech = noneOption
	}
	connName, connAddr := "", ""
	if len(ext.Connect) > 0 {
		connName = ext.Connect[0].Name
		connAddr = ext.Connect[0].Address
	}
	ksqlURL := ""
	if ext.Ksql != nil {
		ksqlURL = ext.Ksql.URL
	}

	return []formpkg.Field{
		// --- Cluster basics ---
		{Name: fName, Label: "Cluster Name", Type: formpkg.Text, Required: true, Default: name},
		{Name: fReadOnly, Label: "Read Only", Type: formpkg.Bool, Default: boolStr(ext.ReadOnly)},
		{Name: fBrokers, Label: "Brokers (comma-separated host:port)", Type: formpkg.Text, Required: true, Default: strings.Join(ext.Brokers, ",")},
		{Name: fKafkaVersion, Label: "Kafka Version (optional)", Type: formpkg.Text, Default: ext.KafkaVersion},
		{Name: fTLSCa, Label: "TLS CA File Path", Type: formpkg.Text, Default: tls.CAPath, Validator: fileExistsValidator},
		{Name: fTLSCert, Label: "TLS Client Cert File Path", Type: formpkg.Text, Default: tls.CertPath, Validator: fileExistsValidator},
		{Name: fTLSKey, Label: "TLS Client Key File Path", Type: formpkg.Text, Default: tls.KeyPath, Validator: fileExistsValidator},
		{Name: fTLSInsecure, Label: "TLS Skip Verify", Type: formpkg.Bool, Default: boolStr(tls.Insecure)},
		// --- Broker authentication ---
		{Name: fSecurityProtocol, Label: "Security Protocol", Type: formpkg.Select, Options: securityProtocolOptions, Default: secProto},
		{Name: fSaslMechanism, Label: "SASL Mechanism", Type: formpkg.Select, Options: saslMechanismOptions, Default: mech},
		{Name: fSaslUsername, Label: "SASL Username", Type: formpkg.Text, Default: sasl.Username},
		{Name: fSaslPassword, Label: "SASL Password", Type: formpkg.Text, Default: sasl.Password},
		{Name: fSaslClientID, Label: "SASL Client ID (OAUTHBEARER)", Type: formpkg.Text, Default: sasl.ClientID},
		{Name: fSaslClientSecret, Label: "SASL Client Secret (OAUTHBEARER)", Type: formpkg.Text, Default: sasl.ClientSecret},
		{Name: fSaslTokenURL, Label: "SASL Token URL (OAUTHBEARER)", Type: formpkg.Text, Default: sasl.TokenURL},
		// --- Schema registry ---
		{Name: fSchemaURL, Label: "Schema Registry URL", Type: formpkg.Text, Default: ext.SchemaRegistryURL},
		{Name: fSchemaUser, Label: "Schema Registry Username", Type: formpkg.Text, Default: ext.SchemaRegistryUsername},
		{Name: fSchemaPassword, Label: "Schema Registry Password", Type: formpkg.Text, Default: ext.SchemaRegistryPassword},
		// --- Extension stubs ---
		{Name: fConnectName, Label: "Connect Name (optional)", Type: formpkg.Text, Default: connName},
		{Name: fConnectAddress, Label: "Connect URL (optional)", Type: formpkg.Text, Default: connAddr},
		{Name: fKsqlURL, Label: "ksqlDB URL (optional)", Type: formpkg.Text, Default: ksqlURL},
		{Name: fMetricsURL, Label: "Metrics URL (optional)", Type: formpkg.Text, Default: metricsPrefill(ext.Metrics)},
	}
}

// candidateFromValues maps submitted form values to a cluster name and a
// fully-kafui-defined ClusterExtension. The selected SASL mechanism decides
// which auth fields are carried into the generated SASLConfig.
func candidateFromValues(v map[string]string) (string, appconfig.ClusterExtension, error) {
	name := strings.TrimSpace(v[fName])
	ext := appconfig.ClusterExtension{ReadOnly: v[fReadOnly] == "true"}

	for _, b := range strings.Split(v[fBrokers], ",") {
		if b = strings.TrimSpace(b); b != "" {
			ext.Brokers = append(ext.Brokers, b)
		}
	}
	ext.KafkaVersion = strings.TrimSpace(v[fKafkaVersion])

	if proto := v[fSecurityProtocol]; proto != "PLAINTEXT" {
		ext.SecurityProtocol = proto
	}

	if mech := v[fSaslMechanism]; mech != "" && mech != noneOption {
		s := &appconfig.SASLConfig{Mechanism: mech}
		switch mech {
		case "OAUTHBEARER":
			s.ClientID = v[fSaslClientID]
			s.ClientSecret = v[fSaslClientSecret]
			s.TokenURL = v[fSaslTokenURL]
		default: // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
			s.Username = v[fSaslUsername]
			s.Password = v[fSaslPassword]
		}
		ext.SASL = s
	}

	if v[fTLSCa] != "" || v[fTLSCert] != "" || v[fTLSKey] != "" || v[fTLSInsecure] == "true" {
		ext.TLS = &appconfig.TLSConfig{
			CAPath:   strings.TrimSpace(v[fTLSCa]),
			CertPath: strings.TrimSpace(v[fTLSCert]),
			KeyPath:  strings.TrimSpace(v[fTLSKey]),
			Insecure: v[fTLSInsecure] == "true",
		}
	}

	ext.SchemaRegistryURL = strings.TrimSpace(v[fSchemaURL])
	ext.SchemaRegistryUsername = v[fSchemaUser]
	ext.SchemaRegistryPassword = v[fSchemaPassword]

	if addr := strings.TrimSpace(v[fConnectAddress]); addr != "" {
		cn := strings.TrimSpace(v[fConnectName])
		if cn == "" {
			cn = "connect"
		}
		ext.Connect = []appconfig.ConnectCluster{{Name: cn, Address: addr}}
	}
	if url := strings.TrimSpace(v[fKsqlURL]); url != "" {
		ext.Ksql = &appconfig.KsqlEndpoint{URL: url}
	}
	if url := strings.TrimSpace(v[fMetricsURL]); url != "" {
		ext.Metrics = map[string]string{"url": url}
	}

	if name == "" {
		return "", ext, fmt.Errorf("cluster name is required")
	}
	return name, ext, nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func metricsPrefill(m map[string]string) string {
	if m == nil {
		return ""
	}
	return m["url"]
}

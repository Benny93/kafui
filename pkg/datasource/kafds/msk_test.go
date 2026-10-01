package kafds

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMSKTokenProvider_TokenSignsOnceAndCaches(t *testing.T) {
	calls := 0
	prov := &mskTokenProvider{
		gen: func(_ context.Context, region string) (string, int64, error) {
			calls++
			assert.Equal(t, "us-east-1", region)
			return fmt.Sprintf("token-%d", calls), time.Now().Add(time.Hour).UnixMilli(), nil
		},
		region: "us-east-1",
	}

	first, err := prov.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-1", first.Token)

	// Well before the replace buffer, the cached token is reused.
	second, err := prov.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-1", second.Token)
	assert.Equal(t, 1, calls, "token must be cached until near expiry")
}

func TestMSKTokenProvider_ReSignsAfterExpiryBuffer(t *testing.T) {
	calls := 0
	prov := &mskTokenProvider{
		gen: func(_ context.Context, _ string) (string, int64, error) {
			calls++
			// Expiry inside the refresh buffer forces re-signing on next call.
			return fmt.Sprintf("token-%d", calls), time.Now().Add(time.Second).UnixMilli(), nil
		},
		region: "eu-west-1",
	}

	_, err := prov.Token()
	require.NoError(t, err)
	tok, err := prov.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-2", tok.Token)
	assert.Equal(t, 2, calls)
}

func TestMSKTokenProvider_GenerationErrorPropagates(t *testing.T) {
	prov := &mskTokenProvider{
		gen: func(_ context.Context, _ string) (string, int64, error) {
			return "", 0, fmt.Errorf("no credentials")
		},
		region: "us-west-2",
	}

	_, err := prov.Token()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate AWS MSK IAM auth token")
	assert.Contains(t, err.Error(), "no credentials")
}

func TestMSKTokenProvider_ConstructorRequiresRegion(t *testing.T) {
	// Point the SDK at a synthetic shared config file so region resolution is
	// deterministic regardless of the developer's own ~/.aws state.
	cfgFile := t.TempDir() + "/aws-config"
	writeAWSConfig(t, cfgFile, "")
	t.Setenv("AWS_CONFIG_FILE", cfgFile)
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")

	_, err := newMSKTokenProvider()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires an AWS region")

	writeAWSConfig(t, cfgFile, "eu-central-1")
	prov, err := newMSKTokenProvider()
	require.NoError(t, err)
	assert.Equal(t, "eu-central-1", prov.region)
	assert.NotNil(t, prov.gen)
	var _ sarama.AccessTokenProvider = prov
}

func writeAWSConfig(t *testing.T, path, region string) {
	t.Helper()
	body := "[default]\n"
	if region != "" {
		body += "region = " + region + "\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
}

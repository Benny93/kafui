package kafds

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
)

// mskTokenProvider implements sarama.AccessTokenProvider for the AWS_MSK_IAM
// SASL mechanism (issue #3): AWS MSK authenticates via SigV4-presigned URLs
// that are carried in the OAUTHBEARER token field. The token is produced by the
// official aws-msk-iam-sasl-signer-go using the default AWS credential chain
// (env, shared config, IMDS, ...), so kafui needs no AK/SK configuration.
type mskTokenProvider struct {
	refreshMutex sync.Mutex
	// gen signs an auth token; injectable for tests. It mirrors the signer
	// package signature: token, expiry as unix milliseconds, error.
	gen func(ctx context.Context, region string) (string, int64, error)
	// region used for signing; resolved from the AWS config chain when the
	// provider is built for production use.
	region string
	// cached token state, following the same replace-before-expiry scheme as
	// the client-credentials tokenProvider.
	expiresAt    time.Time
	replaceAt    time.Time
	currentToken string
}

var _ sarama.AccessTokenProvider = &mskTokenProvider{}

// newMSKTokenProvider resolves the AWS region up front via the SDK default
// chain and returns a provider whose first Token() call signs the initial
// auth token. Region resolution honors AWS_REGION/AWS_DEFAULT_REGION, the
// shared config file, and instance metadata — no kafui-side field is needed.
func newMSKTokenProvider() (*mskTokenProvider, error) {
	cfg, err := awscfg.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load AWS config for AWS_MSK_IAM: %w", err)
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS_MSK_IAM requires an AWS region; set AWS_REGION or configure it in ~/.aws/config")
	}
	return &mskTokenProvider{
		gen: func(ctx context.Context, region string) (string, int64, error) {
			return signer.GenerateAuthToken(ctx, region)
		},
		region: cfg.Region,
	}, nil
}

// Token returns a valid MSK IAM auth token, re-signing shortly before expiry.
// sarama invokes it on every (re)authentication, so with the replace buffer a
// reconnect reuses the cached token unless it is about to expire.
func (p *mskTokenProvider) Token() (*sarama.AccessToken, error) {
	p.refreshMutex.Lock()
	defer p.refreshMutex.Unlock()
	if p.currentToken == "" || time.Now().After(p.replaceAt) {
		if err := p.refreshToken(context.Background()); err != nil {
			return nil, err
		}
	}
	return &sarama.AccessToken{Token: p.currentToken}, nil
}

func (p *mskTokenProvider) refreshToken(ctx context.Context) error {
	if p.currentToken != "" && time.Now().Before(p.replaceAt) {
		return nil // another caller refreshed while we waited for the lock
	}
	token, expiryMs, err := p.gen(ctx, p.region)
	if err != nil {
		return fmt.Errorf("failed to generate AWS MSK IAM auth token: %w", err)
	}
	p.expiresAt = time.UnixMilli(expiryMs)
	p.currentToken = token
	p.replaceAt = p.expiresAt.Add(-refreshBuffer)
	return nil
}

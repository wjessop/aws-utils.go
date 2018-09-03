package credentials

import (
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pkg/errors"
)

var (
	awsOperationRetryCount = 50
)

// The Credentials interface specifies the methods that all profiders must
// provide.
type Credentials interface {
	Get() (credentials.Value, error)
	Expire()
	IsExpired() bool
}

// FromEnvironment returns an AWS credentials-a-like from environment variables
func FromEnvironment() (Credentials, error) {
	creds := credentials.NewEnvCredentials()
	_, err := creds.Get()
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't get AWS credentials from environment")
	}

	return creds, nil
}

// FromProvider returns a Credentials object from a provider
func FromProvider(provider *VaultCredsProvider) Credentials {
	return credentials.NewCredentials(provider)
}

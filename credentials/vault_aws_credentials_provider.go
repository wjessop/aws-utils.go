package credentials

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pkg/errors"
	consul "github.com/wjessop/consul-utils.go/client"
)

// VaultCredsProvider is a custom AWS credentials provider for Vault
type VaultCredsProvider struct {
	vaultClient *vaultClient
	vaultKey    string
}

// NewVaultCredsProvider returns a configured VaultCredsProvider
func NewVaultCredsProvider(key string) (*VaultCredsProvider, error) {
	// Get the address of the Vault server
	consulClient, err := consul.NewClient(consulAddress)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating consul client")
	}

	addrs, _, err := consulClient.Service("vault", "active")
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't get vault address from consul")
	}

	if len(addrs) == 0 {
		// There is no vault service advertised
		return nil, errors.New("Number of vault addresses returned was 0")
	}

	// Get the S3 key and secret from vault
	vaultClient, err := newVaultClient(vaultAddress(addrs[0].Service.Address, addrs[0].Service.Port))

	if err != nil {
		return nil, errors.Wrap(err, "Error creating vault client")
	}

	return &VaultCredsProvider{vaultClient, key}, nil
}

// Retrieve gets a set of credentials from vault. this is used by AWS to retrieve the creds.
func (m *VaultCredsProvider) Retrieve() (credentials.Value, error) {
	data, err := m.vaultClient.ReadKey(m.vaultKey)
	if err != nil {
		return credentials.Value{}, errors.Wrap(err, "Could not read s3 secrets from vault")
	}

	if data == nil || data.Data["s3id"] == "" || data.Data["s3key"] == "" {
		return credentials.Value{}, errors.New("Vault key contained no key or secret data")
	}

	return credentials.Value{
		AccessKeyID:     data.Data["s3id"].(string),
		SecretAccessKey: data.Data["s3key"].(string),
	}, err
}

func vaultAddress(address string, port int) string {
	// Use the environment address for the vault server in preference
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr != "" {
		return vaultAddr
	}

	return fmt.Sprintf("https://%s:%d", address, port)
}

// IsExpired assumes the creds provided by vault are always current
func (m *VaultCredsProvider) IsExpired() bool {
	return false
}

package credentials

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/cloudflare/cfssl/log"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

const (
	consulAddress      string = "localhost:8500"
	vaultTokenFilename string = ".vault-token"
)

var (
	// Search path will be the users home dir, then any paths in this list
	vaultTokenSearchPath = []string{"/var"}
)

type vaultClient struct {
	vault *vault.Client
}

func newVaultClient(addr string) (*vaultClient, error) {
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = addr

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't create vault client")
	}

	token, err := vaultToken()
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't get vault token")
	}

	client.SetToken(token)

	return &vaultClient{vault: client}, nil
}

func vaultToken() (string, error) {
	token, err := vaultTokenFromUserConfig()
	if err == nil {
		return token, nil
	}

	for _, filepath := range vaultTokenSearchPath {
		pathToTokenFile := path.Join(filepath, vaultTokenFilename)
		token, _ := tokenFromFilename(pathToTokenFile)
		if err == nil {
			return token, nil
		}
	}

	return "", errors.New("Unable to load vault token from configured paths")
}

func vaultTokenFromUserConfig() (string, error) {
	homedir := os.Getenv("HOME")
	log.Debug("User home dir from environment is", homedir)
	if homedir == "" {
		return "", errors.New("Couldn't get home dir")
	}

	pathToTokenFile := path.Join(homedir, vaultTokenFilename)

	return tokenFromFilename(pathToTokenFile)
}

func tokenFromFilename(filename string) (string, error) {
	log.Debugf("Looking in %s for vault token", filename)
	tokenBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Debugf("Error reading file %s: %v", filename, err)
		return "", errors.Wrapf(err, "Couldn't read vault config file at %s", filename)
	}

	log.Debugf("Read vault token from %s", filename)
	return strings.TrimSpace(string(tokenBytes)), nil
}

func (v *vaultClient) ReadKey(key string) (*vault.Secret, error) {
	l := v.vault.Logical()
	vaultResponse, err := l.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading secret from vault server")
	}

	return vaultResponse, nil
}

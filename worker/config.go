package worker

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Repository struct {
	// Name of repository
	Name string `yaml:"name"`

	// URL to clone source (e.g. GitHub clone https/ssh URL)
	CloneUrl string `yaml:"cloneUrl"`

	// Option to disable usage of configured auth credentials
	DisableAuth bool `yaml:"disableAuth"`

	// Branches to check out and watch
	Branches []string
}

type Config struct {
	Auth struct {
		// Optional user to authenticate as
		User string `yaml:"user"`

		// Optional password to authenticate with
		Password string `yaml:"password"`

		// Personal access token to access repositories
		AccessToken string `yaml:"accessToken"`
	} `yaml:"auth"`

	// Interval of running git fetch to check for repository updates
	FetchInterval string `yaml:"fetchInterval"`

	// Directory to clone repositories into
	CloneDirectory string `yaml:"cloneDirectory"`

	// List of repositories
	Repositories []Repository `yaml:"repositories"`
}

// Load and prepare configuration
func (c *Config) Load(configFilePath string) error {
	// Read config file
	rawConfig, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("could not read configuration file: %w", err)
	}

	// Load config into memory
	err = yaml.Unmarshal(rawConfig, c)
	if err != nil {
		return fmt.Errorf("could not load config into memory: %w", err)
	}

	// Set defaults to make sure required values exist
	err = c.setDefaults()
	if err != nil {
		return fmt.Errorf("could not set configuration defaults: %w", err)
	}

	return c.validateAndTransform()
}

func (c *Config) setDefaults() error {
	if c.CloneDirectory == "" {
		return fmt.Errorf("missing clone directory in configuration")
	}

	if c.FetchInterval == "" {
		return fmt.Errorf("missing fetch interval in configuration")
	}

	if c.Auth.AccessToken == "" && c.Auth.User == "" {
		// Try to use environment-based access token
		envAccessToken := os.Getenv("HELFERLEIN_GIT_AUTH_ACCESS_TOKEN")
		if envAccessToken == "" {
			return fmt.Errorf("missing access token in configuration and environment")
		}

		c.Auth.AccessToken = envAccessToken
	}

	if (c.Auth.User == "" || c.Auth.Password == "") && c.Auth.AccessToken == "" {
		envUser := os.Getenv("HELFERLEIN_GIT_AUTH_USER")
		if c.Auth.User == "" {
			c.Auth.User = envUser
		}

		envPassword := os.Getenv("HELFERLEIN_GIT_AUTH_PASSWORD")
		if c.Auth.Password == "" {
			c.Auth.Password = envPassword
		}

		if c.Auth.User == "" || c.Auth.Password == "" {
			return fmt.Errorf("missing user or password in auth configuration and environment")
		}
	}

	return nil
}

func (c *Config) validateAndTransform() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	// Convert relative clone directory to absolute path
	if !filepath.IsAbs(c.CloneDirectory) {
		c.CloneDirectory = filepath.Join(wd, c.CloneDirectory)
	}

	return nil
}

func (c *Config) GetAuthCredentials() (transport.AuthMethod, error) {
	// Accept access tokens (GitHub w/ 2FA)
	if c.Auth.AccessToken != "" {
		return &http.BasicAuth{
			Username: "user",
			Password: c.Auth.AccessToken,
		}, nil
	}

	// Accept user/password combinations (GitHub without 2FA, not recommended)
	if c.Auth.User != "" && c.Auth.Password != "" {
		return &http.BasicAuth{
			Username: c.Auth.User,
			Password: c.Auth.Password,
		}, nil
	}

	return nil, fmt.Errorf("could not retrieve user/password combination or access token")
}

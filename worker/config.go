package worker

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type BranchConfig struct {
	// Commands to run in sequence
	Steps []string `yaml:"steps"`
}

type GitAuthConfig struct {
	// Optional user to authenticate as (HELFERLEIN_GIT_USER)
	User string `yaml:"user"`

	// Optional password to authenticate with (HELFERLEIN_GIT_PASSWORD)
	Password string `yaml:"password"`

	// Personal access token to access repository (HELFERLEIN_GIT_ACCESS_TOKEN)
	AccessToken string `yaml:"access_token"`
}

type ProjectConfig struct {
	Auth GitAuthConfig `yaml:"auth"`

	// Interval of running git fetch to check for repository updates
	FetchInterval time.Duration `yaml:"fetch_interval"`

	// URL to clone source (e.g. GitHub clone https/ssh URL)
	CloneUrl string `yaml:"clone_url"`

	// Branches to check out and watch
	Branches map[string]BranchConfig `yaml:"branches"`
}

type Config struct {
	// Directory to clone repositories into
	CloneDirectory string `yaml:"clone_directory"`

	// List of repositories
	Projects map[string]ProjectConfig `yaml:"projects"`
}

// Load and prepare configuration from file path
func (c *Config) LoadFromFile(configFilePath string) error {
	// Read config file
	rawConfig, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("could not read configuration file: %w", err)
	}

	return c.parse(rawConfig)
}

// Load and prepare configuration from bytes
func (c *Config) LoadFromBytes(contents []byte) error {
	return c.parse(contents)
}

// Load and prepare configuration
func (c *Config) parse(contents []byte) error {
	// Unmarshal config
	err := yaml.Unmarshal(contents, c)
	if err != nil {
		return fmt.Errorf("could not load config into memory: %w", err)
	}

	return c.validateAndTransform()
}

// Validate set config values and transform
// elements like file paths to meet usage requirements
func (c *Config) validateAndTransform() error {
	// Clone Directory value is required
	if c.CloneDirectory == "" {
		return fmt.Errorf("missing clone directory in configuration")
	}

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

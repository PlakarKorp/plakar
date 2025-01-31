package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Repositories []Repository `yaml:"repositories"`
	Agent        AgentConfig  `yaml:"agent"`
}

type Repository struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type AgentConfig struct {
	Alerting AlertingConfig `yaml:"alerting"`
	Tasks    []TaskSet      `yaml:"tasks"`
}

type AlertingConfig struct {
	Email []EmailConfig `yaml:"email"`
}

type EmailConfig struct {
	Name       string     `yaml:"name"`
	Sender     string     `yaml:"sender"`
	Recipients []string   `yaml:"recipients"`
	Smtp       SmtpConfig `yaml:"smtp"`
}

type SmtpConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type TaskSet struct {
	Name       string          `yaml:"name"`
	Repository Repository      `yaml:"repository"`
	Check      []CheckConfig   `yaml:"check,omitempty"`
	Sync       []SyncConfig    `yaml:"sync,omitempty"`
	Backup     []BackupConfig  `yaml:"backup,omitempty"`
	Restore    []RestoreConfig `yaml:"restore,omitempty"`
	Archive    []ArchiveConfig `yaml:"archive,omitempty"`
}

type CheckConfig struct {
	Description string `yaml:"description"`
	Name        string `yaml:"name"`
	Type        string `yaml:"type,omitempty"`
	Interval    string `yaml:"interval"`
	Path        string `yaml:"path,omitempty"`
}

type SyncConfig struct {
	Description string     `yaml:"description"`
	Name        string     `yaml:"name"`
	Repository  Repository `yaml:"repository"`
	Target      Repository `yaml:"target"`
	Interval    string     `yaml:"interval"`
}

type BackupConfig struct {
	Description string   `yaml:"description"`
	Name        string   `yaml:"name"`
	Tags        []string `yaml:"tags,omitempty"`
	Path        string   `yaml:"path"`
	Interval    string   `yaml:"interval"`
	Retention   string   `yaml:"retention"`
}

type RestoreConfig struct {
	Description string `yaml:"description"`
	Name        string `yaml:"name"`
	Interval    string `yaml:"interval"`
	Path        string `yaml:"path"`
	Target      string `yaml:"target"`
}

type ArchiveConfig struct {
	Description string `yaml:"description"`
	Name        string `yaml:"name"`
	Interval    string `yaml:"interval"`
	Format      string `yaml:"format"`
	Target      string `yaml:"target"`
}

func NewConfiguration() *Configuration {
	return &Configuration{}
}

func DefaultConfiguration() *Configuration {
	return &Configuration{}
}

// ParseConfig parses the YAML file into the Config struct.
func ParseConfigFile(filename string) (*Configuration, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Configuration
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

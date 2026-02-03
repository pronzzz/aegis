package config

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	Jobs    []Job          `json:"jobs"`
	Storage *Storage       `json:"storage,omitempty"`
	Restore *RestoreConfig `json:"restore,omitempty"`
}

type Storage struct {
	Type      string `json:"type"`       // "local" or "s3"
	Bucket    string `json:"bucket"`     // for S3
	Endpoint  string `json:"endpoint"`   // for S3
	Region    string `json:"region"`     // for S3 (optional)
	UseSSL    bool   `json:"use_ssl"`    // for S3
	AccessKey string `json:"access_key"` // Env var override preferred
	SecretKey string `json:"secret_key"` // Env var override preferred
}

type RestoreConfig struct {
	TargetDir        string   `json:"target_dir"`
	PriorityPatterns []string `json:"priority_patterns"`
}

type Job struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Interval string `json:"interval"` // e.g., "1h", "10m"
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadJSON(path string) (*Config, error) {
	return Load(path)
}

func (j Job) GetDuration() (time.Duration, error) {
	return time.ParseDuration(j.Interval)
}

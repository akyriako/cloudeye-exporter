package config

import (
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type CloudAuth struct {
	ProjectName string `yaml:"project_name"`
	ProjectID   string `yaml:"project_id"`
	DomainName  string `yaml:"domain_name"`
	AccessKey   string `yaml:"access_key"`
	Region      string `yaml:"region"`
	SecretKey   string `yaml:"secret_key"`
	AuthURL     string `yaml:"auth_url"`
	UserName    string `yaml:"user_name"`
	Password    string `yaml:"password"`
}

type Global struct {
	Port            string `yaml:"port"`
	Prefix          string `yaml:"prefix"`
	MetricsPath     string `yaml:"metrics_path"`
	MaxRoutines     int    `yaml:"max_routines"`
	ScrapeBatchSize int    `yaml:"scrape_batch_size"`
}

type CloudConfig struct {
	Auth   CloudAuth `yaml:"auth"`
	Global Global    `yaml:"global"`
}

const (
	DefaultPort            int    = 8087
	DefaultPrefix          string = "opentelekomcloud"
	DefaultMetricsPath     string = "/metrics"
	DefaultMaxRoutines     int    = 20
	DefaultScrapeBatchSize int    = 10
)

var (
	//go:embed metric_filter_config.yml
	metricsFiltersConfigFile []byte

	metricsFilters map[string]map[string][]string
)

func GetConfigFromFile(configPath string, enableFilters bool) (*CloudConfig, error) {
	var config CloudConfig

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	setDefaults(&config)

	if enableFilters {
		err := enableMetricFilters(enableFilters)
		if err != nil {
			return nil, err
		}
	}

	return &config, err
}

func setDefaults(config *CloudConfig) {
	if config.Global.Port == "" {
		config.Global.Port = fmt.Sprintf(":%d", DefaultPort)
	}

	if config.Global.MetricsPath == "" {
		config.Global.MetricsPath = DefaultMetricsPath
	}

	if config.Global.Prefix == "" {
		config.Global.Prefix = DefaultPrefix
	}

	if config.Global.MaxRoutines == 0 {
		config.Global.MaxRoutines = DefaultMaxRoutines
	}

	if config.Global.ScrapeBatchSize == 0 {
		config.Global.ScrapeBatchSize = DefaultScrapeBatchSize
	}
}

func enableMetricFilters(enable bool) error {
	metricsFilters = make(map[string]map[string][]string)
	err := yaml.Unmarshal(metricsFiltersConfigFile, &metricsFilters)
	if err != nil {
		return err
	}
	return nil
}

func GetMetricFilters(namespace string) map[string][]string {
	if configMap, ok := metricsFilters[namespace]; ok {
		return configMap
	}
	return nil
}

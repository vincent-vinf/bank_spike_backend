package config

import (
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

var (
	configPath string
	config     *Config
	once       sync.Once
)

func init() {
	flag.StringVar(&configPath, "config-path", "E:\\vincent\\repos\\bank_spike_backend\\configs\\config.yaml", "")
}

type Config struct {
	Mysql struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Passwd   string `yaml:"passwd"`
		Database string `yaml:"database"`
	}

	JWT struct {
		Secret     string        `yaml:"secret"`
		Timeout    time.Duration `yaml:"timeout"`
		MaxRefresh time.Duration `yaml:"maxRefresh"`
	}

	Redis struct {
		Endpoint string `yaml:"endpoint"`
		DB       int    `yaml:"db"`
		Passwd   string `yaml:"passwd"`
	}
}

func GetConfig() Config {
	if config == nil {
		once.Do(func() {
			config = &Config{}
			data, err := ioutil.ReadFile(configPath)
			if err != nil {
				log.Fatalln(err)
			}
			err = yaml.Unmarshal(data, config)
			if err != nil {
				log.Fatalln(err)
			}
		})
	}
	return *config
}

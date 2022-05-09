package config

import (
	"flag"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
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

func InitViper() {
	once.Do(func() {
		viper.SetConfigFile(configPath)
		err := viper.ReadInConfig()
		if err != nil {
			log.Fatalln(err)
		}
		viper.WatchConfig()
		viper.OnConfigChange(func(in fsnotify.Event) {
			if err := viper.Unmarshal(config); err != nil {
				log.Fatalf("unmarshal conf failed, err:%s \n", err)
			}
			log.Println("config reloaded")
		})
	})
}

type Config struct {
	Mysql struct {
		Host     string
		Port     string
		User     string
		Passwd   string
		Database string
	}

	JWT struct {
		Secret     string
		Timeout    time.Duration
		MaxRefresh time.Duration
	}

	AdminJWT struct {
		Secret     string
		Timeout    time.Duration
		MaxRefresh time.Duration
	} `mapstructure:"admin_jwt"`

	Redis struct {
		Endpoint string
		DB       int
		Passwd   string
	}

	RabbitMQ struct {
		Host   string
		Port   string
		User   string
		Passwd string
	} `mapstructure:"rabbitmq"`

	Spike struct {
		RandUrlKey string `mapstructure:"rand_url_key"`
	}
}

func GetConfig() Config {
	if config != nil {
		return *config
	}
	var c Config
	err := viper.Unmarshal(&c)
	if err != nil {
		log.Fatalln(err)
	}
	config = &c
	return c
}

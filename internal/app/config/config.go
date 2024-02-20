package config

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type KafkaConfig struct {
	Host  string `json:"host"`
	Port  int    `json:"port"`
	Topic string `json:"topic"`
}

type CodingServiceConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Config struct {
	Server ServerConfig `json:"server"`

	Kafka         KafkaConfig         `json:"kafka"`
	CodingService CodingServiceConfig `json:"coding_service"`
}

func NewConfig(ctx context.Context) (*Config, error) {
	var err error

	configName := "config.json"
	_ = godotenv.Load()

	if os.Getenv("MEGACHAT_CONFIG_NAME") != "" {
		configName = os.Getenv("MEGACHAT_CONFIG_NAME")
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("json")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.WatchConfig()

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

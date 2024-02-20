package config

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	ServerHost string
	ServerPort int

	CodingHost string
	CodingPort int

	KafkaHost  string
	KafkaPort  int
	KafkaTopic string
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

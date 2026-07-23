package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func NewViper() *viper.Viper {
	config := viper.New()

	config.SetConfigName("config")
	config.SetConfigType("json")
	config.AddConfigPath("./../")
	config.AddConfigPath("./")
	err := config.ReadInConfig()

	config.AutomaticEnv()
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	return config
}
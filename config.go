package main

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type ConfigVars struct {
	Discord         Discord  `mapstructure:",squash"`
	SubReddit       string   `mapstructure:"subreddit"`
	SearchStrings   []string `mapstructure:"search_strings"`
	ScrapeFrequency int64    `mapstructure:"scrape_frequency"`
}

type Discord struct {
	AppId    string `mapstructure:"discord_app_id"`
	BotToken string `mapstructure:"discord_bot_token"`
}

func LoadConfig(logger *log.Logger) (config ConfigVars, err error) {
	viper.Reset()

	logger.Println("Reading config file...")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Println("Config file not found")
		} else {
			// Config file was found but another error was produced
			return config, err
		}
	}

	err = viper.Unmarshal(&config)
	return config, err
}

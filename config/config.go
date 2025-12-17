package config

import (
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	DiscordToken string
	GuildID      string
	APIEndpoint  string
	APIKey       string
	APITimeout   time.Duration
}

func Load() (*Config, error) {
	godotenv.Load()

	viper.AutomaticEnv()

	viper.SetDefault("API_TIMEOUT", 60)

	viper.BindEnv("DISCORD_TOKEN")
	viper.BindEnv("DISCORD_GUILD_ID")
	viper.BindEnv("API_ENDPOINT")
	viper.BindEnv("API_KEY")
	viper.BindEnv("API_TIMEOUT")

	return &Config{
		DiscordToken: viper.GetString("DISCORD_TOKEN"),
		GuildID:      viper.GetString("DISCORD_GUILD_ID"),
		APIEndpoint:  viper.GetString("API_ENDPOINT"),
		APIKey:       viper.GetString("API_KEY"),
		APITimeout:   time.Duration(viper.GetInt("API_TIMEOUT")) * time.Second,
	}, nil
}

package config

import (
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	DiscordToken                 string
	GuildID                      string
	APIEndpoint                  string
	APIKey                       string
	APITimeout                   time.Duration
	StreamAPIEndpoint            string
	ElysiaAPIEndpoint            string
	ElysiaAPIKey                 string
	UnregisterCommandsOnShutdown bool
}

func Load() (*Config, error) {
	godotenv.Load()

	viper.AutomaticEnv()

	viper.SetDefault("API_TIMEOUT", 60)
	viper.SetDefault("UNREGISTER_COMMANDS_ON_SHUTDOWN", true)

	viper.BindEnv("DISCORD_TOKEN")
	viper.BindEnv("DISCORD_GUILD_ID")
	viper.BindEnv("API_ENDPOINT")
	viper.BindEnv("API_KEY")
	viper.BindEnv("API_TIMEOUT")
	viper.BindEnv("STREAM_API_ENDPOINT")
	viper.BindEnv("ELYSIA_API_ENDPOINT")
	viper.BindEnv("ELYSIA_API_KEY")
	viper.BindEnv("UNREGISTER_COMMANDS_ON_SHUTDOWN")

	return &Config{
		DiscordToken:                 viper.GetString("DISCORD_TOKEN"),
		GuildID:                      viper.GetString("DISCORD_GUILD_ID"),
		APIEndpoint:                  viper.GetString("API_ENDPOINT"),
		APIKey:                       viper.GetString("API_KEY"),
		APITimeout:                   time.Duration(viper.GetInt("API_TIMEOUT")) * time.Second,
		StreamAPIEndpoint:            viper.GetString("STREAM_API_ENDPOINT"),
		ElysiaAPIEndpoint:            viper.GetString("ELYSIA_API_ENDPOINT"),
		ElysiaAPIKey:                 viper.GetString("ELYSIA_API_KEY"),
		UnregisterCommandsOnShutdown: viper.GetBool("UNREGISTER_COMMANDS_ON_SHUTDOWN"),
	}, nil
}

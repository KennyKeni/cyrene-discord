package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/KennyKeni/cyrene-discord.git/bot"
	"github.com/KennyKeni/cyrene-discord.git/client"
	"github.com/KennyKeni/cyrene-discord.git/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if cfg.DiscordToken == "" {
		log.Fatal("DISCORD_TOKEN is required")
	}
	if cfg.APIEndpoint == "" {
		log.Fatal("API_ENDPOINT is required")
	}

	apiClient := client.New(cfg.APIEndpoint, cfg.APIKey, cfg.APITimeout)

	b, err := bot.New(cfg.DiscordToken, cfg.GuildID, apiClient)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	if err := b.Start(); err != nil {
		log.Fatalf("failed to start bot: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	if err := b.Stop(); err != nil {
		log.Printf("error during shutdown: %v", err)
	}
}

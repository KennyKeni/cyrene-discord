package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/KennyKeni/cyrene-discord.git/bot"
	"github.com/KennyKeni/cyrene-discord.git/client"
	"github.com/KennyKeni/cyrene-discord.git/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.DiscordToken == "" {
		slog.Error("missing required config", "field", "DISCORD_TOKEN")
		os.Exit(1)
	}
	if cfg.APIEndpoint == "" {
		slog.Error("missing required config", "field", "API_ENDPOINT")
		os.Exit(1)
	}
	if cfg.ElysiaAPIEndpoint == "" {
		slog.Error("missing required config", "field", "ELYSIA_API_ENDPOINT")
		os.Exit(1)
	}

	slog.Info("config loaded",
		"api_endpoint", cfg.APIEndpoint,
		"stream_api_endpoint", cfg.StreamAPIEndpoint,
		"elysia_api_endpoint", cfg.ElysiaAPIEndpoint,
		"api_timeout", cfg.APITimeout,
		"guild_id", cfg.GuildID,
	)

	chatClient := client.New(cfg.APIEndpoint, cfg.APIKey, cfg.APITimeout)
	elysiaClient := client.New(cfg.ElysiaAPIEndpoint, cfg.ElysiaAPIKey, cfg.APITimeout)
	streamClient := client.NewStreamClient(cfg.StreamAPIEndpoint, cfg.APIKey, cfg.APITimeout)

	b, err := bot.New(cfg.DiscordToken, cfg.GuildID, chatClient, elysiaClient, streamClient, cfg.UnregisterCommandsOnShutdown)
	if err != nil {
		slog.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	if err := b.Start(); err != nil {
		slog.Error("failed to start bot", "error", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop

	slog.Info("shutdown signal received", "signal", sig.String())
	if err := b.Stop(); err != nil {
		slog.Error("error during shutdown", "error", err)
	}
	slog.Info("shutdown complete")
}

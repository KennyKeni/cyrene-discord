package handler

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/KennyKeni/cyrene-discord.git/client"
	"github.com/bwmarrin/discordgo"
)

type StreamChatHandler struct {
	client      *client.StreamClient
	commandName string
}

func NewStreamChatHandler(c *client.StreamClient, commandName string) *StreamChatHandler {
	return &StreamChatHandler{client: c, commandName: commandName}
}

func (h *StreamChatHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != h.commandName {
		return
	}

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	options := i.ApplicationCommandData().Options
	var message string
	for _, opt := range options {
		if opt.Name == "message" {
			message = opt.StringValue()
			break
		}
	}

	slog.Info("stream command received",
		"command", h.commandName,
		"user_id", userID,
		"channel_id", i.ChannelID,
		"guild_id", i.GuildID,
		"message_length", len(message),
	)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		slog.Error("failed to send deferred response",
			"user_id", userID,
			"error", err,
		)
		return
	}

	var mu sync.Mutex
	var accumulated strings.Builder
	var lastUpdate time.Time
	updateInterval := 1 * time.Second
	done := make(chan struct{})

	updateMessage := func(final bool) {
		mu.Lock()
		content := accumulated.String()
		mu.Unlock()

		if content == "" {
			return
		}

		var edit *discordgo.WebhookEdit
		if len(content) <= maxMessageLength-10 {
			edit = &discordgo.WebhookEdit{Content: &content}
		} else {
			displayContent := content
			if len(displayContent) > maxEmbedLength-10 {
				displayContent = displayContent[:maxEmbedLength-13] + "..."
			}
			edit = &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{
					{Description: displayContent},
				},
			}
		}

		_, err := s.InteractionResponseEdit(i.Interaction, edit)
		if err != nil {
			slog.Error("failed to update stream response",
				"user_id", userID,
				"error", err,
			)
		}
	}

	go func() {
		ticker := time.NewTicker(updateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				mu.Lock()
				shouldUpdate := time.Since(lastUpdate) >= updateInterval
				mu.Unlock()

				if shouldUpdate {
					updateMessage(false)
					mu.Lock()
					lastUpdate = time.Now()
					mu.Unlock()
				}
			}
		}
	}()

	start := time.Now()
	err = h.client.SendStream(context.Background(), message, userID, func(chunk string) {
		mu.Lock()
		accumulated.WriteString(chunk)
		mu.Unlock()
	})
	duration := time.Since(start)

	close(done)

	if err != nil {
		slog.Error("failed to get stream response from API",
			"user_id", userID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		errorMsg := "Failed to get response. Please try again later."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorMsg})
		return
	}

	updateMessage(true)

	slog.Info("stream response completed",
		"user_id", userID,
		"duration_ms", duration.Milliseconds(),
		"response_length", accumulated.Len(),
	)
}

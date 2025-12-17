package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/KennyKeni/cyrene-discord.git/client"
	"github.com/bwmarrin/discordgo"
)

type ChatHandler struct {
	client *client.Client
}

func NewChatHandler(c *client.Client) *ChatHandler {
	return &ChatHandler{client: c}
}

func (h *ChatHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if i.ApplicationCommandData().Name != "chat" {
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

	slog.Info("chat command received",
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

	start := time.Now()
	response, err := h.client.Send(context.Background(), message, userID)
	duration := time.Since(start)

	if err != nil {
		slog.Error("failed to get response from API",
			"user_id", userID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		h.editResponse(s, i, userID, "Failed to get response. Please try again later.")
		return
	}

	slog.Info("api response received",
		"user_id", userID,
		"duration_ms", duration.Milliseconds(),
		"response_length", len(response),
	)

	h.editResponse(s, i, userID, response)
}

const maxMessageLength = 2000

func (h *ChatHandler) editResponse(s *discordgo.Session, i *discordgo.InteractionCreate, userID, content string) {
	truncated := false
	if len(content) > maxMessageLength {
		content = content[:maxMessageLength-3] + "..."
		truncated = true
	}
	if truncated {
		slog.Debug("response truncated",
			"user_id", userID,
			"original_length", len(content)+3,
			"max_length", maxMessageLength,
		)
	}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		slog.Error("failed to edit interaction response",
			"user_id", userID,
			"error", err,
		)
	}
}

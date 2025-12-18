package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/KennyKeni/cyrene-discord.git/client"
	"github.com/bwmarrin/discordgo"
)

type ChatHandler struct {
	client      *client.Client
	commandName string
}

func NewChatHandler(c *client.Client, commandName string) *ChatHandler {
	return &ChatHandler{client: c, commandName: commandName}
}

func (h *ChatHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	slog.Info("command received",
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

const (
	maxMessageLength = 2000
	maxEmbedLength   = 4096
)

func (h *ChatHandler) editResponse(s *discordgo.Session, i *discordgo.InteractionCreate, userID, content string) {
	var edit *discordgo.WebhookEdit

	if len(content) <= maxMessageLength-10 {
		edit = &discordgo.WebhookEdit{Content: &content}
	} else {
		truncated := false
		if len(content) > maxEmbedLength-10 {
			content = content[:maxEmbedLength-13] + "..."
			truncated = true
		}
		if truncated {
			slog.Debug("response truncated",
				"user_id", userID,
				"original_length", len(content)+3,
				"max_length", maxEmbedLength,
			)
		}
		edit = &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{Description: content},
			},
		}
	}

	_, err := s.InteractionResponseEdit(i.Interaction, edit)
	if err != nil {
		slog.Error("failed to edit interaction response",
			"user_id", userID,
			"error", err,
		)
	}
}

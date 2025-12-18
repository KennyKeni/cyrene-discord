package handler

import (
	"context"
	"log/slog"
	"strings"
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
	if len(content) <= maxMessageLength-10 {
		h.sendSimpleResponse(s, i, userID, content)
		return
	}

	if len(content) <= maxEmbedLength-10 {
		h.sendEmbedResponse(s, i, userID, content)
		return
	}

	h.sendThreadResponse(s, i, userID, content)
}

func (h *ChatHandler) sendSimpleResponse(s *discordgo.Session, i *discordgo.InteractionCreate, userID, content string) {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
	if err != nil {
		slog.Error("failed to edit interaction response",
			"user_id", userID,
			"error", err,
		)
	}
}

func (h *ChatHandler) sendEmbedResponse(s *discordgo.Session, i *discordgo.InteractionCreate, userID, content string) {
	edit := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{Description: content},
		},
	}
	_, err := s.InteractionResponseEdit(i.Interaction, edit)
	if err != nil {
		slog.Error("failed to edit interaction response",
			"user_id", userID,
			"error", err,
		)
	}
}

func (h *ChatHandler) sendThreadResponse(s *discordgo.Session, i *discordgo.InteractionCreate, userID, content string) {
	chunks := chunkContent(content, maxEmbedLength-10)

	firstChunk := chunks[0]
	edit := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{Description: firstChunk},
		},
	}
	msg, err := s.InteractionResponseEdit(i.Interaction, edit)
	if err != nil {
		slog.Error("failed to edit interaction response",
			"user_id", userID,
			"error", err,
		)
		return
	}

	if len(chunks) == 1 {
		return
	}

	thread, err := s.MessageThreadStart(i.ChannelID, msg.ID, "Full Response", 60)
	if err != nil {
		slog.Error("failed to create thread",
			"user_id", userID,
			"message_id", msg.ID,
			"error", err,
		)
		return
	}

	for idx, chunk := range chunks[1:] {
		_, err := s.ChannelMessageSendEmbed(thread.ID, &discordgo.MessageEmbed{
			Description: chunk,
		})
		if err != nil {
			slog.Error("failed to send thread message",
				"user_id", userID,
				"thread_id", thread.ID,
				"chunk_index", idx+1,
				"error", err,
			)
		}
	}

	slog.Info("response sent via thread",
		"user_id", userID,
		"thread_id", thread.ID,
		"chunk_count", len(chunks),
	)
}

func chunkContent(content string, maxLen int) []string {
	var chunks []string

	for len(content) > 0 {
		if len(content) <= maxLen {
			chunks = append(chunks, content)
			break
		}

		cutPoint := maxLen
		if idx := strings.LastIndex(content[:maxLen], "\n"); idx > maxLen/2 {
			cutPoint = idx + 1
		} else if idx := strings.LastIndex(content[:maxLen], ". "); idx > maxLen/2 {
			cutPoint = idx + 2
		}

		chunks = append(chunks, strings.TrimSpace(content[:cutPoint]))
		content = strings.TrimSpace(content[cutPoint:])
	}

	return chunks
}

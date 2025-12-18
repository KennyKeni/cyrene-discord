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

func (h *ChatHandler) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		return
	}

	if !channel.IsThread() {
		return
	}

	if channel.Name != h.commandName {
		return
	}

	slog.Info("thread message received",
		"command", h.commandName,
		"user_id", m.Author.ID,
		"thread_id", m.ChannelID,
		"message_length", len(m.Content),
	)

	s.ChannelTyping(m.ChannelID)

	start := time.Now()
	response, err := h.client.Send(context.Background(), m.Content, m.ChannelID)
	duration := time.Since(start)

	if err != nil {
		slog.Error("failed to get response from API",
			"user_id", m.Author.ID,
			"thread_id", m.ChannelID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		h.sendToThread(s, m.ChannelID, "Failed to get response. Please try again later.")
		return
	}

	slog.Info("api response received",
		"user_id", m.Author.ID,
		"thread_id", m.ChannelID,
		"duration_ms", duration.Milliseconds(),
		"response_length", len(response),
	)

	h.sendToThread(s, m.ChannelID, response)
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

	threadID := i.ChannelID
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		slog.Error("failed to get channel info",
			"channel_id", i.ChannelID,
			"error", err,
		)
	}

	isThread := channel != nil && channel.IsThread()

	if !isThread {
		questionContent := message
		msg, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &questionContent,
		})
		if err != nil {
			slog.Error("failed to edit interaction response",
				"user_id", userID,
				"error", err,
			)
			return
		}

		threadName := h.commandName
		thread, err := s.MessageThreadStart(i.ChannelID, msg.ID, threadName, 60)
		if err != nil {
			slog.Error("failed to create thread",
				"user_id", userID,
				"message_id", msg.ID,
				"error", err,
			)
			return
		}
		threadID = thread.ID

		slog.Info("thread created",
			"user_id", userID,
			"thread_id", threadID,
		)
	}

	s.ChannelTyping(threadID)

	start := time.Now()
	response, err := h.client.Send(context.Background(), message, threadID)
	duration := time.Since(start)

	if err != nil {
		slog.Error("failed to get response from API",
			"user_id", userID,
			"thread_id", threadID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		h.sendToThread(s, threadID, "Failed to get response. Please try again later.")
		return
	}

	slog.Info("api response received",
		"user_id", userID,
		"thread_id", threadID,
		"duration_ms", duration.Milliseconds(),
		"response_length", len(response),
	)

	h.sendToThread(s, threadID, response)
}

const (
	maxMessageLength = 2000
	maxEmbedLength   = 4096
)

func (h *ChatHandler) sendToThread(s *discordgo.Session, threadID, content string) {
	chunks := chunkContent(content, maxEmbedLength-10)

	for idx, chunk := range chunks {
		var err error
		if len(chunk) <= maxMessageLength-10 {
			_, err = s.ChannelMessageSend(threadID, chunk)
		} else {
			_, err = s.ChannelMessageSendEmbed(threadID, &discordgo.MessageEmbed{
				Description: chunk,
			})
		}
		if err != nil {
			slog.Error("failed to send thread message",
				"thread_id", threadID,
				"chunk_index", idx,
				"error", err,
			)
		}
	}
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

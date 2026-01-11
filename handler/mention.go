package handler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/KennyKeni/cyrene-discord.git/client"
	"github.com/bwmarrin/discordgo"
)

type MentionHandler struct {
	client *client.Client
}

func NewMentionHandler(c *client.Client) *MentionHandler {
	return &MentionHandler{client: c}
}

func (h *MentionHandler) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if !isBotMentioned(s, m) {
		return
	}

	message := extractMessage(s, m.Content)
	if message == "" {
		return
	}

	userID := m.Author.ID

	slog.Info("mention received",
		"user_id", userID,
		"channel_id", m.ChannelID,
		"guild_id", m.GuildID,
		"message_length", len(message),
	)

	if err := s.ChannelTyping(m.ChannelID); err != nil {
		slog.Debug("failed to send typing indicator", "error", err)
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
		h.sendResponse(s, m, "Failed to get response. Please try again later.")
		return
	}

	slog.Info("api response received",
		"user_id", userID,
		"duration_ms", duration.Milliseconds(),
		"response_length", len(response),
	)

	h.sendResponse(s, m, response)
}

func isBotMentioned(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	for _, user := range m.Mentions {
		if user.ID == s.State.User.ID {
			return true
		}
	}
	return false
}

func extractMessage(s *discordgo.Session, content string) string {
	botID := s.State.User.ID
	content = strings.ReplaceAll(content, "<@"+botID+">", "")
	content = strings.ReplaceAll(content, "<@!"+botID+">", "")
	return strings.TrimSpace(content)
}

func (h *MentionHandler) sendResponse(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	if len(content) <= maxMessageLength-10 {
		h.sendSimpleResponse(s, m, content)
		return
	}

	if len(content) <= maxEmbedLength-10 {
		h.sendEmbedResponse(s, m, content)
		return
	}

	h.sendThreadResponse(s, m, content)
}

func (h *MentionHandler) sendSimpleResponse(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	_, err := s.ChannelMessageSendReply(m.ChannelID, content, m.Reference())
	if err != nil {
		slog.Error("failed to send message reply",
			"user_id", m.Author.ID,
			"channel_id", m.ChannelID,
			"error", err,
		)
	}
}

func (h *MentionHandler) sendEmbedResponse(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	_, err := s.ChannelMessageSendEmbedReply(m.ChannelID, &discordgo.MessageEmbed{
		Description: content,
	}, m.Reference())
	if err != nil {
		slog.Error("failed to send embed reply",
			"user_id", m.Author.ID,
			"channel_id", m.ChannelID,
			"error", err,
		)
	}
}

func (h *MentionHandler) sendThreadResponse(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	chunks := chunkContent(content, maxEmbedLength-10)

	firstChunk := chunks[0]
	msg, err := s.ChannelMessageSendEmbedReply(m.ChannelID, &discordgo.MessageEmbed{
		Description: firstChunk,
	}, m.Reference())
	if err != nil {
		slog.Error("failed to send embed reply",
			"user_id", m.Author.ID,
			"channel_id", m.ChannelID,
			"error", err,
		)
		return
	}

	if len(chunks) == 1 {
		return
	}

	thread, err := s.MessageThreadStart(m.ChannelID, msg.ID, "Full Response", 60)
	if err != nil {
		slog.Error("failed to create thread",
			"user_id", m.Author.ID,
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
				"user_id", m.Author.ID,
				"thread_id", thread.ID,
				"chunk_index", idx+1,
				"error", err,
			)
		}
	}

	slog.Info("response sent via thread",
		"user_id", m.Author.ID,
		"thread_id", thread.ID,
		"chunk_count", len(chunks),
	)
}

package handler

import (
	"context"
	"log"

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

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("failed to send deferred response: %v", err)
		return
	}

	options := i.ApplicationCommandData().Options
	var message string
	for _, opt := range options {
		if opt.Name == "message" {
			message = opt.StringValue()
			break
		}
	}

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	response, err := h.client.Send(context.Background(), message, userID)
	if err != nil {
		log.Printf("failed to get response from API: %v", err)
		h.editResponse(s, i, "Failed to get response. Please try again later.")
		return
	}

	h.editResponse(s, i, response)
}

const maxMessageLength = 2000

func (h *ChatHandler) editResponse(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if len(content) > maxMessageLength {
		content = content[:maxMessageLength-3] + "..."
	}
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		log.Printf("failed to edit interaction response: %v", err)
	}
}

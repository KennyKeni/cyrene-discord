package bot

import (
	"log/slog"

	"github.com/KennyKeni/cyrene-discord.git/client"
	"github.com/KennyKeni/cyrene-discord.git/handler"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session     *discordgo.Session
	guildID     string
	chatHandler *handler.ChatHandler
	commandIDs  []string
}

var chatCommand = &discordgo.ApplicationCommand{
	Name:        "chat",
	Description: "Send a message to the AI assistant",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "message",
			Description: "Your message to send",
			Required:    true,
		},
	},
}

func New(token, guildID string, apiClient *client.Client) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		session:     session,
		guildID:     guildID,
		chatHandler: handler.NewChatHandler(apiClient),
	}, nil
}

func (b *Bot) Start() error {
	b.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		go b.chatHandler.Handle(s, i)
	})

	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		slog.Info("bot ready",
			"username", r.User.Username,
			"discriminator", r.User.Discriminator,
			"bot_id", r.User.ID,
		)
	})

	if err := b.session.Open(); err != nil {
		return err
	}

	cmd, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.guildID, chatCommand)
	if err != nil {
		return err
	}
	b.commandIDs = append(b.commandIDs, cmd.ID)

	slog.Info("command registered",
		"command", chatCommand.Name,
		"command_id", cmd.ID,
		"guild_id", b.guildID,
	)
	return nil
}

func (b *Bot) Stop() error {
	for _, cmdID := range b.commandIDs {
		if err := b.session.ApplicationCommandDelete(b.session.State.User.ID, b.guildID, cmdID); err != nil {
			slog.Error("failed to delete command", "command_id", cmdID, "error", err)
		} else {
			slog.Debug("command deleted", "command_id", cmdID)
		}
	}
	slog.Info("commands cleaned up", "count", len(b.commandIDs))

	return b.session.Close()
}

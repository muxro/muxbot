package bot

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type Content interface {
	ToMessage(ctx context.Context, replyTo *discordgo.Message) Message
}

type Message interface {
	Content() string
	Embed() *discordgo.MessageEmbed
}

type OnSender interface {
	OnSend(ctx context.Context, b *Bot, msg *discordgo.Message) error
}

type OnEditer interface {
	OnEdit(ctx context.Context, b *Bot, msg *discordgo.Message) error
}

type OnReactAdder interface {
	OnReactAdd(context.Context, *Bot, *discordgo.MessageReaction) error
}

type OnReactRemover interface {
	OnReactRemove(context.Context, *Bot, *discordgo.MessageReaction) error
}

type textMessage string

func (tm textMessage) Content() string                { return string(tm) }
func (tm textMessage) Embed() *discordgo.MessageEmbed { return nil }

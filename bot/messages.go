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
	OnSend(ctx context.Context, dmsg *discordgo.Message) error
}

type OnEditer interface {
	OnEdit(ctx context.Context, dmsg *discordgo.Message, cur Message) error
}

type OnReactAdder interface {
	OnReactAdd(context.Context, *discordgo.MessageReaction) error
}

type OnReactRemover interface {
	OnReactRemove(context.Context, *discordgo.MessageReaction) error
}

type MessageFunc func(ctx context.Context, replyTo *discordgo.Message) Message

func (mf MessageFunc) ToMessage(ctx context.Context, replyTo *discordgo.Message) Message {
	return mf(ctx, replyTo)
}

func Send(ctx context.Context, content Content) error {
	hmsg, _ := ctx.Value(ctxHistoryKey).(*message)
	if hmsg == nil {
		return nil
	}

	hmsg.Lock()
	defer hmsg.Unlock()

	if hmsg.removed {
		return nil
	}

	msg := content.ToMessage(ctx, hmsg.ReplyTo)

	bot := FromContext(ctx)
	return bot.sendMessage(ctx, hmsg, msg)
}

func SendMessage(ctx context.Context, msg Message) error {
	hmsg, _ := ctx.Value(ctxHistoryKey).(*message)
	if hmsg == nil {
		return nil
	}

	hmsg.Lock()
	defer hmsg.Unlock()

	if hmsg.removed {
		return nil
	}

	bot := FromContext(ctx)
	return bot.sendMessage(ctx, hmsg, msg)
}

func (b *Bot) sendMessage(ctx context.Context, hmsg *message, msg Message) error {
	if hmsg.Sent == nil {
		newMsg, err := b.Disco.ChannelMessageSendComplex(hmsg.ReplyTo.ChannelID, &discordgo.MessageSend{
			Content: msg.Content(),
			Embed:   msg.Embed(),
		})
		if err != nil {
			return err
		}

		hmsg.Sent = newMsg
		hmsg.Message = msg

		go func() {
			if msg, ok := msg.(OnSender); ok {
				err := msg.OnSend(ctx, newMsg)
				if err != nil {
					// TODO: log
				}
			}
		}()

		return nil
	}

	cont := msg.Content()
	editMsg, err := b.Disco.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      hmsg.Sent.ID,
		Channel: hmsg.Sent.ChannelID,
		Content: &cont,
		Embed:   msg.Embed(),
	})
	if err != nil {
		return err
	}

	prevMsg := hmsg.Message
	hmsg.Sent = editMsg
	hmsg.Message = msg

	go func() {
		if on, ok := prevMsg.(OnEditer); ok {
			err := on.OnEdit(ctx, editMsg, msg)
			if err != nil {
				// TODO: log
			}
		}

		if on, ok := msg.(OnSender); ok {
			err := on.OnSend(ctx, editMsg)
			if err != nil {
				// TODO: log
			}
		}
	}()

	return nil
}

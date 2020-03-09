package bot

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type contextKey struct {
	name string
}

var (
	ctxBotKey     = &contextKey{"bot"}
	ctxHistoryKey = &contextKey{"history message"}
)

func FromContext(ctx context.Context) *Bot {
	bot, _ := ctx.Value(ctxBotKey).(*Bot)
	return bot
}

func Reply(ctx context.Context, content Content) error {
	hmsg, _ := ctx.Value(ctxHistoryKey).(*message)
	if hmsg == nil {
		return nil
	}

	hmsg.mu.Lock()
	defer hmsg.mu.Unlock()

	if hmsg.removed {
		return nil
	}

	msg := content.ToMessage(ctx, hmsg.ReplyTo)

	bot := FromContext(ctx)
	if hmsg.Sent == nil {
		newMsg, err := bot.Disco.ChannelMessageSendComplex(hmsg.ReplyTo.ChannelID, &discordgo.MessageSend{
			Content: msg.Content(),
			Embed:   msg.Embed(),
		})
		if err != nil {
			return err
		}

		hmsg.Sent = newMsg

		//if msg, ok := msg.(OnSender); ok {
		//	msg.OnSend(context.TODO(), b, dmsg)
		//}

		return nil
	}

	cont := msg.Content()
	editMsg, err := bot.Disco.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      hmsg.Sent.ID,
		Channel: hmsg.Sent.ChannelID,
		Content: &cont,
		Embed:   msg.Embed(),
	})
	if err != nil {
		return err
	}

	hmsg.Sent = editMsg

	//if on, ok := reply.Content.(OnEditer); ok {
	//	err := on.OnEdit(context.TODO(), b, editMsg)
	//	if err != nil {
	//		return err
	//	}
	//}

	//if on, ok := msg.(OnSender); ok {
	//	err := on.OnSend(context.TODO(), b, editMsg)
	//	if err != nil {
	//		return err
	//	}
	//}

	return nil
}

func Edit(ctx context.Context, content Content) error {
	hmsg, _ := ctx.Value(ctxHistoryKey).(*message)
	if hmsg == nil {
		return nil
	}

	hmsg.mu.Lock()
	defer hmsg.mu.Unlock()

	if hmsg.removed {
		return nil
	}

	msg := content.ToMessage(ctx, hmsg.ReplyTo)

	bot := FromContext(ctx)
	if hmsg.Sent == nil {
		return nil
	}

	cont := msg.Content()
	editMsg, err := bot.Disco.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      hmsg.Sent.ID,
		Channel: hmsg.Sent.ChannelID,
		Content: &cont,
		Embed:   msg.Embed(),
	})
	if err != nil {
		return err
	}

	hmsg.Sent = editMsg

	//if on, ok := reply.Content.(OnEditer); ok {
	//	err := on.OnEdit(context.TODO(), b, editMsg)
	//	if err != nil {
	//		return err
	//	}
	//}

	//if on, ok := msg.(OnSender); ok {
	//	err := on.OnSend(context.TODO(), b, editMsg)
	//	if err != nil {
	//		return err
	//	}
	//}

	return nil

}

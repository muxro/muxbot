package bot

import (
	"log"
	"runtime/debug"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// MessageHandler is the base handler for commands
type MessageHandler func(bot *Bot, message *discordgo.Message) (bool, error)

func (b *Bot) RegisterHandler(handler MessageHandler) {
	b.handlers = append(b.handlers, handler)
}

func (b *Bot) onMessage(msg *discordgo.Message) {
	if msg.Author == nil || msg.Author.ID == b.Disco.State.User.ID {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			b.SendReply(msg, "something went wrong... ")
			log.Println(r, string(debug.Stack()))
		}
	}()

	content := strings.TrimSpace(msg.Content)
	if strings.HasPrefix(content, b.config.Prefix) {
		err := b.onCommand(msg, content[len(b.config.Prefix):])
		if err != nil {
			panic(err)
		}
		return
	}

	for _, handler := range b.handlers {
		handled, err := handler(b, msg)
		if err != nil {
			panic(err)
		}

		if handled {
			break
		}
	}
}

func (b *Bot) onMessageCreate(session *discordgo.Session, msg *discordgo.MessageCreate) {
	b.onMessage(msg.Message)
}

func (b *Bot) onMessageEdit(session *discordgo.Session, msg *discordgo.MessageUpdate) {
	b.onMessage(msg.Message)
}

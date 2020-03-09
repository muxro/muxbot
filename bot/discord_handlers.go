package bot

import (
	"context"
	"log"
	"runtime/debug"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) registerDiscordHandlers() {
	//b.Disco.AddHandler(b.onReady)
	b.Disco.AddHandler(b.onMessageCreate)
	b.Disco.AddHandler(b.onMessageEdit)

	b.Disco.AddHandler(b.onMessageReactionAdd)
	b.Disco.AddHandler(b.onMessageReactionRemove)
}

// MessageHandler is the base handler for commands
type MessageHandler func(context.Context, *discordgo.Message) (bool, error)

func (b *Bot) AddHandler(prio int, handler MessageHandler) {
	ph := prioHandler{prio: prio, handler: handler}
	b.handlers = append(b.handlers, ph)
	sort.Slice(b.handlers, func(i, j int) bool { return b.handlers[i].prio < b.handlers[j].prio })
}

type prioHandler struct {
	prio    int
	handler MessageHandler
}

type handlers []prioHandler

func (b *Bot) onMessage(msg *discordgo.Message) {
	if msg.Author == nil || b.IsMe(msg.Author.ID) {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("panicked: %v\n%s", r, string(debug.Stack()))
		}
	}()

	ctx, cancel := context.WithTimeout(b.ctx, 20*time.Second)

	hmsg := b.history.Add(msg)
	ctx = context.WithValue(ctx, ctxHistoryKey, hmsg)

	go func() {
		defer cancel()

		for _, ph := range b.handlers {
			handled, err := ph.handler(ctx, msg)
			if err != nil {
				log.Println("there was an error in the handler", err)
				return
			}

			if handled {
				break
			}
		}
		cancel()
	}()
}

func (b *Bot) onMessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {
	b.onMessage(msg.Message)
}

func (b *Bot) onMessageEdit(s *discordgo.Session, msg *discordgo.MessageUpdate) {
	b.onMessage(msg.Message)
}

func (b *Bot) onMessageReactionAdd(s *discordgo.Session, react *discordgo.MessageReactionAdd) {
	if b.IsMe(react.UserID) {
		return
	}

	hmsg := b.history.GetMessage(react.ChannelID, react.MessageID)
	if hmsg == nil {
		return
	}

	ctx, cancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer cancel()

	ctx = context.WithValue(ctx, ctxHistoryKey, hmsg)

	if on, ok := hmsg.Message.(OnReactAdder); ok {
		err := on.OnReactAdd(ctx, b, react.MessageReaction)
		if err != nil {
			log.Printf("OnReactAdd: %v", err)
		}
	}
}

func (b *Bot) onMessageReactionRemove(s *discordgo.Session, react *discordgo.MessageReactionRemove) {
	if b.IsMe(react.UserID) {
		return
	}

	hmsg := b.history.GetMessage(react.ChannelID, react.MessageID)
	if hmsg == nil {
		return
	}

	ctx, cancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer cancel()

	ctx = context.WithValue(ctx, ctxHistoryKey, hmsg)

	if on, ok := hmsg.Message.(OnReactRemover); ok {
		err := on.OnReactRemove(ctx, b, react.MessageReaction)
		if err != nil {
			log.Printf("OnReactRemove: %v", err)
		}
	}
}

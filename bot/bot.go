package bot

import (
	"context"
	"errors"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Disco *discordgo.Session

	cmds     commands
	handlers handlers

	// bot message history
	history *msgHistory

	config *Config

	mu  sync.Mutex
	ctx context.Context
}

type Config struct {
	Token  string
	Prefix string

	Addons []interface{}
}

func New(config Config) (*Bot, error) {
	disco, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, err
	}
	disco.MaxRestRetries = 3
	disco.ShouldReconnectOnError = true

	b := &Bot{
		Disco: disco,

		// 50 bot messages remembered per channel
		history: newMessageHistory(50),

		config: &config,
	}

	b.AddHandler(50, b.commandHandler)

	b.registerDiscordHandlers()

	return b, nil
}

func (b *Bot) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.ctx != nil {
		b.mu.Unlock()
		return errors.New("already started")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	err := b.Disco.Open()
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()

		b.mu.Lock()
		b.ctx = nil
		b.Disco.Close()
		b.mu.Unlock()
	}()

	ctx = context.WithValue(ctx, ctxBotKey, b)
	b.ctx = ctx
	return nil
}

func (b *Bot) Config() Config {
	return *b.config
}

func (b *Bot) IsMe(id string) bool {
	return id == b.Disco.State.User.ID
}

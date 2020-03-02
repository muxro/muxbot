package bot

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/urfave/cli/v2"
)

type Bot struct {
	Disco *discordgo.Session

	commands  []*cli.Command
	handlers  []MessageHandler
	replyHist *replyHistory

	config *Config
	ctx    context.Context
}

// replyHistory is stores the last max messages to update if the original message is edited
type replyHistory struct {
	no   int
	msgs [][]*discordgo.Message
}

// newMessageHistory creates a replyHistory instance with `max` possible slots
func newMessageHistory(max int) *replyHistory {
	return &replyHistory{
		msgs: make([][]*discordgo.Message, max, max),
	}
}

// Add an element to the history "cache" with rollover
func (rh *replyHistory) Add(msg, reply *discordgo.Message) {
	rh.msgs[rh.no] = []*discordgo.Message{msg, reply}
	rh.no = (rh.no + 1) % len(rh.msgs)
}

type Config struct {
	Token  string
	Prefix string

	Addons []interface{}
}

func New(ctx context.Context, config Config) (*Bot, error) {
	disco, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		Disco: disco,

		replyHist: newMessageHistory(1000),

		config: &config,
		ctx:    ctx,
	}

	bot.registerHandlers()

	return bot, nil
}

func (b *Bot) Start() error {
	return b.Disco.Open()
}

func (b *Bot) Config() Config {
	return *b.config
}

func (b *Bot) registerHandlers() {
	//b.Disco.AddHandler(b.onReady)
	b.Disco.AddHandler(b.onMessageCreate)
	b.Disco.AddHandler(b.onMessageEdit)
}

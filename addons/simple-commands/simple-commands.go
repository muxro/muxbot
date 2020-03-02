package simple_commands

import (
	"github.com/urfave/cli/v2"
	"gitlab.com/muxro/muxbot/addons"
	"gitlab.com/muxro/muxbot/bot"
)

func init() {
	addons.Register("simple-commands", SimpleCommands{})
}

type SimpleCommands struct{}

func (_ SimpleCommands) Add(b *bot.Bot) error {
	b.RegisterCommand(&cli.Command{
		Name:   "ping",
		Action: bot.CommandHandler(ping),
	})

	b.RegisterCommand(&cli.Command{
		Name:   "echo",
		Action: bot.CommandHandler(echo),
	})

	return nil
}

func ping() string {
	return "pong"
}

func echo(args string) string {
	return args
}

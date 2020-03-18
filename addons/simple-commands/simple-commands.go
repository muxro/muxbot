package simple_commands

import (
	"github.com/urfave/cli/v2"
	"gitlab.com/muxro/muxbot/addons"
	"gitlab.com/muxro/muxbot/bot"
)

func init() {
	addons.Register("test", SimpleCommands{})
}

type SimpleCommands struct{}

func (_ SimpleCommands) Add(b *bot.Bot) error {
	b.RegisterCommand(&cli.Command{
		Name:   "ping",
		Usage:  `replies with "pong"`,
		Action: bot.CommandHandler(ping),
	})

	b.RegisterCommand(&cli.Command{
		Name:   "echo",
		Usage:  "echoes back the given text",
		Action: bot.CommandHandler(echo),
	})

	b.RegisterCommand(&cli.Command{
		Name:   "escape",
		Usage:  "escape the given text",
		Action: bot.CommandHandler(escape),
	})

	return nil
}

func ping() bot.Message {
	return "pong"
}

func echo(arg string) bot.Message {
	return bot.RawString{arg}
}

func escape(arg string) bot.Message {
	return arg
}

package bot

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mattn/go-shellwords"
	"github.com/urfave/cli/v2"
)

func (b *Bot) RegisterCommand(cmd *cli.Command) {
	b.commands = append(b.commands, cmd)
}

func toFixedHandler(fn interface{}) func(*Bot, *discordgo.Message, *cli.Context) error {
	switch fn := fn.(type) {
	case func() string:
		return func(b *Bot, msg *discordgo.Message, c *cli.Context) error {
			reply := fn()
			b.SendReply(msg, reply)
			return nil
		}

	case func(string) string:
		return func(b *Bot, msg *discordgo.Message, c *cli.Context) error {
			args := strings.Join(c.Args().Slice(), " ")
			reply := fn(args)
			b.SendReply(msg, reply)
			return nil
		}

	case func(string) (string, error):
		return func(b *Bot, msg *discordgo.Message, c *cli.Context) error {
			args := strings.Join(c.Args().Slice(), " ")
			reply, err := fn(args)
			if err != nil {
				return err
			}
			b.SendReply(msg, reply)
			return nil
		}

	default:
		panic(fmt.Sprintf("unsupported handler type: %T", fn))
	}
}

func CommandHandler(handler interface{}) cli.ActionFunc {
	fixedHandler := toFixedHandler(handler)

	return func(c *cli.Context) error {
		bot := c.App.Metadata["bot"].(*Bot)
		msg := c.App.Metadata["message"].(*discordgo.Message)
		return fixedHandler(bot, msg, c)
	}
}

func (b *Bot) onCommand(msg *discordgo.Message, cmd string) error {
	parser := shellwords.NewParser()
	parsed, err := parser.Parse(cmd)
	if err != nil {
		return err
	}

	params := append([]string{"$t"}, parsed...)

	var outBuf bytes.Buffer
	app := &cli.App{
		Writer:    &outBuf,
		ErrWriter: &outBuf,
		Metadata:  map[string]interface{}{"bot": b, "message": msg},
		Commands:  b.commands,
	}

	err = app.Run(params)
	if err != nil {
		return err
	}

	if outBuf.String() != "" {
		b.SendReply(msg, outBuf.String())
	}
	return nil

}

package test_commands

import (
	"strconv"
	"time"

	"github.com/urfave/cli/v2"
	"gitlab.com/muxro/muxbot/addons"
	"gitlab.com/muxro/muxbot/bot"
)

func init() {
	addons.Register("simple-commands", TestCommands{})
}

type TestCommands struct{}

func (_ TestCommands) Add(b *bot.Bot) error {
	b.AddCommand(&cli.Command{
		Name:     "t-size",
		Usage:    "replies with a message of given size",
		Category: "test",
		Action:   bot.CommandHandler(testSize),
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:    "delay",
				Aliases: []string{"d"},
				Usage:   "delay until the showing the message",
			},
			&cli.DurationFlag{
				Name:    "sleep",
				Aliases: []string{"s"},
				Usage:   "how long to sleep until returning",
			},
			&cli.StringFlag{
				Name:    "header",
				Aliases: []string{"hdr"},
				Usage:   "message header",
			},
			&cli.BoolFlag{
				Name:    "raw",
				Aliases: []string{"r"},
				Usage:   "string should be treated as raw",
			},
			&cli.StringFlag{
				Name:    "footer",
				Aliases: []string{"ft"},
				Usage:   "message footer",
			},
			&cli.BoolFlag{
				Name:    "quoted",
				Aliases: []string{"q"},
				Usage:   "string should be treated as quoted",
			},
			&cli.StringFlag{
				Name:    "quote-type",
				Aliases: []string{"qt"},
				Usage:   "quote type",
			},
			&cli.BoolFlag{
				Name:    "pagination",
				Aliases: []string{"p"},
				Usage:   "enable pagination",
			},
			&cli.BoolFlag{
				Name:    "paste",
				Aliases: []string{"pb"},
				Usage:   "enable pastebin",
			},
		},
	})

	b.AddCommand(&cli.Command{
		Name:     "t-echo",
		Usage:    "echoes the given string",
		Category: "test",
		Action:   bot.CommandHandler(testEcho),
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:    "delay",
				Aliases: []string{"d"},
				Usage:   "delay until the showing the message",
			},
			&cli.DurationFlag{
				Name:    "sleep",
				Aliases: []string{"s"},
				Usage:   "how long to sleep until returning",
			},
			&cli.StringFlag{
				Name:    "header",
				Aliases: []string{"hdr"},
				Usage:   "message header",
			},
			&cli.BoolFlag{
				Name:    "raw",
				Aliases: []string{"r"},
				Usage:   "string should be treated as raw",
			},
			&cli.StringFlag{
				Name:    "footer",
				Aliases: []string{"ft"},
				Usage:   "message footer",
			},
			&cli.BoolFlag{
				Name:    "quoted",
				Aliases: []string{"q"},
				Usage:   "string should be treated as quoted",
			},
			&cli.StringFlag{
				Name:    "quote-type",
				Aliases: []string{"qt"},
				Usage:   "quote type",
			},
			&cli.BoolFlag{
				Name:    "pagination",
				Aliases: []string{"p"},
				Usage:   "enable pagination",
			},
			&cli.BoolFlag{
				Name:    "paste",
				Aliases: []string{"pb"},
				Usage:   "enable pastebin",
			},
		},
	})

	//b.AddCommand(&cli.Command{
	//	Name:     "t-embed",
	//	Usage:    "test messages with embeds",
	//	Category: "test",
	//	Action:   bot.CommandHandler(testEmbed),
	//})

	//b.AddCommand(&cli.Command{
	//	Name:     "t-react",
	//	Usage:    "test message reacts",
	//	Category: "test",
	//	Action:   bot.CommandHandler(testReact),
	//})

	return nil
}

func testSize(c *bot.CommandContext) bot.Content {
	size, err := strconv.Atoi(c.Args().First())
	if err != nil {
		// TODO: handle errors
		return bot.Text{Content: err.Error()}
	}

	if size > 10000 {
		// TODO: handle errors
		return bot.Text{Content: "too big"}
	}

	msg := make([]rune, size)
	for i := range msg {
		x := rune(i % 94)
		msg[i] = '!' + x
		if x == 93 {
			msg[i] = '\n'
		}
	}

	content := bot.Text{
		Header:     c.String("header"),
		Content:    string(msg),
		Footer:     c.String("footer"),
		Raw:        c.Bool("raw"),
		Quoted:     c.Bool("quoted"),
		QuoteType:  c.String("quote-type"),
		Pagination: c.Bool("pagination"),
		Pastebin:   c.Bool("paste"),
	}

	sleep := c.Duration("sleep")
	if sleep > 0 {
		time.Sleep(sleep)
	}

	delay := c.Duration("delay")
	if delay > 0 {
		return bot.Delayed(c.Context.Context, &bot.DelayedConfig{Name: "delayed waiting"}, func() bot.Content {
			time.Sleep(delay)
			return content
		})
	}

	return content
}

func testEcho(c *bot.CommandContext) bot.Content {
	content := bot.Text{
		Header:     c.String("header"),
		Content:    c.RawArg(0),
		Footer:     c.String("footer"),
		Raw:        c.Bool("raw"),
		Quoted:     c.Bool("quoted"),
		QuoteType:  c.String("quote-type"),
		Pagination: c.Bool("pagination"),
		Pastebin:   c.Bool("paste"),
	}

	sleep := c.Duration("sleep")
	if sleep > 0 {
		time.Sleep(sleep)
	}

	delay := c.Duration("delay")
	if delay > 0 {
		return bot.Delayed(c.Context.Context, &bot.DelayedConfig{Name: "delayed waiting"}, func() bot.Content {
			time.Sleep(delay)
			return content
		})
	}

	return content
}

func testEmbed() bot.Content {
	return nil
}

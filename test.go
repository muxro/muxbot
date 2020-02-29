package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mattn/go-shellwords"
	"github.com/urfave/cli/v2"
)

func testHandler(b *Bot, msg *discordgo.Message, args string) error {
	parser := shellwords.NewParser()
	parsed, err := parser.Parse(args)
	if err != nil {
		return err
	}

	params := append([]string{"$t"}, parsed...)

	var outBuf bytes.Buffer
	app := &cli.App{
		Version:     version,
		Description: "Chatbot with gitlab integration",
		Writer:      &outBuf,
		ErrWriter:   &outBuf,
		Metadata:    map[string]interface{}{"bot": b, "message": msg},
		Commands: []*cli.Command{
			&cli.Command{
				Name:        "g",
				Action:      wrapSimple(gHandler),
				Description: "`g` scrapes the first web result on dogpile.com (a search engine based on bing) for the desired query with the link and the description of the result. Even though it's not google, it returns on-topic results.",
				Usage:       "`.g <query>`",
			},
			&cli.Command{
				Name:        "yt",
				Action:      wrapSimple(ytHandler),
				Description: "`yt` acts like `g` but instead of scraping the first web result, it scrapes the first video result",
				Usage:       "`.yt <query>`",
			},
			&cli.Command{
				Name:        "gis",
				Action:      wrapSimple(gisHandler),
				Description: "`gis` acts like `g`, but instead of scraping the first web result, it scrapes the first image result",
				Usage:       "`.gis <query>`",
			},
			&cli.Command{
				Name:        "echo",
				Action:      wrapSimple(echoHandler),
				Description: "`echo` replies back with the text sent by the user. There aren't many use cases for it but it's a nice-to-have.",
				Usage:       "`.echo <text>`",
			},
			&cli.Command{
				Name:   "regex",
				Action: wrapSimple(regexCommandHandler),
			},
			&cli.Command{
				Name:        "ping",
				Action:      wrapSimple(pingHandler),
				Description: "`ping` replies to the user with `pong`, to test the latency between the user and the bot.",
			},
			&cli.Command{
				Name:        "issues",
				Description: "is a set of commands that revolve around gitlab issues in projects the bot is in (from the associated gitlab key entered when running). It is still a work in progress command and everything about it is subject to change. `activeRepo` changing is finished, but it isn't integrated in the `list` and `add` commands for now.",
				Subcommands: []*cli.Command{
					&cli.Command{
						Name:        "list",
						Description: "lists issues based on the parameters",
						Action:      wrapIssues(issueListHandler),
					},
					&cli.Command{
						Name:        "add",
						Description: "adds an issue with the title being the text coming after it.",
						Action:      wrapIssues(issueAddHandler),
					},
					&cli.Command{
						Name:        "close",
						Description: "closes a specified issue and returns an error if it couldn't close it",
						Action:      wrapIssues(issueCloseHandler),
					},
					&cli.Command{
						Name:        "active-repo",
						Description: "is a command used for setting the repository that the channel is working on.",
						Action:      wrapIssues(issuesActiveRepoHandler),
					},
					&cli.Command{
						Name:        "modify",
						Description: "updates an issue",
						Action:      wrapIssues(issueModifyHandler),
					},
				},
			},
			&cli.Command{
				Name:        "gitlab-key",
				Description: "associates a discord user with a personal access token and is used by `.issues` when `list`ing issues assigned to `$self` and `add`ing issues",
				Usage:       "`.gitlab-key <personal access token>`",
				Action:      wrapComplex(gitlabKeyHandler),
			},
			&cli.Command{
				Name:        "ghtrends",
				Description: "queries a GitHub trends API and returns 10 results in an embed",
				Usage:       "You can specify the date (`daily`, `weekly`, `monthly`) and the language (`js`, `c++`, etc) in any order, but only the last language will be counted if you include more than one. ",
				Action:      wrapComplex(ghTrends),
			},
			&cli.Command{
				Name:        "e",
				Description: "executes some code with a supported language. Visit [the evaluator's github repo]() to see all supported languages as this list is evolving",
				Usage:       "`.e <language> ```code``` [```standard input```]`",
				Action: func(c *cli.Context) error {
					if !evalDisabled {
						return wrapComplex(executeHandler)(c)
					}
					fmt.Fprint(c.App.Writer, "This command was disabled due to not having the required run options")
					return nil
				},
			},
			&cli.Command{
				Name:        "todo",
				Usage:       "Doesn't exist anymore :(",
				Description: "`todo` doesn't exist anymore, but I am leaving this here as an homage to it: my first big command I implemented. Thank you for your service <3",
				Action: wrapSimple(func(args []string) (string, error) {
					return "This command doesn't exist anymore :((((((", nil
				}),
			},
		},
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

func wrapSimple(command SimpleCommandHandler) cli.ActionFunc {
	return func(c *cli.Context) error {
		fmt.Println(c.Args())
		params := append(c.Args().Slice())
		ret, err := command(params)
		c.App.Metadata["bot"].(*Bot).SendReply(c.App.Metadata["message"].(*discordgo.Message), ret)
		return err
	}
}

func wrapComplex(command CommandHandler) cli.ActionFunc {
	return func(c *cli.Context) error {
		bot := c.App.Metadata["bot"].(*Bot)
		msg := c.App.Metadata["message"].(*discordgo.Message)
		args := strings.Join(c.Args().Slice(), " ")
		return command(bot, msg, args)
	}
}

func wrapIssues(command IssueCommandHandler) cli.ActionFunc {
	return func(c *cli.Context) error {
		bot := c.App.Metadata["bot"].(*Bot)
		msg := c.App.Metadata["message"].(*discordgo.Message)
		return command(bot, c.Args().Slice(), msg)
	}
}

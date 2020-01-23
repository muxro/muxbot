package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/russross/blackfriday/v2"
)

var (
	errUnsupportedLanguage = errors.New("language is not in the supported list")

	languages = map[string]int{
		"go":     1,
		"golang": 1,
	}
)

func executeHandler(bot *Bot, msg *discordgo.Message, key string) error {
	md := blackfriday.New()
	node := md.Parse([]byte(msg.Content))

	quotes := []string{}

	visit := func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		if node.Type == blackfriday.CodeBlock ||
			node.Type == blackfriday.BlockQuote ||
			node.Type == blackfriday.Code {
			quotes = append(quotes, string(node.Literal))
		}
		return 0
	}
	node.Walk(visit)

	var code, stdin string
	var lang int

	if len(quotes) > 2 {
		return errTooManyArgs
	}

	parts := strings.SplitN(msg.Content, " ", 3)
	lang = getSupportedLanguage(parts[1])
	if len(quotes) >= 1 {
		if lang == 0 {
			parts = strings.SplitN(quotes[0], "\n", 2)
			lang, code = getSupportedLanguage(parts[0]), parts[1]
			if lang == 0 {
				return errUnsupportedLanguage
			}
		} else {
			code = quotes[0]
		}
	}

	if len(quotes) == 2 {
		stdin = strings.Trim(quotes[1], " \n\t")
	}

	// fmt.Printf("%d %q %q\n", lang, code, stdin)
	bot.SendReply(msg, "running...")
	bot.SendReply(msg, "Output:\n```"+run(lang, code, stdin)+"```")

	return nil
}

func run(language int, code string, stdin string) string {
	return fmt.Sprintln("Not working yet ;)")
}

func getSupportedLanguage(language string) int {
	if ret, ok := languages[language]; ok {
		return ret
	}
	return 0
}

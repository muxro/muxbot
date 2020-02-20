package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/russross/blackfriday/v2"
)

var (
	errUnsupportedLanguage = errors.New("language is not in the supported list")
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
	var lang string

	if len(quotes) > 2 {
		return errTooManyArgs
	}

	idx := strings.IndexAny(msg.Content, " \n\t")
	if idx == -1 {
		return errInsufficientArgs
	}
	args := msg.Content[idx+1:]
	parts := strings.SplitN(args, " ", 2)
	if !strings.Contains(parts[0], "```") {
		lang = parts[0]
		if len(quotes) == 0 {
			code = strings.TrimSpace(parts[1])
		} else {
			code = strings.TrimSpace(quotes[0])
		}
	} else {
		if len(quotes) == 0 {
			return errInsufficientArgs
		}
		lines := strings.SplitN(quotes[0], "\n", 2)
		lang, code = lines[0], lines[1]
	}

	if len(quotes) == 2 {
		stdin = strings.TrimSpace(quotes[1])
	}
	// fmt.Printf("%d %q %q\n", lang, code, stdin)
	bot.SendReply(msg, "Output: \n```\nRunning...\n```")
	err := run(bot, msg, lang, code, stdin)

	return err
}

func run(bot *Bot, msg *discordgo.Message, language string, code string, stdin string) error {
	var output, errors []byte
	os.Mkdir("/tmp/"+msg.ID, 0777)

	codePath := "/tmp/" + msg.ID + "/code"
	err := ioutil.WriteFile(codePath, []byte(code), 0777)
	if err != nil {
		return err
	}
	stdinPath := "/tmp/" + msg.ID + "/stdin"
	err = ioutil.WriteFile(stdinPath, []byte(stdin), 0777)
	if err != nil {
		return err
	}

	cmd := exec.Command(*executeToken+"eval.sh", language, codePath, stdinPath)
	cmd.Dir = *executeToken
	out, err := cmd.CombinedOutput()
	if err != nil {
		bot.SendReply(msg, fmt.Sprintf("%s\n%s", string(out), err))
		return nil
	}
	errors, err = ioutil.ReadFile(*executeToken + "result/build-error")
	if len(errors) > 2 {
		bot.SendReply(msg, "compile error: "+string(errors))
		return nil
	}
	output, err = ioutil.ReadFile(*executeToken + "result/out")
	if err != nil {
		return err
	}
	_, err = bot.SendReply(msg, "Output:\n```"+language+"\n"+string(output)+"```")
	return err
}

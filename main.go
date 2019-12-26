package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Knetic/govaluate"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

var token = flag.String("token", "none", "Specify the token")
var googleDevKey = flag.String("gkey", "none", "Specify the google dev key")
var gitlabToken = flag.String("glt", "none", "Specify the Gitlab Token")
var prefix = flag.String("prefix", ".", "Specify the bot prefix")

func main() {
	flag.Parse()

	envVal, exists := os.LookupEnv("token")
	if exists {
		*token = envVal
	}

	envVal, exists = os.LookupEnv("gkey")
	if exists {
		*googleDevKey = envVal
	}

	envVal, exists = os.LookupEnv("glt")
	if exists {
		*gitlabToken = envVal
	}

	if *token == "none" || *googleDevKey == "none" || *gitlabToken == "none" {
		fmt.Println("You need to specify a token, google dev and gitlab key")
		return
	}

	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		fmt.Println("Could not create Discord session: ", err)
		return
	}

	err = initDB()
	if err != nil {
		log.Fatal(err)
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	fmt.Println("Running MuxBot. Press Ctrl+C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "MuxBot is Aliiive!")
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	sendMessage := func(sentMsg string) *discordgo.Message {
		resulting, _ := session.ChannelMessageSend(message.ChannelID, sentMsg)
		return resulting
	}

	sendReply := func(sentMsg string) *discordgo.Message {
		return sendMessage(fmt.Sprintf("(%s) %s", strings.Split(message.Author.String(), "#")[0], sentMsg))
	}

	sendError := func(err error) *discordgo.Message {
		return sendReply(fmt.Sprintf("Eroare: %v", err))
	}

	// It is a command
	if strings.HasPrefix(message.Content, *prefix) {
		commandMessage := strings.Join(strings.Split(message.Content, " ")[1:], " ")
		if startsCommand(message.Content, "help") {
			sendReply("Head over to https://gitlab.com/muxro/muxbot/blob/master/commands.md for information regarding available commands.")

		} else if startsCommand(message.Content, "g ") {
			res, err := scrapeFirstWebRes(commandMessage)
			if err != nil {
				sendError(err)
				return
			}
			sendReply(fmt.Sprintf("%s -- %s", res["url"], res["desc"]))

		} else if startsCommand(message.Content, "gis ") {
			res, err := scrapeFirstImgRes(commandMessage)
			if err != nil {
				sendError(err)
				return
			}
			sendReply(res)

		} else if startsCommand(message.Content, "yt") {
			res, err := getFirstYTResult(commandMessage)
			if err != nil {
				sendError(err)
				return
			}
			sendReply(res)

		} else if startsCommand(message.Content, "ping") {
			sendReply("pong")

		} else if startsCommand(message.Content, "echo") {
			if commandMessage != "" {
				sendReply(commandMessage)
			}

		} else if startsCommand(message.Content, "eval") {
			expr, err := govaluate.NewEvaluableExpression(commandMessage)
			if err != nil {
				sendError(err)
				return
			}
			result, err := expr.Evaluate(nil)
			if err != nil {

				sendError(err)
				return
			}
			sendReply(fmt.Sprintf("%v", result))

		} else if startsCommand(message.Content, "todo") {
			handleTodo(session, message.Message)

		} else if startsCommand(message.Content, "issues") {
			handleIssue(session, message.Message)

		} else if startsCommand(message.Content, "glkey") {
			key := commandMessage
			err := session.ChannelMessageDelete(message.ChannelID, message.ID)
			if err != nil {
				sendError(err)
				return
			}
			result, ok := testKey(key)
			if ok == true {
				err := associateUserToToken(message.Author.ID, key)

				if err != nil {
					sendError(err)
					return
				}
				sendReply("Associated user with gitlab user " + result.Name)
			} else {
				sendReply("Invalid key")
			}

		}
	}

}

func startsCommand(content string, command string) bool {
	return strings.HasPrefix(content, fmt.Sprintf("%s%s", *prefix, command))
}

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

type messageSender func(message string) *discordgo.Message
type errorSender func(err error) *discordgo.Message
type cmd func(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender)

var (
	token        = flag.String("token", "none", "Specify the token")
	googleDevKey = flag.String("gkey", "none", "Specify the google dev key")
	gitlabToken  = flag.String("glt", "none", "Specify the Gitlab Token")

	prefix   = flag.String("prefix", ".", "Specify the bot prefix")
	commands = make(map[string]cmd)
)

func main() {
	flag.Parse()

	getEnvVar("token", &token)
	getEnvVar("gkey", &googleDevKey)
	getEnvVar("glt", &gitlabToken)

	if *token == "none" {
		panic("No discord token specified, can't run the bot without it.")
	}

	if *googleDevKey == "none" {
		fmt.Println("No google dev key specified, `yt` command will be disabled.")
	}

	if *gitlabToken == "none" {
		fmt.Println("No gitlab token specified, the `issues` and `glkey` commands will be disabled.")
	}

	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		fmt.Println("Could not create Discord session: ", err)
		return
	}

	err = initDB()
	if err != nil {
		panic(err)
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	// register commands
	registerCommand("help", helpHandler)
	registerCommand("ping", pingHandler)
	registerCommand("echo", echoHandler)
	registerCommand("eval", evalHandler)
	registerCommand("g", gHandler)
	registerCommand("gis", gisHandler)
	if *googleDevKey != "none" {
		registerCommand("yt", ytHandler)
	} else {
		registerCommand("yt", nonExistentHandler)
	}
	registerCommand("todo", todoHandler)
	if *gitlabToken != "none" {
		registerCommand("issues", issueHandler)
		registerCommand("glkey", glKeyHandler)
	} else {
		registerCommand("issues", nonExistentHandler)
		registerCommand("glkey", nonExistentHandler)
	}

	err = dg.Open()
	if err != nil {
		panic(err)
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

	sendMessage, sendReply, sendError := initMessageSenders(session, message)

	// It is a command
	if strings.HasPrefix(message.Content, *prefix) {
		command := strings.Split(strings.TrimPrefix(message.Content, *prefix), " ")[0]
		if cmdHandler, ok := commands[command]; ok {
			cmdHandler(session, message, sendReply, sendMessage, sendError)
		}
	}
}

func registerCommand(name string, handler cmd) {
	commands[name] = handler
}

func initMessageSenders(session *discordgo.Session, message *discordgo.MessageCreate) (sendMessage messageSender, sendReply messageSender, sendError errorSender) {
	sendMessage = func(sentMsg string) *discordgo.Message {
		resulting, _ := session.ChannelMessageSend(message.ChannelID, sentMsg)
		return resulting
	}

	sendReply = func(sentMsg string) *discordgo.Message {
		return sendMessage(fmt.Sprintf("(%s) %s", strings.Split(message.Author.String(), "#")[0], sentMsg))
	}

	sendError = func(err error) *discordgo.Message {
		return sendReply(fmt.Sprintf("Eroare: %v", err))
	}

	return
}

func getEnvVar(name string, variable **string) {
	envVal, exists := os.LookupEnv(name)
	if exists {
		**variable = envVal
	}
}

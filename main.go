package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
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
	db       *sql.DB
)

func main() {
	flag.Parse()

	getEnvVar("token", token)
	getEnvVar("gkey", googleDevKey)
	getEnvVar("glt", gitlabToken)

	if *token == "none" {
		log.Fatal("No discord token specified, can't run the bot without it.")
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

	db, err = sql.Open("sqlite3", "database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS gitlabKeys (dtag varchar(512) UNIQUE, key varchar(512));")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS activeRepo (dtag varchar(512) UNIQUE, repo varchar(512));")
	if err != nil {
		log.Fatal(err)
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	// register commands
	commands["help"] = helpHandler
	commands["ping"] = pingHandler
	commands["echo"] = echoHandler
	commands["eval"] = evalHandler
	commands["g"] = gHandler
	commands["gis"] = gisHandler
	if *googleDevKey != "none" {
		commands["yt"] = ytHandler
	} else {
		commands["yt"] = nonExistentHandler
	}
	commands["todo"] = todoHandler
	if *gitlabToken != "none" {
		commands["issues"] = issueHandler
		commands["glkey"] = glKeyHandler
	} else {
		commands["issues"] = nonExistentHandler
		commands["glkey"] = nonExistentHandler
	}

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Running MuxBot. Press Ctrl+C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "MuxBot is Aliive! Use "+*prefix+"help for a list of commands")
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	sendMessage, sendReply, sendError := initMessageSenders(session, message)

	defer func() {
		if r := recover(); r != nil {
			sendReply("something went wrong...")
			fmt.Println(r, string(debug.Stack()))
		}
	}()

	// It is a command
	if strings.HasPrefix(message.Content, *prefix) {
		command := strings.Split(strings.TrimPrefix(message.Content, *prefix), " ")[0]
		if cmdHandler, ok := commands[command]; ok {
			cmdHandler(session, message, sendReply, sendMessage, sendError)
		}
	}
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
		return sendReply(fmt.Sprintf("Error: %v", err))
	}

	return
}

func getEnvVar(name string, variable *string) {
	envVal, exists := os.LookupEnv(name)
	if exists {
		*variable = envVal
	}
}

func getArguments(message *discordgo.MessageCreate) []string {
	return strings.Split(message.Content, " ")[1:]
}

func getText(message *discordgo.MessageCreate) string {
	return strings.Join(getArguments(message), " ")
}

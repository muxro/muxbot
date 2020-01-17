package main

import (
	"database/sql"
	"errors"
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
	"github.com/xanzy/go-gitlab"
)

var (
	// tokens
	token        = flag.String("token", "none", "Specify the token")
	googleDevKey = flag.String("gkey", "none", "Specify the google dev key")
	gitlabToken  = flag.String("glt", "none", "Specify the Gitlab Token")

	// functional stuff
	prefix = flag.String("prefix", ".", "Specify the bot prefix")
	db     *sql.DB
)

// MessageSender is an old relic from todos, but I won't touch it yet
type MessageSender func(message string) *discordgo.Message

// CommandHandler is a type for all commands, called by the bot
type CommandHandler func(bot *Bot, message *discordgo.Message, args string) error

// SimpleCommandHandler is the handler for all commands
type SimpleCommandHandler func(args []string) (string, error)

// IssueCommandHandler is the handler for issue commands
type IssueCommandHandler func(bot *Bot, git *gitlab.Client, projects []*gitlab.Project, args []string, msg *discordgo.Message) error

// CommandMux is an abstraction of the commands and subcommands, to simplify stuff
type CommandMux struct {
	cmds map[string]CommandHandler
}

// Bot stores the session data about the bot currently running
type Bot struct {
	ds   *discordgo.Session
	db   *sql.DB
	cmds *CommandMux
}

// NewCommandMux creates a new command muxer instance
func NewCommandMux() *CommandMux {
	return &CommandMux{
		cmds: map[string]CommandHandler{},
	}
}

// Handle gets the params of the command and then runs it
func (cm *CommandMux) Handle(bot *Bot, msg *discordgo.Message) error {
	parts := strings.SplitN(msg.Content, " ", 2)
	cmd, args := parts[0], ""
	if len(parts) > 1 {
		args = parts[1]
	}

	handler, ok := cm.cmds[cmd]
	if !ok {
		return fmt.Errorf("command %q not found", cmd)
	}

	return handler(bot, msg, args)
}

// SimpleCommand registers a wrapper for the simple commands that don't need much data about their environment
func (cm *CommandMux) SimpleCommand(name string, handler SimpleCommandHandler) {
	cm.cmds[name] = func(bot *Bot, msg *discordgo.Message, args string) error {
		parts := strings.Fields(args)

		resp, err := handler(parts)
		if err != nil {
			return err
		}

		if resp == "" {
			return fmt.Errorf("command %q returned empty response", name+" "+args)
		}

		bot.SendReply(msg, resp)

		return nil
	}
}

// IssueCommand wraps the git and projects params to the mux handler
func (cm *CommandMux) IssueCommand(name string, handler IssueCommandHandler, git *gitlab.Client, projects []*gitlab.Project) {
	cm.cmds[name] = func(bot *Bot, msg *discordgo.Message, args string) error {
		parts := strings.Fields(args)
		return handler(bot, git, projects, parts, msg)
	}
}

// Command registers the more complex commands that need the whole environment to properly function
func (cm *CommandMux) Command(name string, handler CommandHandler) {
	cm.cmds[name] = handler
}

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

	err := errors.New("I need this")
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
	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		fmt.Println("Could not create Discord session: ", err)
		return
	}

	bot := Bot{ds: dg,
		db:   db,
		cmds: NewCommandMux(),
	}
	bot.registerDiscordHandlers()

	// register commands
	bot.cmds.SimpleCommand("help", helpHandler)
	bot.cmds.SimpleCommand("ping", pingHandler)
	bot.cmds.SimpleCommand("echo", echoHandler)
	bot.cmds.SimpleCommand("eval", evalHandler)
	bot.cmds.SimpleCommand("g", gHandler)
	bot.cmds.SimpleCommand("gis", gisHandler)
	if *googleDevKey != "none" {
		bot.cmds.SimpleCommand("yt", ytHandler)
	} else {
		bot.cmds.SimpleCommand("yt", nonExistentHandler)
	}
	// bot.RegisterCommand("todo", todoHandler)
	if *gitlabToken != "none" {
		bot.cmds.Command("issues", issueHandler)
		bot.cmds.Command("gitlab-key", gitlabKeyHandler)
	} else {
		bot.cmds.SimpleCommand("issues", nonExistentHandler)
		bot.cmds.SimpleCommand("gitlab-key", nonExistentHandler)
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

func (b *Bot) registerDiscordHandlers() {
	b.ds.AddHandler(b.ready)
	b.ds.AddHandler(b.messageCreate)
}

func (b *Bot) ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "I'm alive! Use "+*prefix+"help for a list of commands")
}

// SendMessage sends a message to discord
func (b *Bot) SendMessage(channel string, msg string) (*discordgo.Message, error) {
	// TODO: retry if error
	sent, err := b.ds.ChannelMessageSend(channel, msg)
	if err != nil {
		return nil, err
	}
	return sent, nil
}

// SendReply sends a message with the username in parentheses at the start
func (b *Bot) SendReply(msg *discordgo.Message, reply string) (*discordgo.Message, error) {
	author := strings.Split(msg.Author.String(), "#")[0]
	return b.SendMessage(msg.ChannelID, fmt.Sprintf("(%s) %s", author, reply))
}

func (b *Bot) messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			b.SendReply(message.Message, "something went wrong...")
			fmt.Println(r, string(debug.Stack()))
		}
	}()

	if !strings.HasPrefix(message.Content, *prefix) {
		return
	}
	message.Content = strings.TrimPrefix(message.Content, *prefix)

	err := b.cmds.Handle(b, message.Message)
	if err != nil {
		b.SendMessage(message.ChannelID, fmt.Sprintln("error: ", err))
	}
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

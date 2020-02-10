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
)

var (
	// tokens
	token        = flag.String("token", "none", "Specify the token")
	googleDevKey = flag.String("gkey", "none", "Specify the google dev key")
	gitlabToken  = flag.String("glt", "none", "Specify the Gitlab Token")
	executeToken = flag.String("exToken", "none", "Specify the absolute path to the eval directory")

	// errors
	errInvalidCommand = errors.New("invalid command")
	errTooManyArgs    = errors.New("too many arguments")

	// functional stuff
	prefix = flag.String("prefix", ".", "Specify the bot prefix")
	db     *sql.DB
)

// MessageHandler is the base handler for commands
type MessageHandler func(bot *Bot, message *discordgo.Message) (bool, error)

// CommandHandler is a type for all commands, called by the bot
type CommandHandler func(bot *Bot, message *discordgo.Message, args string) error

// SimpleCommandHandler is the handler for all commands
type SimpleCommandHandler func(args []string) (string, error)

// IssueCommandHandler is the handler for issue commands
type IssueCommandHandler func(bot *Bot, args []string, msg *discordgo.Message) error

// CommandMux is an abstraction of the commands and subcommands, to simplify stuff
type CommandMux struct {
	cmds map[string]CommandHandler
}

// Bot stores the session data about the bot currently running
type Bot struct {
	ds       *discordgo.Session
	db       *sql.DB
	msgHist  *MessageHistory
	handlers []MessageHandler
}

// MessageHistory is stores the last max messages to update if the original message is edited
type MessageHistory struct {
	no   int
	msgs [][]*discordgo.Message
}

// NewMessageHistory creates a MessageHistory instance with `max` possible slots
func NewMessageHistory(max int) *MessageHistory {
	return &MessageHistory{
		msgs: make([][]*discordgo.Message, max, max),
	}
}

// NewCommandMux creates a new command muxer instance
func NewCommandMux() *CommandMux {
	return &CommandMux{
		cmds: map[string]CommandHandler{},
	}
}

// Add an element to the history "cache" with rollover
func (mh *MessageHistory) Add(msg, reply *discordgo.Message) {
	mh.msgs[mh.no] = []*discordgo.Message{msg, reply}
	mh.no = (mh.no + 1) % len(mh.msgs)
}

// Handle gets the params of the command and then runs it
func (cm *CommandMux) Handle(bot *Bot, msg *discordgo.Message) error {
	content := msg.Content
	content = strings.TrimPrefix(content, *prefix)
	idx := strings.IndexAny(content, " \n\t")
	var cmd, args string
	if idx == -1 {
		cmd = content
	} else {
		cmd, args = content[:idx], content[idx:]
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
func (cm *CommandMux) IssueCommand(name string, handler IssueCommandHandler) {
	cm.cmds[name] = func(bot *Bot, msg *discordgo.Message, args string) error {
		parts := strings.Fields(args)
		return handler(bot, parts, msg)
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
	getEnvVar("exToken", executeToken)

	if *token == "none" {
		log.Fatal("No discord token specified, can't run the bot without it.")
	}

	if *googleDevKey == "none" {
		log.Println("No google dev key specified, `yt` command will be disabled.")
	}

	if *gitlabToken == "none" {
		log.Println("No gitlab token specified, the `issues` and `glkey` commands will be disabled.")
	}

	if *executeToken == "none" {
		log.Println("No execute path specified, the `e` command will be disabled")
	}

	var err error
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
		log.Fatalln("Could not create Discord session: ", err)
	}

	bot := Bot{ds: dg,
		db:      db,
		msgHist: NewMessageHistory(100),
	}
	bot.registerDiscordHandlers()

	cmds := NewCommandMux()
	// register commands
	if *executeToken != "none" {
		cmds.Command("e", executeHandler)
	} else {
		cmds.SimpleCommand("e", nonExistentHandler)
	}
	cmds.Command("ghtrends", ghTrends)
	cmds.SimpleCommand("help", helpHandler)
	cmds.SimpleCommand("ping", pingHandler)
	cmds.SimpleCommand("echo", echoHandler)
	cmds.SimpleCommand("g", gHandler)
	cmds.SimpleCommand("gis", gisHandler)
	if *googleDevKey != "none" {
		cmds.SimpleCommand("yt", ytHandler)
	} else {
		cmds.SimpleCommand("yt", nonExistentHandler)
	}
	var issues *Issues
	if *gitlabToken != "none" {
		issues = NewIssues(*gitlabToken)
		cmds.Command("issues", issues.issueHandler)
		cmds.Command("gitlab-key", issues.gitlabKeyHandler)
	} else {
		cmds.SimpleCommand("issues", nonExistentHandler)
		cmds.SimpleCommand("gitlab-key", nonExistentHandler)
	}
	cmds.SimpleCommand("regex", regexCommandHandler)
	bot.AddMessageHandler(CommandMessageHandler(cmds))
	if *gitlabToken != "none" {
		bot.AddMessageHandler(issues.issueReferenceHandler)
	}
	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Running MuxBot. Press Ctrl+C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

}

func (b *Bot) registerDiscordHandlers() {
	b.ds.AddHandler(b.onReady)
	b.ds.AddHandler(b.onMessageCreate)
	b.ds.AddHandler(b.onMessageEdit)
}

func (b *Bot) onMessage(message *discordgo.Message) {
	if message.Author == nil || message.Author.ID == b.ds.State.User.ID {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			b.SendReply(message, "something went wrong...")
			fmt.Println(r, string(debug.Stack()))
		}
	}()

	for _, handler := range b.handlers {
		handled, err := handler(b, message)
		if err != nil {
			b.SendReply(message, fmt.Sprintln("error:", err))
		}
		if handled {
			break
		}
	}
}

func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "I'm alive! Use "+*prefix+"help for a list of commands")
}

// SendReplyComplex is the base for SendReply and SendReplyEmbed
func (b *Bot) SendReplyComplex(msg *discordgo.Message, data *discordgo.MessageSend) (*discordgo.Message, error) {
	var existing *discordgo.Message
	for _, pair := range b.msgHist.msgs {
		if pair != nil && pair[0].ID == msg.ID {
			existing = pair[1]
			break
		}
	}

	if data.Content != "" {
		author := strings.Split(msg.Author.String(), "#")[0]
		data.Content = fmt.Sprintf("(%s) %s", author, data.Content)
		data.Embed = nil
	}
	var replyMsg *discordgo.Message
	var err error
	if existing != nil {
		replyMsg, err = b.ds.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID: existing.ID, Channel: existing.ChannelID,
			Content: &data.Content,
			Embed:   data.Embed,
		})
	} else {
		replyMsg, err = b.ds.ChannelMessageSendComplex(msg.ChannelID, data)
	}
	if err != nil {
		return nil, err
	}
	b.msgHist.Add(msg, replyMsg)
	return replyMsg, err
}

// SendReply sends a message with the username in parentheses at the start
func (b *Bot) SendReply(msg *discordgo.Message, reply string) (*discordgo.Message, error) {
	return b.SendReplyComplex(msg, &discordgo.MessageSend{Content: reply})
}

// SendReplyEmbed sends an embed
func (b *Bot) SendReplyEmbed(msg *discordgo.Message, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	return b.SendReplyComplex(msg, &discordgo.MessageSend{Embed: embed})
}

func (b *Bot) onMessageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	b.onMessage(message.Message)
}

func (b *Bot) onMessageEdit(session *discordgo.Session, message *discordgo.MessageUpdate) {
	b.onMessage(message.Message)
}

// AddMessageHandler adds a new handler to the bot
func (b *Bot) AddMessageHandler(handler MessageHandler) {
	b.handlers = append(b.handlers, handler)
}

func getEnvVar(name string, variable *string) {
	envVal, exists := os.LookupEnv(name)
	if exists {
		*variable = envVal
	}
}

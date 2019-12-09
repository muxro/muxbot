package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Knetic/govaluate"

	"github.com/bwmarrin/discordgo"
)

func main() {
	tokenPtr := flag.String("token", "none", "Specify the token")
	flag.Parse()
	token := *tokenPtr

	if token == "none" {
		fmt.Println("You need to specify a token")
		return
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Could not create Discord session: ", err)
		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Could not open Discord session: ", err)
		return
	}

	fmt.Println("Running MuxBot. Press Ctrl+C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "MuxBot is Aliiive!")
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {

	if message.Author.ID == session.State.User.ID {
		return
	}

	// It is a command
	if strings.HasPrefix(message.Content, ".") {
		commandMessage := strings.Join(strings.Split(message.Content, " ")[1:], " ")

		if strings.HasPrefix(message.Content, ".help") {
			sendMessageInChannel(`
.help - shows this
.ping - pong
.echo - echoes back whatever you send it
.calc - compute simple expression
			`, session, message.ChannelID)
		} else if strings.HasPrefix(message.Content, ".ping") {
			sendMessageInChannel("pong", session, message.ChannelID)
		} else if strings.HasPrefix(message.Content, ".echo") {

			if commandMessage != "" {
				sendMessageInChannel(commandMessage, session, message.ChannelID)
			}
		} else if strings.HasPrefix(message.Content, ".calc") {
			expr, err := govaluate.NewEvaluableExpression(commandMessage)
			if err != nil {
				sendMessageInChannel(fmt.Sprintf("Nu am putut calcula: Eroare %v", err), session, message.ChannelID)
				return
			}
			result, err := expr.Evaluate(nil)
			if err != nil {
				sendMessageInChannel(fmt.Sprintf("Nu am putut calcula, eroare %v", err), session, message.ChannelID)
			}
			sendMessageInChannel(fmt.Sprintf("%v", result), session, message.ChannelID)
		}
	}

}

func sendEmbedInChannel(embed *discordgo.MessageEmbed, session *discordgo.Session, channel string) {
	_, err := session.ChannelMessageSendEmbed(channel, embed)
	if err != nil {
		fmt.Println(err)
		return
	}

}

func sendMessageInChannel(message string, session *discordgo.Session, channel string) {
	session.ChannelMessageSend(channel, message)
}

func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

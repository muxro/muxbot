package main

import (
	"fmt"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/bwmarrin/discordgo"
)

func helpHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	sendReply("Head over to https://gitlab.com/muxro/muxbot/blob/master/commands.md for information regarding available commands.")
}

func pingHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	sendReply("pong")
}

func echoHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	commandMessage := strings.Join(strings.Split(message.Content, " ")[1:], " ")
	if commandMessage != "" {
		sendReply(commandMessage)
	}
}

func evalHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	commandMessage := strings.Join(strings.Split(message.Content, " ")[1:], " ")
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
}

func gHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	commandMessage := strings.Join(strings.Split(message.Content, " ")[1:], " ")
	res, err := scrapeFirstWebRes(commandMessage)
	if err != nil {
		sendError(err)
		return
	}
	sendReply(fmt.Sprintf("%s -- %s", res["url"], res["desc"]))
}

func gisHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	commandMessage := strings.Join(strings.Split(message.Content, " ")[1:], " ")
	res, err := scrapeFirstImgRes(commandMessage)
	if err != nil {
		sendError(err)
		return
	}
	sendReply(res)
}

func ytHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	commandMessage := strings.Join(strings.Split(message.Content, " ")[1:], " ")
	res, err := getFirstYTResult(commandMessage)
	if err != nil {
		sendError(err)
		return
	}
	sendReply(res)
}

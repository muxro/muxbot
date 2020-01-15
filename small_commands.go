package main

import (
	"fmt"

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
	commandMessage := getText(message)
	if commandMessage != "" {
		sendReply(commandMessage)
	}
}

func evalHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	commandMessage := getText(message)
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
	commandMessage := getText(message)
	res, err := scrapeFirstWebRes(commandMessage)
	if err != nil {
		sendError(err)
		return
	}
	sendReply(fmt.Sprintf("%s -- %s", res["url"], res["desc"]))
}

func gisHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	commandMessage := getText(message)
	res, err := scrapeFirstImgRes(commandMessage)
	if err != nil {
		sendError(err)
		return
	}
	sendReply(res)
}

func ytHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	commandMessage := getText(message)
	res, err := getFirstYTResult(commandMessage)
	if err != nil {
		sendError(err)
		return
	}
	sendReply(res)
}

func nonExistentHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	sendReply("This command has been disabled because the bot maintainer didn't specify the required key")
}

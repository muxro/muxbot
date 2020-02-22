package main

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// CommandMessageHandler is a wrapper for command handling
func CommandMessageHandler(cmds *CommandMux) MessageHandler {
	return func(b *Bot, message *discordgo.Message) (bool, error) {
		if !strings.HasPrefix(message.Content, *prefix) {
			return false, nil
		}

		return true, cmds.Handle(b, message)
	}
}

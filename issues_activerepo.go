package main

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (i *Issues) issuesActiveRepoHandler(bot *Bot, args []string, msg *discordgo.Message) error {
	if len(args) < 1 {
		return errInsufficientArgs
	}
	switch args[0] {
	case "set":
		if len(args) != 2 {
			return errInsufficientArgs
		}
		if i.isRepo(args[1]) == false {
			return errNoRepoFound
		}
		if strings.ContainsAny(args[1], "/") == false { // we would also like a group name
			rawRepo, err := i.getGitlabRepo(args[1], msg)
			if err != nil {
				return err
			}
			namespace, repo := i.getFullRepoName(rawRepo)
			args[1] = namespace + "/" + repo
		}
		err := setActiveRepo(msg.ChannelID, args[1])
		if err != nil {
			return err
		}

		bot.SendReply(msg, "set active repo "+args[1])
	case "get":
		repo, exists := getActiveRepo(msg.ChannelID)
		if exists == false {
			return errNoActiveRepo
		}
		bot.SendReply(msg, "the channel's active repo is "+repo)
	case "erase":
		err := removeActiveRepo(msg.ChannelID)
		if err != nil {
			return err
		}

		bot.SendReply(msg, "your active repo has been erased from the database")
	default:
		return errInvalidCommand
	}
	return nil
}

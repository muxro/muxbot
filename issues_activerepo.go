package main

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

func issuesActiveRepoHandler(bot *Bot, args []string, msg *discordgo.Message) error {
	git, err := getUserGit(msg)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return errInsufficientArgs
	}
	switch args[0] {
	case "set":
		if len(args) != 2 {
			return errInsufficientArgs
		}
		if isRepo(git, args[1]) == false {
			return errNoRepoFound
		}
		if strings.ContainsAny(args[1], "/") == false { // we would also like a group name
			rawRepo, err := getGitlabRepo(git, args[1], msg)
			if err != nil {
				return err
			}
			namespace, repo := getFullRepoName(rawRepo)
			args[1] = namespace + "/" + repo
		}
		err := setActiveRepo(msg.ChannelID, args[1])
		if err != nil {
			return err
		}

		_, err = bot.SendReply(msg, "set active repo "+args[1])
		return err
	case "get":
		repo, exists := getActiveRepo(msg.ChannelID)
		if exists == false {
			return errNoActiveRepo
		}
		_, err := bot.SendReply(msg, "the channel's active repo is "+repo)
		return err
	case "erase":
		err := removeActiveRepo(msg.ChannelID)
		if err != nil {
			return err
		}

		_, err = bot.SendReply(msg, "the channel's active repo has been erased from the database")
		return err
	default:
		return errInvalidCommand
	}
}

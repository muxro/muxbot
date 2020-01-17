package main

import (
	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

func issuesActiveRepoHandler(bot *Bot, git *gitlab.Client, projects []*gitlab.Project, args []string, msg *discordgo.Message) error {
	if len(args) < 1 {
		return errors.New("not parameters specified")
	}
	switch args[0] {
	case "set":
		if len(args) != 2 {
			return errors.New("not parameters specified")
		}
		if isRepo(args[1], projects) == false {
			return errors.New("you need to specify a valid repo which you are a member of that the bot can see")
		}
		if strings.ContainsAny(args[1], "/") == false { // we would also like a group name
			for _, project := range projects {
				if isSameRepo(args[1], project) {
					args[1] = project.Namespace.Path + "/" + project.Path
					break
				}
			}
		}
		err := setActiveRepo(msg.Author.ID, args[1])
		if err != nil {
			return err
		}

		bot.SendReply(msg, "set active repo "+args[1])
	case "get":
		repo, exists := getActiveRepo(msg.Author.ID)
		if exists == false {
			return errors.New("no active repo set")
		}
		bot.SendReply(msg, "your active repo is "+repo)
	case "erase":
		err := removeActiveRepo(msg.Author.ID)
		if err != nil {
			return err
		}

		bot.SendReply(msg, "your active repo has been erased from the database")
	default:
		bot.SendReply(msg, "invalid command")
	}
	return nil
}

package main

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

func issueCloseHandler(bot *Bot, projects []*gitlab.Project, args []string, msg *discordgo.Message) error {
	asTok, exists := associatedKey(msg.Author.ID)
	if exists == false {
		return errors.New("you don't have a gitlab Personal Access Token associated with your account")
	}
	if len(args) != 1 {
		return errors.New("not enough parameters")
	}

	userGit := gitlab.NewClient(nil, asTok)

	id, repo, err := parseIssueParam(args[0])
	if err != nil {
		return err
	}

	if repo == "" {
		activeRepo, exists := getActiveRepo(msg.Author.ID)
		if exists {
			repo = activeRepo
		} else {
			return errors.New("you need to specify either an active repo or a repo to search in")
		}
	}
	pid := getRepo(repo, projects).ID

	gitIssue, _, err := userGit.Issues.UpdateIssue(pid, id, &gitlab.UpdateIssueOptions{StateEvent: gitlab.String("close")})
	if err != nil {
		return err
	}

	bot.SendReply(msg, fmt.Sprintf("Closed <%s>", gitIssue.WebURL))
	return nil
}

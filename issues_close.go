package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

func issueCloseHandler(bot *Bot, args []string, msg *discordgo.Message) error {
	git, err := getUserGit(msg)
	if err != nil {
		return err
	}
	if len(args) != 1 {
		return errInsufficientArgs
	}

	id, repo, err := parseIssueParam(args[0])
	if err != nil {
		return err
	}

	repo = getRepo(git, msg, repo)
	pid, err := getRepoID(git, repo, msg)
	if err != nil {
		return err
	}

	gitIssue, _, err := git.Issues.UpdateIssue(pid, id, &gitlab.UpdateIssueOptions{StateEvent: gitlab.String("close")})
	if err != nil {
		return err
	}

	_, err = bot.SendReply(msg, fmt.Sprintf("Closed <%s>", gitIssue.WebURL))
	return err
}

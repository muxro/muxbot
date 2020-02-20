package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

func (i *Issues) issueCloseHandler(bot *Bot, args []string, msg *discordgo.Message) error {
	asTok, exists := associatedKey(msg.Author.ID)
	if exists == false {
		return errNoPAC
	}
	if len(args) != 1 {
		return errInsufficientArgs
	}

	userGit := gitlab.NewClient(nil, asTok)

	id, repo, err := parseIssueParam(args[0])
	if err != nil {
		return err
	}

	repo = i.getRepo(msg, repo)
	pid, err := i.getRepoID(repo, msg)
	if err != nil {
		return err
	}

	gitIssue, _, err := userGit.Issues.UpdateIssue(pid, id, &gitlab.UpdateIssueOptions{StateEvent: gitlab.String("close")})
	if err != nil {
		return err
	}

	_, err = bot.SendReply(msg, fmt.Sprintf("Closed <%s>", gitIssue.WebURL))
	return err
}

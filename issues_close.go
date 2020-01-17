package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

func issueCloseHandler(bot *Bot, git *gitlab.Client, projects []*gitlab.Project, args []string, msg *discordgo.Message) error {
	asTok, exists := associatedKey(msg.Author.ID)
	if exists == false {
		return errors.New("you don't have a gitlab Personal Access Token associated with your account")
	}
	if len(args) != 1 {
		return errors.New("not enough parameters")
	}

	userGit := gitlab.NewClient(nil, asTok)

	var id int
	var repo string

	issue := args[0]
	if len(strings.Split(issue, "#")) == 2 {
		split := strings.Split(issue, "#")
		ID, err := strconv.Atoi(split[1])
		if err != nil {
			return errors.New("invalid ID")
		}

		id = ID
		repo = split[0]
	} else {
		if issue[0] >= '0' && issue[0] <= '9' {
			ID, err := strconv.Atoi(issue)
			if err != nil {
				return errors.New("invalid ID")
			}

			id = ID
		} else {
			return errors.New("invalid ID")
		}
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

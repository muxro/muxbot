package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

func issueCloseHandler(git *gitlab.Client, projects []*gitlab.Project, session *discordgo.Session, message *discordgo.MessageCreate) {
	params := strings.Split(message.Content, " ")[1:]
	_, sendReply, sendError := initMessageSenders(session, message)

	asTok, exists := associatedKey(message.Author.ID)
	if exists == false {
		sendReply("Error: You don't have a gitlab Personal Access Token associated with your account")
		return
	}
	if len(params) != 2 {
		sendReply("Usage: " + *prefix + "issues close issueid")
		return
	}

	userGit := gitlab.NewClient(nil, asTok)

	var id int
	var repo string

	issue := params[1]
	if len(strings.Split(issue, "#")) == 2 {
		split := strings.Split(issue, "#")
		ID, err := strconv.Atoi(split[1])
		if err != nil {
			sendReply("Invalid id")
			return
		}
		id = ID
		repo = split[0]
	} else {
		if issue[0] >= '0' && issue[0] <= '9' {
			ID, err := strconv.Atoi(issue)
			if err != nil {
				sendReply("Invalid id")
				return
			}
			id = ID
		} else {
			sendReply("Invalid id")
			return
		}
	}

	if repo == "" {
		activeRepo, exists := getActiveRepo(message.Author.ID)
		if exists {
			repo = activeRepo
		} else {
			sendReply("You need to specify either an active repo or a repo to search in")
		}
	}
	pid := getRepo(repo, projects).ID

	gitIssue, _, err := userGit.Issues.UpdateIssue(pid, id, &gitlab.UpdateIssueOptions{StateEvent: gitlab.String("close")})
	if err != nil {
		sendError(err)
		return
	}
	sendReply(fmt.Sprintf("Closed %s", gitIssue.WebURL))
}

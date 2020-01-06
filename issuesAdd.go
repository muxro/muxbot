package main

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

// IssuesAddOptions marks the data used to create a gitlab issue
type IssuesAddOptions struct {
	Assignee    string
	Repo        string
	Title       string
	Description string
	Tags        []string
}

func issueAddHandler(git *gitlab.Client, projects []*gitlab.Project, session *discordgo.Session, message *discordgo.MessageCreate) {
	params := strings.Split(message.Content, " ")[1:]

	_, sendReply, sendError := initMessageSenders(session, message)

	asTok, exists := associatedKey(message.Author.ID)
	if exists == false {
		sendReply("Error: You don't have a gitlab Personal Access Token associated with your account")
		return
	}
	if len(params) < 3 {
		sendReply("Utilizare: " + *prefix + "issues add <project name> <issue title>")
		return
	}
	ok := false
	for _, project := range projects {
		if isSameRepo(params[1], project) {
			ok = true
			issueName := strings.Join(params[2:], " ")
			userGit := gitlab.NewClient(nil, asTok)
			_, _, err := userGit.Issues.CreateIssue(project.ID, &gitlab.CreateIssueOptions{Title: gitlab.String(issueName)})
			if err != nil {
				sendError(err)
				return
			}
			sendReply("Issue adaugat")
			break
		}
	}
	if ok == false {
		sendReply("Error: No project found")
	}
}

func parseAddOpts(params []string, projects []*gitlab.Project) IssuesAddOptions {
	// TODO
	return IssuesAddOptions{}
}

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

// IssuesAddOptions marks the data used to create a gitlab issue
type IssuesAddOptions struct {
	Assignee    int
	Title       string
	Description string
	Tags        gitlab.Labels
	ProjectID   int
}

func issueAddHandler(git *gitlab.Client, projects []*gitlab.Project, session *discordgo.Session, message *discordgo.MessageCreate) {
	params := strings.Split(message.Content, " ")[1:]

	sendEmbed := func(embed *discordgo.MessageEmbed) *discordgo.Message {
		resulting, _ := session.ChannelMessageSendEmbed(message.ChannelID, embed)
		return resulting
	}

	_, sendReply, sendError := initMessageSenders(session, message)

	asTok, exists := associatedKey(message.Author.ID)
	if exists == false {
		sendReply("Error: You don't have a gitlab Personal Access Token associated with your account")
		return
	}
	if len(params) < 3 {
		sendReply("Usage: " + *prefix + "issues add <title> <issue opts> <description>")
		return
	}
	opts, err := parseAddOpts(params[1:], projects, git)
	if err != nil {
		sendError(err)
		return
	}
	// fmt.Printf("%#v", opts)
	activeRepo, exists := getActiveRepo(message.Author.ID)
	if opts.ProjectID == 0 {
		if exists {
			opts.ProjectID = getRepo(activeRepo, projects).ID
		} else {
			sendReply("Error: No repo specified and no active repo set")
			return
		}
	}
	userGit := gitlab.NewClient(nil, asTok)
	issue, _, err := userGit.Issues.CreateIssue(opts.ProjectID,
		&gitlab.CreateIssueOptions{
			Title:       gitlab.String(opts.Title),
			Description: gitlab.String(opts.Description),
			AssigneeIDs: []int{opts.Assignee},
			Labels:      &opts.Tags,
		})
	if err != nil {
		sendError(err)
	}
	sendEmbed(&discordgo.MessageEmbed{Description: fmt.Sprintf("Created issue [#%d %s](%s)", issue.IID, issue.Title, issue.WebURL)})
}

func parseAddOpts(params []string, projects []*gitlab.Project, git *gitlab.Client) (IssuesAddOptions, error) {
	ret := IssuesAddOptions{}
	if len(params) < 1 {
		return ret, errors.New("No parameters specified")
	}
	var noParamText string
	for _, param := range params {
		switch param[0] {
		case '&':
			if isRepo(param[1:], projects) {
				ret.ProjectID = getRepo(param[1:], projects).ID
			} else {
				return IssuesAddOptions{}, errors.New("Invalid Repo")
			}
		case '+':
			ret.Tags = append(ret.Tags, param[1:])
		case '$':
			user, err := getUserFromName(param[1:], git)
			if err != nil || user == nil {
				return IssuesAddOptions{}, errors.New("Assignee user not found")
			}
			ret.Assignee = user.ID
		default:
			noParamText += param + " "
		}
	}
	titleAndDesc := strings.SplitN(noParamText, " -- ", 2)
	if len(titleAndDesc) == 0 {
		return IssuesAddOptions{}, errors.New("No title specified")
	}
	ret.Title = titleAndDesc[0]
	if len(titleAndDesc) == 2 {
		ret.Description = titleAndDesc[1]
	} else {
		ret.Description = "No description provided."
	}
	return ret, nil
}

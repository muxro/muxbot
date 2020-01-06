package main

import (
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
	opts, strErr := parseAddOpts(params[1:], projects, git)
	if strErr != "" {
		sendReply("Error: " + strErr)
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
	sendEmbed(&discordgo.MessageEmbed{Description: fmt.Sprintf("Created issue [#%d %s](%s)", issue.IID, issue.Title, issue.Links.Self)})
}

func parseAddOpts(params []string, projects []*gitlab.Project, git *gitlab.Client) (IssuesAddOptions, string) {
	ret := IssuesAddOptions{}
	if len(params) < 1 {
		return ret, ""
	}
	descActive := false
	for _, param := range params {
		if param[0] == '&' { // project
			descActive = true
			if isRepo(param[1:], projects) {
				ret.ProjectID = getRepo(param[1:], projects).ID
			} else {
				return IssuesAddOptions{}, "Invalid Repo"
			}
		} else if param[0] == '+' { // tag
			descActive = true
			ret.Tags = append(ret.Tags, param[1:])
		} else if param[0] == '$' { // assignee
			descActive = true
			user, err := getUserFromName(param[1:], git)
			if err != nil {
				return ret, "User not found"
			}
			ret.Assignee = user.ID
		} else {
			if descActive { // append to description
				if ret.Description == "" {
					ret.Description = param
				} else {
					ret.Description += " " + param
				}
			} else { // append to title
				if ret.Title == "" {
					ret.Title = param
				} else {
					ret.Title += " " + param
				}
			}
		}
	}
	return ret, ""
}

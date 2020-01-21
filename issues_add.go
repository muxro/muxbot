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
	Assignee    string
	Title       string
	Description string
	Tags        gitlab.Labels
	Project     string
	ProjectID   int
}

func issueAddHandler(bot *Bot, projects []*gitlab.Project, args []string, msg *discordgo.Message) error {
	asTok, exists := associatedKey(msg.Author.ID)
	if exists == false {
		return errors.New("you don't have a gitlab Personal Access Token associated with your account")
	}
	if len(args) < 1 {
		return errors.New("not enough parameters")
	}
	opts, err := parseAddOpts(args)
	if err != nil {
		return err
	}
	if opts.Project != "" {
		if !isRepo(opts.Project, projects) {
			return errors.New("invalid project")
		}
		opts.ProjectID = getRepo(opts.Project, projects).ID
	}

	var assigneeID int
	if opts.Assignee != "" {
		user, err := getUserFromName(opts.Assignee, bot.git)
		if err != nil {
			return errors.New("assignee not found")
		}

		assigneeID = user.ID
	}

	activeRepo, exists := getActiveRepo(msg.Author.ID)
	if opts.ProjectID == 0 {
		if exists {
			opts.ProjectID = getRepo(activeRepo, projects).ID
		} else {
			return errors.New("No repo specified and no active repo set")
		}
	}
	userGit := gitlab.NewClient(nil, asTok)
	issue, _, err := userGit.Issues.CreateIssue(opts.ProjectID,
		&gitlab.CreateIssueOptions{
			Title:       gitlab.String(opts.Title),
			Description: gitlab.String(opts.Description),
			AssigneeIDs: []int{assigneeID},
			Labels:      &opts.Tags,
		})
	if err != nil {
		return err
	}

	bot.SendReply(msg, fmt.Sprintf("created issue <%s>", issue.WebURL))
	return nil
}

func parseAddOpts(args []string) (IssuesAddOptions, error) {
	ret := IssuesAddOptions{}
	if len(args) < 1 {
		return ret, errors.New("No parameters specified")
	}
	var noParamText string
	for _, param := range args {
		switch param[0] {
		case '&':
			ret.Project = param[1:]
		case '+':
			ret.Tags = append(ret.Tags, param[1:])
		case '$':
			ret.Assignee = param[1:]
		default:
			noParamText += param + " "
		}
	}
	titleAndDesc := strings.SplitN(strings.Trim(noParamText, " "), " -- ", 2)
	if titleAndDesc[0] == "" {
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

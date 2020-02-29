package main

import (
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

func issueAddHandler(bot *Bot, args []string, msg *discordgo.Message) error {
	git, err := getUserGit(msg)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return errInsufficientArgs
	}
	opts, err := parseAddOpts(args)
	if err != nil {
		return err
	}

	var assigneeID int
	if opts.Assignee != "" {
		user, err := getUserFromName(git, opts.Assignee)
		if err != nil {
			return errAssigneeNotFound
		}

		assigneeID = user.ID
	}

	opts.Project = getRepo(git, msg, opts.Project)
	if opts.Project != "" {
		if !isRepo(git, opts.Project) {
			return errInvalidRepo
		}
		opts.ProjectID, err = getRepoID(git, opts.Project, msg)
		if err != nil {
			return err
		}
	}
	issue, _, err := git.Issues.CreateIssue(opts.ProjectID,
		&gitlab.CreateIssueOptions{
			Title:       gitlab.String(opts.Title),
			Description: gitlab.String(opts.Description),
			AssigneeIDs: []int{assigneeID},
			Labels:      &opts.Tags,
		})
	if err != nil {
		return err
	}

	_, err = bot.SendReply(msg, fmt.Sprintf("created issue <%s>", issue.WebURL))
	return err
}

func parseAddOpts(args []string) (IssuesAddOptions, error) {
	ret := IssuesAddOptions{}
	if len(args) < 1 {
		return ret, errInsufficientArgs
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
		return IssuesAddOptions{}, errNoTitleSpecified
	}
	ret.Title = titleAndDesc[0]
	if len(titleAndDesc) == 2 {
		ret.Description = titleAndDesc[1]
	} else {
		ret.Description = "No description provided."
	}
	return ret, nil
}

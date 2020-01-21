package main

import (
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

// IssuesModifyOptions specifies the parameters given to .issues modify
type IssuesModifyOptions struct {
	TagsRemove gitlab.Labels
	TagsAdd    gitlab.Labels
	Repo       string
	ID         int
	Assignee   string
}

func issueModifyHandler(bot *Bot, projects []*gitlab.Project, args []string, msg *discordgo.Message) error {

	asTok, exists := associatedKey(msg.Author.ID)
	if exists == false {
		return errors.New("you don't have a gitlab Personal Access Token associated with your account")
	}
	if len(args) < 1 {
		return errors.New("not enough parameters")
	}

	opts, err := parseModifyOpts(args)
	if err != nil {
		return err
	}

	if opts.Repo == "" {
		activeRepo, exists := getActiveRepo(msg.Author.ID)
		if exists {
			opts.Repo = activeRepo
		} else {
			return errors.New("you need to specify either an active repo or a repo to search in")
		}
	}

	userGit := gitlab.NewClient(nil, asTok)
	pid := getRepo(opts.Repo, projects).ID
	issue, _, err := userGit.Issues.GetIssue(pid, opts.ID)
	if err != nil {
		return err
	}

	totalTags := []string(issue.Labels)
	newTags := gitlab.Labels{}
	for _, tag := range totalTags {
		if inTagArray(tag, opts.TagsRemove) == false && inTagArray(tag, opts.TagsAdd) == false {
			newTags = append(newTags, tag)
		}
	}
	for _, tag := range opts.TagsAdd {
		newTags = append(newTags, tag)
	}
	updateOpts := &gitlab.UpdateIssueOptions{}
	if opts.Assignee != "" {
		user, err := getUserFromName(opts.Assignee, bot.git)
		if err != nil || user == nil {
			return errors.New("could not find user")
		}

		updateOpts.AssigneeIDs = []int{user.ID}
	}
	updateOpts.Labels = &newTags
	issue, _, err = userGit.Issues.UpdateIssue(pid, opts.ID, updateOpts)
	if err != nil {
		return err
	}

	bot.SendReply(msg, "Issue successfully modified.")
	return nil
}

func parseModifyOpts(args []string) (IssuesModifyOptions, error) {
	var opts, emptyOpts IssuesModifyOptions
	id, repo, err := parseIssueParam(args[0])
	if err != nil {
		return emptyOpts, err
	}

	opts.ID = id
	opts.Repo = repo
	args = args[1:]
	for _, param := range args {
		switch param[0] {
		case '$':
			opts.Assignee = param[1:]
		case '-':
			opts.TagsRemove = append(opts.TagsRemove, param[1:])
		case '+':
			opts.TagsAdd = append(opts.TagsAdd, param[1:])
		}
	}
	return opts, nil
}

func inTagArray(element string, array []string) bool {
	for _, el := range array {
		if el == element {
			return true
		}
	}
	return false
}

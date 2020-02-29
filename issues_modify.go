package main

import (
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

func issueModifyHandler(bot *Bot, args []string, msg *discordgo.Message) error {
	git, err := getUserGit(msg)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return errInsufficientArgs
	}

	opts, err := parseModifyOpts(args)
	if err != nil {
		return err
	}

	opts.Repo = getRepo(git, msg, opts.Repo)

	pid, err := getRepoID(git, opts.Repo, msg)
	if err != nil {
		return err
	}

	issue, _, err := git.Issues.GetIssue(pid, opts.ID)
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
		user, err := getUserFromName(git, opts.Assignee)
		if err != nil || user == nil {
			return errNoUserFound
		}

		updateOpts.AssigneeIDs = []int{user.ID}
	}
	updateOpts.Labels = &newTags
	issue, _, err = git.Issues.UpdateIssue(pid, opts.ID, updateOpts)
	if err != nil {
		return err
	}

	_, err = bot.SendReply(msg, "Issue successfully modified.")
	return err
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

package main

import (
	"fmt"
	"strconv"
	"strings"

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

func issueModifyHandler(git *gitlab.Client, projects []*gitlab.Project, session *discordgo.Session, message *discordgo.MessageCreate) {
	params := strings.Split(message.Content, " ")[1:]
	_, sendReply, sendError := initMessageSenders(session, message)

	asTok, exists := associatedKey(message.Author.ID)
	if exists == false {
		sendReply("Error: You don't have a gitlab Personal Access Token associated with your account")
		return
	}
	if len(params) < 3 {
		sendReply("Usage: " + *prefix + "issues modify issueid <issue opts>")
		return
	}

	opts, ok := parseModifyOpts(params[1:])
	if ok == false {
		sendReply("Something you gave me isn't right, please try again.")
		return
	}
	if opts.Repo == "" {
		activeRepo, exists := getActiveRepo(message.Author.ID)
		if exists {
			opts.Repo = activeRepo
		} else {
			sendReply("You need to specify either an active repo or a repo to search in")
		}
	}

	userGit := gitlab.NewClient(nil, asTok)
	pid := getRepo(opts.Repo, projects).ID
	issue, _, err := userGit.Issues.GetIssue(pid, opts.ID)
	if err != nil {
		sendError(err)
		return
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
		user, err := getUserFromName(opts.Assignee, git)
		if err != nil {
			sendError(err)
			return
		}
		updateOpts.AssigneeIDs = []int{user.ID}
	}
	updateOpts.Labels = &newTags
	issue, _, err = userGit.Issues.UpdateIssue(pid, opts.ID, updateOpts)
	if err != nil {
		sendError(err)
		return
	}
	sendReply("Issue successfully modified.")
	fmt.Printf("%#v\n", issue)
}

func parseModifyOpts(params []string) (opts IssuesModifyOptions, ok bool) {
	fmt.Println(params)
	issue := params[0]
	ok = true
	if len(strings.Split(issue, "#")) == 2 {
		split := strings.Split(issue, "#")
		id, err := strconv.Atoi(split[1])
		if err != nil {
			ok = false
			return
		}
		opts.ID = id
		opts.Repo = split[0]
	} else {
		if issue[0] >= '0' && issue[0] <= '9' {
			id, err := strconv.Atoi(issue)
			if err != nil {
				ok = false
				return
			}
			opts.ID = id
		} else {
			ok = false
			return
		}
	}
	params = params[1:]
	for _, param := range params {
		if param[0] == '$' { // assignee
			opts.Assignee = param[1:]
		} else if param[0] == '-' { // remove tag
			opts.TagsRemove = append(opts.TagsRemove, param[1:])
		} else if param[0] == '+' { // add tag
			opts.TagsAdd = append(opts.TagsAdd, param[1:])
		}
	}
	return
}

func inTagArray(element string, array []string) bool {
	for _, el := range array {
		if el == element {
			return true
		}
	}
	return false
}

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

var (
	issueRegex = regexp.MustCompile(`(([^/\s]+/)?([^\s]+))?#(\d+)`)
)

func issueReferenceHandler(bot *Bot, msg *discordgo.Message) (bool, error) {
	git, err := getUserGit(msg)
	if err != nil {
		return false, nil
	}
	matches := issueRegex.FindAllStringSubmatch(msg.Content, -1)
	if len(matches) == 0 {
		return false, nil
	}

	var issues []*gitlab.Issue
	found := make(map[string]bool)

	for _, match := range matches {
		repo := getRepo(git, msg, match[1])
		if repo == "" {
			return false, nil
		}

		issueid, err := strconv.Atoi(match[4])
		if err != nil {
			return false, nil
		}

		namespace := strings.SplitN(repo, "/", 2)[0]
		fqID := fmt.Sprintf("%s/%s#%d", namespace, repo, issueid)
		if _, ok := found[fqID]; ok {
			continue
		}
		found[fqID] = true

		issue, err := getIssue(git, msg, issueid, repo)
		if err != nil {
			return false, nil
		}

		issues = append(issues, issue)
		if len(issues) == 5 {
			break
		}
	}

	var resp string
	for _, issue := range issues {
		resp += displayIssue(git, issue) + "\n"
	}

	bot.SendReply(msg, resp)
	return true, nil
}

func displayIssue(git *gitlab.Client, issue *gitlab.Issue) string {
	var ret = "```golang\n"
	project, _ := getIssueProject(git, issue)
	ret += project.Namespace.Path + "/" + project.Path + "#" + strconv.Itoa(issue.IID) + "\n"
	ret += issue.Title
	if issue.Assignee != nil {
		ret += " -- assigned to " + issue.Assignee.Name
	} else {
		ret += " -- created by " + issue.Author.Name
	}

	ret += "\nStatus: " + issue.State + "\n"
	if len(issue.Labels) > 0 {
		ret += "Tags: " + strings.Join(issue.Labels, " ") + "\n"
	}

	ret += "\n"
	if issue.Description != "" {
		ret += issue.Description + "\n"
	}

	ret += "```"

	return ret
}

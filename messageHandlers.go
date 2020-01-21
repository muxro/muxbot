package main

import (
	"errors"
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

// CommandMessageHandler is a wrapper for command handling
func CommandMessageHandler(cmds *CommandMux) MessageHandler {
	return func(b *Bot, message *discordgo.Message) (bool, error) {
		if !strings.HasPrefix(message.Content, *prefix) {
			return false, nil
		}

		return true, cmds.Handle(b, message)
	}
}

func issueReferenceHandler(bot *Bot, message *discordgo.Message) (bool, error) {
	matches := issueRegex.FindAllStringSubmatch(message.Content, -1)
	if len(matches) == 0 {
		return false, nil
	}

	shown := make(map[string]bool)
	issues := []*gitlab.Issue{}

	opt := &gitlab.ListProjectsOptions{Membership: gitlab.Bool(true)}
	projects, _, err := bot.git.Projects.ListProjects(opt)
	if err != nil {
		return true, err
	}

	for _, match := range matches {
		repo := match[1]
		if repo == "" {
			activeRepo, exists := getActiveRepo(message.Author.ID)
			if exists == false {
				return true, errors.New("no active repo set and no repo specified")
			}
			repo = activeRepo
		}

		issueid, err := strconv.Atoi(match[4])
		if err != nil {
			return true, errors.New("invalid issue id")
		}

		gottenRepo := getRepo(repo, projects)
		if gottenRepo == nil {
			return true, errors.New("invalid repo")
		}

		issueString := fmt.Sprintf("%s/%s#%d", gottenRepo.Namespace.Path, gottenRepo.Path, issueid)
		if _, ok := shown[issueString]; !ok {
			projectid := gottenRepo.ID
			issue, _, err := bot.git.Issues.GetIssue(projectid, issueid)
			if err != nil {
				return true, err
			}
			issues = append(issues, issue)
			shown[issueString] = true
		}
	}

	var rez string
	for _, issue := range issues {
		rez += displayIssue(issue) + "\n"
	}

	bot.SendReply(message, rez)
	return true, nil
}

func displayIssue(issue *gitlab.Issue) string {
	var ret = "```golang\n"
	project, _ := getIssueProject(issue)
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

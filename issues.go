package main

import (
	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

// func issueHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender) error {
func issueHandler(bot *Bot, msg *discordgo.Message, args string) error {
	parts := strings.Fields(args)
	git := gitlab.NewClient(nil, *gitlabToken)
	opt := &gitlab.ListProjectsOptions{Membership: gitlab.Bool(true)}
	projects, _, err := git.Projects.ListProjects(opt)
	if err != nil {
		return err
	}

	if len(parts) < 1 {
		return errors.New("not enough args")
	}

	issueMux := NewCommandMux()
	issueMux.IssueCommand("list", issueListHandler, git, projects)
	issueMux.IssueCommand("add", issueAddHandler, git, projects)
	issueMux.IssueCommand("activeRepo", issuesActiveRepoHandler, git, projects)
	issueMux.IssueCommand("modify", issueModifyHandler, git, projects)
	issueMux.IssueCommand("close", issueCloseHandler, git, projects)

	msg.Content = strings.Join(parts, " ")
	return issueMux.Handle(bot, msg)

	// switch name {
	// case "list":
	// 	return issueListHandler(bot, git, projects, parts, msg)
	// case "add":
	// 	return issueAddHandler(bot, git, projects, parts, msg)
	// case "activeRepo":
	// 	return issuesActiveRepoHandler(bot, git, projects, parts, msg)
	// case "modify":
	// 	return issueModifyHandler(bot, git, projects, parts, msg)
	// case "close":
	// 	return issueCloseHandler(bot, git, projects, parts, msg)
	// default:
	// 	return errors.New("Not enough parameters")
	// }
}

func getIssueProject(issue *gitlab.Issue) (*gitlab.Project, error) {
	git := gitlab.NewClient(nil, *gitlabToken)
	project, _, err := git.Projects.GetProject(issue.ProjectID, nil)
	return project, err
}

func getRepo(name string, projects []*gitlab.Project) *gitlab.Project {
	for _, project := range projects {
		if isSameRepo(name, project) {
			return project
		}
	}
	return nil
}

func isRepo(name string, projects []*gitlab.Project) bool {
	name = strings.ToLower(name)
	for _, project := range projects {
		if isSameRepo(name, project) {
			return true
		}
	}
	return false
}

func isSameRepo(name string, project *gitlab.Project) bool {
	name = strings.ToLower(name)
	if project.Path == name ||
		project.Name == name ||
		project.Namespace.Name+"/"+project.Path == name ||
		project.Namespace.Path+"/"+project.Path == name ||
		project.Namespace.Name+"/"+project.Name == name ||
		project.Namespace.Path+"/"+project.Name == name {
		return true
	}
	return false
}

func gitlabKeyHandler(bot *Bot, msg *discordgo.Message, key string) error {
	err := bot.ds.ChannelMessageDelete(msg.ChannelID, msg.ID)
	if err != nil {
		bot.SendReply(msg, "Beware, I can't delete the message, keep the key safe")
	}

	result, ok := testKey(key)
	if ok == true {
		err := associateUserToToken(msg.Author.ID, key)
		if err != nil {
			return err
		}

		bot.SendReply(msg, "Associated user with gitlab user "+result.Name)
	} else {
		bot.SendReply(msg, "Invalid key")
	}
	return nil
}

func getUserFromName(username string, git *gitlab.Client) (*gitlab.User, error) {
	users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{Username: gitlab.String(username)})
	if err != nil {
		return nil, err
	}

	if len(users) < 1 {
		return nil, errors.New("No user found")
	}
	return users[0], nil
}

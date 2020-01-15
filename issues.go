package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

func issueHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	git := gitlab.NewClient(nil, *gitlabToken)
	opt := &gitlab.ListProjectsOptions{Membership: gitlab.Bool(true)}
	params := getArguments(message)
	projects, _, err := git.Projects.ListProjects(opt)
	if err != nil {
		sendError(err)
		return
	}
	if len(params) < 1 {
		sendReply("Usage: " + *prefix + "issues <list|create|modify>")
		return
	}
	switch params[0] {
	case "list":
		issueListHandler(git, projects, session, message)
	case "add":
		issueAddHandler(git, projects, session, message)
	case "activeRepo":
		if len(params) < 2 {
			sendReply("Usage: " + *prefix + "issues activeRepo <set/get/erase>")
			return
		}
		switch params[1] {
		case "set":
			if len(params) != 3 {
				sendReply("Usage: " + *prefix + "isseus activeRepo set <repo>")
				return
			}
			if isRepo(params[2], projects) == false {
				sendReply("You need to specify a valid repo which you are a member of that the bot can see")
				return
			}
			if strings.ContainsAny(params[2], "/") == false { // we would also like a group name
				for _, project := range projects {
					if isSameRepo(params[2], project) {
						params[2] = project.Namespace.Path + "/" + project.Path
						break
					}
				}
			}
			err := setActiveRepo(message.Author.ID, params[2])
			if err != nil {
				sendError(err)
			}
			sendReply("Set active repo " + params[2])
		case "get":
			repo, exists := getActiveRepo(message.Author.ID)
			if exists == false {
				sendReply("Error: No active repo set")
				return
			}
			sendReply("Your active repo is " + repo)
		case "erase":
			err = removeActiveRepo(message.Author.ID)
			if err != nil {
				sendError(err)
				return
			}
			sendReply("Your active repo has been erased from the database")
		default:
			sendReply("Usage: " + *prefix + "issues activeRepo <set/get/erase>")
		}
	case "modify":
		issueModifyHandler(git, projects, session, message)
	case "close":
		issueCloseHandler(git, projects, session, message)
	default:
		sendReply("Refer to " + *prefix + "help for a list of commands")
	}
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

func glKeyHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	key := strings.Join(strings.Split(message.Content, " ")[1:], " ")
	err := session.ChannelMessageDelete(message.ChannelID, message.ID)
	if err != nil {
		sendReply("Beware, I can't delete the message, keep the key safe")
	}
	result, ok := testKey(key)
	if ok == true {
		err := associateUserToToken(message.Author.ID, key)

		if err != nil {
			sendError(err)
			return
		}
		sendReply("Associated user with gitlab user " + result.Name)
	} else {
		sendReply("Invalid key")
	}
}

func getUserFromName(username string, git *gitlab.Client) (*gitlab.User, error) {
	users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{Username: gitlab.String(username)})
	if err != nil {
		fmt.Printf("%#v\n", err)
		return nil, err
	}
	if len(users) < 1 {
		fmt.Printf("No user found\n")
		return nil, nil
	}
	return users[0], nil
}

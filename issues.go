package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

// IssueMsgOptions stores data about the rendering of the list
type IssueMsgOptions struct {
	ShowGroup    bool
	ShowRepo     bool
	ShowTags     bool
	ShowAuthor   bool
	ShowAssignee bool
}

// IssuesListOptions stores data about the issues
type IssuesListOptions struct {
	Group      string
	Repo       string
	Author     string
	Assignee   string
	Tags       []string
	Title      string
	InternalID int
	URL        string
}

// IssuesSearchOptions stores data about the searching of issues
type IssuesSearchOptions struct {
	Group    string
	Repo     string
	Author   string
	Assignee string
	Tags     []string
	Self     bool
}

func handleIssue(session *discordgo.Session, message *discordgo.Message) {
	sendMessage := func(sentMsg string) *discordgo.Message {
		resulting, _ := session.ChannelMessageSend(message.ChannelID, sentMsg)
		return resulting
	}

	sendReply := func(sentMsg string) *discordgo.Message {
		return sendMessage(fmt.Sprintf("(%s) %s", strings.Split(message.Author.String(), "#")[0], sentMsg))
	}

	sendError := func(err error) *discordgo.Message {
		return sendReply(fmt.Sprintf("Eroare: %v", err))
	}

	sendEmbed := func(embed *discordgo.MessageEmbed) *discordgo.Message {
		resulting, _ := session.ChannelMessageSendEmbed(message.ChannelID, embed)
		return resulting
	}

	git := gitlab.NewClient(nil, *gitlabToken)
	opt := &gitlab.ListProjectsOptions{Membership: gitlab.Bool(true)}
	params := strings.Split(message.Content, " ")[1:]
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
		searchOpts, msgOpts := parseIssueOpts(params[1:], projects)
		issueList := []IssuesListOptions{}
		if searchOpts.Self == true {
			selfUname, err := getGitlabUnameFromUser(message.Author.ID)
			if err != nil {
				sendError(err)
				return
			}
			if selfUname == "" {
				sendReply("You mentioned $self, but you don't have a gitlab key associated")
				return
			}
			searchOpts.Assignee = selfUname
		}
		for _, project := range projects {
			if searchOpts.Group == "" || project.Namespace.Path == searchOpts.Group {
				if searchOpts.Repo == "" || project.Name == searchOpts.Repo {
					issues, _, err := git.Issues.ListProjectIssues(project.ID, &gitlab.ListProjectIssuesOptions{Sort: gitlab.String("asc"), Labels: searchOpts.Tags})
					if err != nil {
						sendError(err)
						return
					}
					for _, issue := range issues {
						if searchOpts.Author == "" || issue.Author.Name == searchOpts.Author {
							if (searchOpts.Assignee == "") ||
								(issue.Assignee != nil && issue.Assignee.Name == searchOpts.Assignee) {

								assignee := ""
								if issue.Assignee != nil {
									assignee = issue.Assignee.Name
								}
								issueList = append(issueList, IssuesListOptions{Group: project.Namespace.Path,
									Repo:       project.Name,
									Author:     issue.Author.Name,
									Assignee:   assignee,
									Tags:       issue.Labels,
									Title:      issue.Title,
									InternalID: issue.IID,
									URL:        issue.WebURL})
							}
						}
					}
				}
			}
		}
		if len(issueList) == 0 {
			sendReply("No issue found")
		} else {
			issues := []string{}
			for _, issue := range issueList {
				issueText := "["
				if msgOpts.ShowGroup {
					issueText += issue.Group
					if msgOpts.ShowRepo {
						issueText += "/"
					}
				}
				if msgOpts.ShowRepo {
					issueText += issue.Repo
				}
				issueText += "#" + strconv.Itoa(issue.InternalID) + " "
				if len(issue.Tags) > 0 && msgOpts.ShowTags {
					issueText += "["
					for i, tag := range issue.Tags {
						issueText += tag
						if i != len(issue.Tags)-1 {
							issueText += ", "
						}
					}
					issueText += "] "
				}
				issueText += issue.Title
				if issue.Assignee != "" && msgOpts.ShowAssignee {
					issueText += " - assigned to " + issue.Assignee
				} else if msgOpts.ShowAuthor {
					issueText += " - created by " + issue.Author
				}
				issueText += "](" + issue.URL + ")"
				issues = append(issues, issueText)
			}
			sendEmbed(&discordgo.MessageEmbed{Description: strings.Join(issues, "\n")})
		}
	case "add":
		asTok, exists := associatedKey(message.Author.ID)
		if exists == false {
			sendReply("Eroare: Nu ai asociat un Personal Access Token gitlab cu contul tau")
			return
		}
		if len(params) < 3 {
			sendReply("Utilizare: " + *prefix + "issues add <project name> <issue title>")
			return
		}
		ok := false
		for _, project := range projects {
			if project.Name == params[1] || project.PathWithNamespace == params[1] {
				ok = true
				issueName := strings.Join(params[2:], " ")
				userGit := gitlab.NewClient(nil, asTok)
				_, _, err := userGit.Issues.CreateIssue(project.ID, &gitlab.CreateIssueOptions{Title: gitlab.String(issueName)})
				if err != nil {
					sendError(err)
					return
				}
				sendReply("Issue adaugat")
				break
			}
		}
		if ok == false {
			sendReply("Error: No project found")
		}
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
				sendReply("You need to specify a valid repo which you are a member of")
				return
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

	}
}

func getIssueProject(issue *gitlab.Issue) (*gitlab.Project, error) {
	git := gitlab.NewClient(nil, *gitlabToken)
	project, _, err := git.Projects.GetProject(issue.ProjectID, nil)
	return project, err
}

func parseIssueOpts(params []string, projects []*gitlab.Project) (IssuesSearchOptions, IssueMsgOptions) {
	ret := IssuesSearchOptions{}
	msgOptions := IssueMsgOptions{ShowGroup: true, ShowAuthor: true, ShowRepo: true, ShowTags: true, ShowAssignee: true}
	if len(params) < 1 { // It's empty
		return ret, msgOptions
	}
	for _, param := range params {
		if param[0] == '^' { // Author
			msgOptions.ShowAuthor = false
			ret.Author = param[1:]
		} else if param[0] == '$' { // assignee
			if param == "$any" {
				msgOptions.ShowAuthor = false
			} else if param == "$self" {
				ret.Self = true
			} else {
				ret.Assignee = param[1:]
			}
			msgOptions.ShowAssignee = false
		} else if param[0] == '+' {
			ret.Tags = append(ret.Tags, param[1:])
			msgOptions.ShowTags = false
		} else {
			msgOptions.ShowGroup = false
			if strings.Contains(param, "/") {
				msgOptions.ShowRepo = false
				repoName := strings.Split(param, "/")
				ret.Group = repoName[0]
				ret.Repo = repoName[1]
			} else {

				if isRepo(param, projects) == true {
					msgOptions.ShowRepo = false
					ret.Repo = param
				} else {
					ret.Group = param
				}
			}
		}
	}
	return ret, msgOptions
}

func isRepo(name string, projects []*gitlab.Project) bool {
	name = strings.ToLower(name)
	for _, project := range projects {
		if project.Path == name ||
			project.Name == name ||
			project.Namespace.Name+"/"+project.Path == name ||
			project.Namespace.Path+"/"+project.Path == name ||
			project.Namespace.Name+"/"+project.Name == name ||
			project.Namespace.Path+"/"+project.Name == name {
			return true
		}
	}
	return false
}

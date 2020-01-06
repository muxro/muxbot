package main

import (
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
	Any      bool
}

func issueListHandler(git *gitlab.Client, projects []*gitlab.Project, session *discordgo.Session, message *discordgo.MessageCreate) {
	sendEmbed := func(embed *discordgo.MessageEmbed) *discordgo.Message {
		resulting, _ := session.ChannelMessageSendEmbed(message.ChannelID, embed)
		return resulting
	}

	_, sendReply, sendError := initMessageSenders(session, message)

	params := strings.Split(message.Content, " ")[1:]

	searchOpts, msgOpts := parseListOpts(params[1:], projects)
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
	activeRepo, exists := getActiveRepo(message.Author.ID)
	if searchOpts.Any == false && searchOpts.Group == "" && searchOpts.Repo == "" && exists == true {
		repoData := strings.SplitN(activeRepo, "/", 2)
		searchOpts.Group = repoData[0]
		searchOpts.Repo = repoData[1]
		msgOpts.ShowGroup = false
		msgOpts.ShowRepo = false
		sendReply("Using active repo " + activeRepo)
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
}

func parseListOpts(params []string, projects []*gitlab.Project) (IssuesSearchOptions, IssueMsgOptions) {
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
		} else if param[0] == '&' {
			msgOptions.ShowGroup = false
			if param == "&any" {
				msgOptions.ShowGroup = true
				msgOptions.ShowRepo = true
				ret.Any = true
			} else if strings.Contains(param, "/") && isRepo(param, projects) {
				msgOptions.ShowRepo = false
				repoName := strings.Split(param[1:], "/")
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
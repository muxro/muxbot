package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xanzy/go-gitlab"
)

var (
	errInsufficientArgs = errors.New("not enough arguments")
	errNoPAC            = errors.New("you don't have a gitlab Personal Access Token (PAC) associated with your account")
	errNoRepoSpecified  = errors.New("you need to specify either an active repo or a repo to search in")
	errNoRepoFound      = errors.New("you need to specify a valid repo which you are a member of that the bot can see")
	errNoActiveRepo     = errors.New("no channel active repo set")
	errNoUserFound      = errors.New("user not found")
	errInvalidRepo      = errors.New("invalid repo")
	errAssigneeNotFound = errors.New("assignee not found")
	errNoTitleSpecified = errors.New("no title specified")
	errNoIssueFound     = errors.New("no issue found")
	errInvalidID        = errors.New("invalid ID")
)

// Issues is the main wrapper for all issue related
type Issues struct {
	git *gitlab.Client
}

// NewIssues initializes a new Issues instance
func NewIssues(key string) *Issues {
	return &Issues{
		git: gitlab.NewClient(nil, *gitlabToken),
	}
}

func (i *Issues) issueHandler(bot *Bot, msg *discordgo.Message, args string) error {
	parts := strings.Fields(args)
	if len(parts) < 1 {
		return errInsufficientArgs
	}

	issueMux := NewCommandMux()
	issueMux.IssueCommand("list", i.issueListHandler)
	issueMux.IssueCommand("add", i.issueAddHandler)
	issueMux.IssueCommand("active-repo", i.issuesActiveRepoHandler)
	issueMux.IssueCommand("modify", i.issueModifyHandler)
	issueMux.IssueCommand("close", i.issueCloseHandler)

	msg.Content = strings.Join(parts, " ")
	return issueMux.Handle(bot, msg)
}

func (i *Issues) getIssueProject(issue *gitlab.Issue) (*gitlab.Project, error) {
	git := gitlab.NewClient(nil, *gitlabToken)
	project, _, err := git.Projects.GetProject(issue.ProjectID, nil)
	return project, err
}

func (i *Issues) getGitlabRepo(name string, message *discordgo.Message) (*gitlab.Project, error) {
	if !i.isRepo(name) {
		return nil, errInvalidRepo
	}

	projects, err := i.getProjects()
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if i.isSameRepo(name, project) {
			return project, nil
		}
	}

	activeRepo, exists := getActiveRepo(message.ChannelID)
	if exists == false {
		return nil, errNoRepoFound
	}

	return i.getGitlabRepo(activeRepo, message)
}

func (i *Issues) getRepo(message *discordgo.Message, repo string) string {
	parts := strings.SplitN(repo, "/", 2)
	var namespace string
	if len(parts) == 2 {
		namespace, repo = parts[0], parts[1]
	} else {
		repo = parts[0]
	}
	if repo == "" {
		activeRepo, exists := getActiveRepo(message.ChannelID)
		if !exists {
			return ""
		}
		return activeRepo
	}
	if namespace == "" {
		repo, err := i.getGitlabRepo(repo, message)
		if err != nil {
			return ""
		}
		return repo.Namespace.Path + "/" + repo.Path
	}
	return namespace + "/" + repo
}

func (i *Issues) getFullRepoName(repo *gitlab.Project) (string, string) {
	return repo.Namespace.Path, repo.Path
}

func (i *Issues) splitRepo(repo string) (string, string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

func (i *Issues) getIssue(message *discordgo.Message, iid int, repo string) (*gitlab.Issue, error) {
	rawRepo, err := i.getGitlabRepo(repo, message)
	if err != nil {
		return nil, err
	}

	issue, _, err := i.git.Issues.GetIssue(rawRepo.ID, iid)
	if err != nil {
		return nil, errNoIssueFound
	}
	return issue, nil
}

func (i *Issues) isRepo(name string) bool {
	projects, err := i.getProjects()
	if err != nil {
		return false
	}
	name = strings.ToLower(name)
	for _, project := range projects {
		if i.isSameRepo(name, project) {
			return true
		}
	}
	return false
}

func (i *Issues) getRepoID(name string, message *discordgo.Message) (int, error) {
	repo, err := i.getGitlabRepo(name, message)
	if err != nil {
		return -1, err
	}
	return repo.ID, nil
}

func (i *Issues) getProjects() ([]*gitlab.Project, error) {
	opt := &gitlab.ListProjectsOptions{Membership: gitlab.Bool(true)}
	projects, _, err := i.git.Projects.ListProjects(opt)
	return projects, err
}

func (i *Issues) isSameRepo(name string, project *gitlab.Project) bool {
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

func (i *Issues) gitlabKeyHandler(bot *Bot, msg *discordgo.Message, key string) error {
	err := bot.ds.ChannelMessageDelete(msg.ChannelID, msg.ID)
	preMessage := ""
	if err != nil {
		preMessage = "Beware, I can't delete the message, keep the key safe\n"
	}

	result, ok := testKey(key)
	if ok == true {
		err := associateUserToToken(msg.Author.ID, key)
		if err != nil {
			return err
		}

		bot.SendReply(msg, preMessage+"Associated user with gitlab user "+result.Name)
	} else {
		bot.SendReply(msg, "Invalid key")
	}
	return nil
}

func (i *Issues) getUserFromName(username string) (*gitlab.User, error) {
	users, _, err := i.git.Users.ListUsers(&gitlab.ListUsersOptions{Username: gitlab.String(username)})
	if err != nil {
		return nil, err
	}

	if len(users) < 1 {
		return nil, errNoUserFound
	}
	return users[0], nil
}

func parseIssueParam(issue string) (int, string, error) {
	var id int
	var repo string
	if len(strings.Split(issue, "#")) == 2 {
		split := strings.Split(issue, "#")
		ID, err := strconv.Atoi(split[1])
		if err != nil {
			return -1, "", errInvalidID
		}

		id = ID
		repo = split[0]
	} else {
		if issue[0] >= '0' && issue[0] <= '9' {
			ID, err := strconv.Atoi(issue)
			if err != nil {
				return -1, "", errInvalidID
			}

			id = ID
		} else {
			return -1, "", errInvalidID
		}
	}
	return id, repo, nil
}

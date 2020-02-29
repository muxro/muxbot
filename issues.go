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
		git: gitlab.NewClient(nil, key),
	}
}

func issueHandler(bot *Bot, msg *discordgo.Message, args string) error {
	parts := strings.Fields(args)
	if len(parts) < 1 {
		return errInsufficientArgs
	}

	issueMux := NewCommandMux()
	issueMux.IssueCommand("list", issueListHandler)
	issueMux.IssueCommand("add", issueAddHandler)
	issueMux.IssueCommand("active-repo", issuesActiveRepoHandler)
	issueMux.IssueCommand("modify", issueModifyHandler)
	issueMux.IssueCommand("close", issueCloseHandler)

	msg.Content = strings.Join(parts, " ")
	return issueMux.Handle(bot, msg)
}

func getIssueProject(git *gitlab.Client, issue *gitlab.Issue) (*gitlab.Project, error) {
	project, _, err := git.Projects.GetProject(issue.ProjectID, nil)
	return project, err
}

func getGitlabRepo(git *gitlab.Client, name string, message *discordgo.Message) (*gitlab.Project, error) {
	if !isRepo(git, name) {
		return nil, errInvalidRepo
	}

	projects, err := getProjects(git)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if isSameRepo(name, project) {
			return project, nil
		}
	}

	activeRepo, exists := getActiveRepo(message.ChannelID)
	if exists == false {
		return nil, errNoRepoFound
	}

	return getGitlabRepo(git, activeRepo, message)
}

func getRepo(git *gitlab.Client, message *discordgo.Message, repo string) string {
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
		repo, err := getGitlabRepo(git, repo, message)
		if err != nil {
			return ""
		}
		return repo.Namespace.Path + "/" + repo.Path
	}
	return namespace + "/" + repo
}

func getFullRepoName(repo *gitlab.Project) (string, string) {
	return repo.Namespace.Path, repo.Path
}

func splitRepo(repo string) (string, string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

func getIssue(git *gitlab.Client, message *discordgo.Message, iid int, repo string) (*gitlab.Issue, error) {
	rawRepo, err := getGitlabRepo(git, repo, message)
	if err != nil {
		return nil, err
	}

	issue, _, err := git.Issues.GetIssue(rawRepo.ID, iid)
	if err != nil {
		return nil, errNoIssueFound
	}
	return issue, nil
}

func isRepo(git *gitlab.Client, name string) bool {
	projects, err := getProjects(git)
	if err != nil {
		return false
	}
	name = strings.ToLower(name)
	for _, project := range projects {
		if isSameRepo(name, project) {
			return true
		}
	}
	return false
}

func getRepoID(git *gitlab.Client, name string, message *discordgo.Message) (int, error) {
	repo, err := getGitlabRepo(git, name, message)
	if err != nil {
		return -1, err
	}
	return repo.ID, nil
}

func getProjects(git *gitlab.Client) ([]*gitlab.Project, error) {
	opt := &gitlab.ListProjectsOptions{Membership: gitlab.Bool(true)}
	projects, _, err := git.Projects.ListProjects(opt)
	return projects, err
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
	preMessage := ""
	if err != nil {
		preMessage = "Beware, I can't delete the message, keep the key safe\n"
	}

	result, ok := testKey(key)
	if ok == true {
		err = associateUserToToken(msg.Author.ID, key)
		if err != nil {
			return err
		}

		_, err = bot.SendReply(msg, preMessage+"Associated user with gitlab user "+result.Name)
	} else {
		_, err = bot.SendReply(msg, "Invalid key")
	}
	return err
}

func getUserFromName(git *gitlab.Client, username string) (*gitlab.User, error) {

	users, _, err := git.Users.ListUsers(&gitlab.ListUsersOptions{Username: gitlab.String(username)})
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

func getUserGit(msg *discordgo.Message) (*gitlab.Client, error) {
	asTok, exists := associatedKey(msg.Author.ID)
	if exists == false {
		return nil, errNoPAC
	}
	git := gitlab.NewClient(nil, asTok)
	return git, nil
}
